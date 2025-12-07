package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	bytesToFetch = 1024
	timeout      = 30 * time.Second
)

// --- Styles ---

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	current   = lipgloss.AdaptiveColor{Light: "#FFAA00", Dark: "#FFD700"}

	// Card Styles
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			MarginRight(1).
			Width(12).
			Align(lipgloss.Center)

	winOneStyle = cardStyle.Copy().
			BorderForeground(special).
			Foreground(special)

	winZeroStyle = cardStyle.Copy().
			BorderForeground(highlight).
			Foreground(highlight)

	tieStyle = cardStyle.Copy().
			BorderForeground(subtle).
			Foreground(subtle)

	currentOneStyle = cardStyle.Copy().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(current).
			Foreground(special)

	currentZeroStyle = cardStyle.Copy().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(current).
				Foreground(highlight)

	currentTieStyle = cardStyle.Copy().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(current).
			Foreground(subtle)

	// Diamond border for current result
	diamondOneStyle = cardStyle.Copy().
			Border(lipgloss.Border{
			Top:         "◆",
			Bottom:      "◆",
			Left:        "◆",
			Right:       "◆",
			TopLeft:     "◆",
			TopRight:    "◆",
			BottomLeft:  "◆",
			BottomRight: "◆",
		}).
		BorderForeground(current).
		Foreground(special)

	diamondZeroStyle = cardStyle.Copy().
				Border(lipgloss.Border{
			Top:         "◆",
			Bottom:      "◆",
			Left:        "◆",
			Right:       "◆",
			TopLeft:     "◆",
			TopRight:    "◆",
			BottomLeft:  "◆",
			BottomRight: "◆",
		}).
		BorderForeground(current).
		Foreground(highlight)

	diamondTieStyle = cardStyle.Copy().
			Border(lipgloss.Border{
			Top:         "◆",
			Bottom:      "◆",
			Left:        "◆",
			Right:       "◆",
			TopLeft:     "◆",
			TopRight:    "◆",
			BottomLeft:  "◆",
			BottomRight: "◆",
		}).
		BorderForeground(current).
		Foreground(subtle)

	// Text Styles
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})
)

// --- Structs for API ---

type QRandomResponse struct {
	BinaryURL string `json:"binaryURL"`
}

type AnuQrngResponse struct {
	Data    []uint8 `json:"data"`
	Success bool    `json:"success"`
}

// --- Bubble Tea Model & Messages ---

type resultType int

const (
	resOnes resultType = iota
	resZeros
	resTie
)

type flipResult struct {
	ones   int
	zeros  int
	winner resultType
}

type model struct {
	source  string
	results []flipResult
	loading bool
	err     error
	width   int // Fixed: was 'wid'
	height  int // Fixed: was 'heigh'
}

type flipMsg struct {
	result flipResult
	err    error
}

func initialModel(source string) model {
	return model{
		source:  source,
		results: []flipResult{},
		loading: false,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			if !m.loading {
				m.loading = true
				m.err = nil
				return m, fetchAndFlipCmd(m.source)
			}
		case "r":
			m.results = []flipResult{}
			return m, nil
		case "c":
			// Toggle between sources
			if m.source == "qr" {
				m.source = "anu"
			} else {
				m.source = "qr"
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case flipMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.results = append(m.results, msg.result)
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "loading..."
	}

	// 1. Header
	header := titleStyle.Render("QCOIN - Quantum Flip")

	// 2. The Carousel (Horizontal Scroll)
	// Calculate how many cards fit in the width with buffer
	cardWidth := 14               // 12 width + 1 margin + borders usually add up
	availableWidth := m.width - 8 // Leave 4 chars buffer on each side
	maxCards := availableWidth / cardWidth
	if maxCards < 1 {
		maxCards = 1
	}

	visibleResults := m.results
	// If we have more results than fit, slice to show the most recent ones
	if len(m.results) > maxCards {
		visibleResults = m.results[len(m.results)-maxCards:]
	}

	var cards []string
	for i, res := range visibleResults {
		label := ""
		style := tieStyle

		// Check if this is the most recent result
		isLatest := (i == len(visibleResults)-1) && len(m.results) > 0

		switch res.winner {
		case resOnes:
			label = "ONES"
			if isLatest {
				style = currentOneStyle
			} else {
				style = winOneStyle
			}
		case resZeros:
			label = "ZEROS"
			if isLatest {
				style = currentZeroStyle
			} else {
				style = winZeroStyle
			}
		default:
			label = "TIE"
			if isLatest {
				style = currentTieStyle
			} else {
				style = tieStyle
			}
		}

		content := fmt.Sprintf("%s\n\n1: %d\n0: %d", label, res.ones, res.zeros)
		cards = append(cards, style.Render(content))
	}

	carousel := lipgloss.JoinHorizontal(lipgloss.Top, cards...)

	// Center the carousel with padding
	carousel = lipgloss.NewStyle().
		PaddingLeft(4).
		Render(carousel)

	// If empty history
	if len(m.results) == 0 {
		carousel = lipgloss.NewStyle().
			Height(6).
			Width(m.width).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(subtle).
			Render("No flips yet. Spin the quantum coin!")
	}

	// 3. Status Bar
	var status string
	if m.err != nil {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("Error: %v", m.err))
	} else if m.loading {
		status = "Extracting entropy..."
	} else {
		status = fmt.Sprintf("Source: %s | Total Flips: %d", strings.ToUpper(m.source), len(m.results))
	}

	help := statusStyle.Render("\nPress [Enter] to Flip • [r] to Reset • [c] to Change Source • [q] to Quit")

	// Layout Composition
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"\n",
		carousel,
		"\n",
		status,
		help,
	)
}

// --- Commands ---

func fetchAndFlipCmd(source string) tea.Cmd {
	return func() tea.Msg {
		bytes, err := fetchRandomBytes(source)
		if err != nil {
			return flipMsg{err: err}
		}

		ones, zeros := countBits(bytes)

		res := flipResult{
			ones:   ones,
			zeros:  zeros,
			winner: resTie,
		}

		if ones > zeros {
			res.winner = resOnes
		} else if zeros > ones {
			res.winner = resZeros
		}

		return flipMsg{result: res}
	}
}

// --- Main ---

func main() {
	source := flag.String("s", "qr", "Source: qr (qrandom.io) or anu (ANU QRNG)")
	interactive := flag.Bool("i", false, "Start interactive TUI mode")
	flag.Parse()

	if *interactive {
		p := tea.NewProgram(initialModel(*source), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	} else {
		// Standard CLI Mode
		runCLI(*source)
	}
}

// --- Existing Logic (Refactored slightly for reuse) ---

func runCLI(source string) {
	bytes, err := fetchRandomBytes(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ones, zeros := countBits(bytes)

	fmt.Printf("Ones: %d\n", ones)
	fmt.Printf("Zeros: %d\n", zeros)

	if ones > zeros {
		fmt.Println("Result: ONES")
	} else if zeros > ones {
		fmt.Println("Result: ZEROS")
	} else {
		fmt.Println("Result: TIE")
	}
}

func countBits(bytes []byte) (int, int) {
	ones := 0
	zeros := 0

	for _, b := range bytes {
		for i := 0; i < 8; i++ {
			if (b & (1 << i)) != 0 {
				ones++
			} else {
				zeros++
			}
		}
	}

	return ones, zeros
}

func fetchRandomBytes(source string) ([]byte, error) {
	switch source {
	case "qr":
		return fetchQRandomBytes()
	case "anu":
		return fetchAnuQrngBytes()
	default:
		return nil, fmt.Errorf("unknown source: %s (use 'qr' or 'anu')", source)
	}
}

func fetchQRandomBytes() ([]byte, error) {
	client := &http.Client{Timeout: timeout}

	url := fmt.Sprintf("https://qrandom.io/api/random/binary?bytes=%d", bytesToFetch)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("qrandom.io request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qrandom.io returned status %d", resp.StatusCode)
	}

	var qrResp QRandomResponse
	if err := json.NewDecoder(resp.Body).Decode(&qrResp); err != nil {
		return nil, fmt.Errorf("failed to parse qrandom.io response: %w", err)
	}

	binaryResp, err := client.Get(qrResp.BinaryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch binary data: %w", err)
	}
	defer binaryResp.Body.Close()

	if binaryResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binary fetch returned status %d", binaryResp.StatusCode)
	}

	bytes, err := io.ReadAll(binaryResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary data: %w", err)
	}

	return bytes, nil
}

func fetchAnuQrngBytes() ([]byte, error) {
	client := &http.Client{Timeout: timeout}

	url := fmt.Sprintf("https://qrng.anu.edu.au/API/jsonI.php?length=%d&type=uint8", bytesToFetch)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ANU QRNG request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ANU QRNG returned status %d", resp.StatusCode)
	}

	var anuResp AnuQrngResponse
	if err := json.NewDecoder(resp.Body).Decode(&anuResp); err != nil {
		return nil, fmt.Errorf("failed to parse ANU QRNG response: %w", err)
	}

	if !anuResp.Success {
		return nil, fmt.Errorf("ANU QRNG API returned success=false")
	}

	if len(anuResp.Data) != bytesToFetch {
		return nil, fmt.Errorf("expected %d bytes, got %d", bytesToFetch, len(anuResp.Data))
	}

	return anuResp.Data, nil
}

// These are kept for potential future use or fallback
func fetchCryptoRandomBytes() ([]byte, error) {
	bytes := make([]byte, bytesToFetch)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("crypto random failed: %w", err)
	}
	return bytes, nil
}

func saveToFile(bytes []byte, filename string) error {
	hexString := hex.EncodeToString(bytes)
	return os.WriteFile(filename, []byte(hexString), 0644)
}

func loadFromFile(filename string) ([]byte, error) {
	hexString, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(string(hexString))
}
