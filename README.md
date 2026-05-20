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

## Project

- **Team:** The AI Dev Team
- **Format:** AI Coding Dojo — built in parallel by the team agents (Finn × 4 + Mia for UI)
- **Kick-off notes:** [kick-off.md](kick-off.md)
