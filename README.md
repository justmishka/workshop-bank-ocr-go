# Bank OCR (Go)

Workshop Product 01 — **Go rebuild** of the original Python version.

**Source kata:** [codingdojo.org/kata/BankOCR](https://codingdojo.org/kata/BankOCR/)
**Original Python repo:** [justmishka/workshop-bank-ocr](https://github.com/justmishka/workshop-bank-ocr)

## What it does

A bank's scanning machine produces files with account numbers written in ASCII art using pipes and underscores. This tool:

1. Parses OCR output into readable account numbers
2. Validates numbers using a checksum algorithm
3. Generates output with validation status (valid / ERR / ILL)
4. Attempts error correction for invalid or illegible numbers
5. Provides a web UI for paste/upload + visualization

## Requirements

- Go 1.22+
- No external dependencies (stdlib only)

## Usage

```bash
# CLI
go run ./cmd/bank-ocr samples/sample.txt

# Web UI
go run ./cmd/bank-ocr -web
# → http://localhost:8080
```

## Tests

```bash
go test ./...
go test -cover ./...
```

## Notable difference vs the Python reference

The Python CLI and web pipeline ship without ever invoking the corrector — Story 4 lives there as a tested library function but does not affect end-user output. This Go rebuild **wires Story 4 into the pipeline**, so:

- Invalid-checksum accounts that have exactly one valid single-character correction are corrected (e.g. `111111111 ERR` → `711111111`).
- Accounts with multiple valid corrections are marked `AMB` and the alternatives are listed.
- The JSON API grows an additive `alternatives` field (empty array for non-AMB results).

This is an intentional divergence — the kata acceptance criteria call for the correction to affect output, and the Go rebuild honors that.

## Project

- **Team:** The AI Dev Team
- **Format:** AI Coding Dojo — built in parallel by the team agents (Finn × 4 + Mia for UI), reviewed by Nova + Sage + Dex
- **Kick-off notes:** [kick-off.md](kick-off.md)
