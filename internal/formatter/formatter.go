// Package formatter combines parser + checksum into status-annotated output.
//
// It is the top-level pipeline for User Story 3: take raw OCR file content,
// parse it into account strings, validate each, and emit one line per account
// with an optional status marker.
//
// Output format (one account per line):
//
//	"123456789"     — valid account, no marker
//	"664371495 ERR" — well-formed but invalid checksum
//	"86110??36 ILL" — contains at least one illegible ('?') digit
package formatter

import (
	"strings"

	"github.com/justmishka/workshop-bank-ocr-go/internal/checksum"
	"github.com/justmishka/workshop-bank-ocr-go/internal/parser"
	"github.com/justmishka/workshop-bank-ocr-go/internal/types"
)

// ClassifyAccount classifies a single account number and returns the formatted
// output line.
//
// Rules (checked in this order):
//   - account contains '?'        → "<account> ILL"
//   - checksum invalid             → "<account> ERR"
//   - otherwise (valid checksum)   → "<account>"
func ClassifyAccount(account string) string {
	if strings.ContainsRune(account, '?') {
		return account + " ILL"
	}

	valid, _ := checksum.IsValid(account)
	if !valid {
		return account + " ERR"
	}
	return account
}

// ClassifyToAccount classifies a single account number and returns a fully
// populated types.Account. Alternatives is left empty — corrections are the
// corrector package's responsibility, not the formatter's.
//
// Useful for callers (e.g. the web package) that want structured data rather
// than a pre-formatted line.
func ClassifyToAccount(number string) types.Account {
	if strings.ContainsRune(number, '?') {
		return types.Account{Number: number, Status: types.StatusILL}
	}

	valid, _ := checksum.IsValid(number)
	if !valid {
		return types.Account{Number: number, Status: types.StatusERR}
	}
	return types.Account{Number: number, Status: types.StatusOK}
}

// FormatOutput parses an OCR file's content and produces formatted output for
// every account it contains, one per line, joined with '\n'.
//
// Empty input produces an empty string (no trailing newline).
func FormatOutput(ocrContent string) string {
	accounts := parser.ParseFile(ocrContent)
	if len(accounts) == 0 {
		return ""
	}

	lines := make([]string, len(accounts))
	for i, account := range accounts {
		lines[i] = ClassifyAccount(account)
	}
	return strings.Join(lines, "\n")
}
