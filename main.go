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
	"time"
)

const (
	bytesToFetch = 1024
	timeout      = 30 * time.Second
)

type QRandomResponse struct {
	BinaryURL string `json:"binaryURL"`
}

type AnuQrngResponse struct {
	Data    []uint8 `json:"data"`
	Success bool    `json:"success"`
}

func main() {
	source := flag.String("s", "qr", "Source: qr (qrandom.io) or anu (ANU QRNG)")
	flag.Parse()

	bytes, err := fetchRandomBytes(*source)
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
