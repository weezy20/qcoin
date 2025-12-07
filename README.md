# qcoin

A quantum coin toss application that sources 1 KB of bytes from one of the quantum rng sources below & counts 1s & 0s in them to produce a quantum coin toss result. Possible values are `ONES` (1s > 0s), `ZEROS` (0s > 1s) and `TIE` (0s == 1s).

## Install
```sh
# requires go 1.25.3
go install github.com/weezy20/qcoin@latest
```

## Run 

Normal coin toss:

```sh
qcoin
```

Use alternate quantum sources
```sh
# Using ANU.org:
qcoin -s anu # Note: this is rate limited to 1 request per minute
# Using qrandom.io, this is the default so no need for -s qr
qcoin -s qr
```

Or run using TUI for multiple runs, interactive quantum source changes. 

```sh
qcoin -i
```
