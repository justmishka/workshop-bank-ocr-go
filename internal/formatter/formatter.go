// Package formatter is the top-level pipeline that turns raw OCR file content
// into status-annotated output. It composes parser → checksum → corrector
// (Story 4) into a single ProcessAll function and offers a CLI-friendly
// FormatOutput wrapper for one-line-per-account text.
//
// Output format (one account per line):
//
//	"123456789"                       — valid account, no marker
//	"664371495 ERR"                   — well-formed but invalid checksum
//	"86110??36 ILL"                   — contains at least one illegible ('?') digit
//	"711111111"                       — was ERR / ILL but corrected to a unique valid number
//	"888888888 AMB ['888886888', …]"  — multiple valid corrections; original unchanged
//
// The corrector runs only on accounts that need it (Status ERR or ILL after
// classification), so valid accounts pass through unchanged at zero extra
// cost. This is the wiring the Python reference shipped without — the Go
// rebuild closes that gap so Story 4 affects end-user output.
package formatter

import (
	"strings"

	"github.com/justmishka/workshop-bank-ocr-go/internal/checksum"
	"github.com/justmishka/workshop-bank-ocr-go/internal/corrector"
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

// ProcessAll is the full Story 1-4 pipeline: parse the OCR content, classify
// each account, and attempt single-character correction for accounts that
// land in ERR or ILL. Valid accounts pass through unchanged.
//
// The returned slice has one Account per input entry, in input order. Empty
// input returns an empty slice.
func ProcessAll(ocrContent string) []types.Account {
	entries := parser.ParseEntries(ocrContent)
	if len(entries) == 0 {
		return []types.Account{}
	}

	out := make([]types.Account, len(entries))
	for i, e := range entries {
		base := ClassifyToAccount(e.Number)
		if base.Status == types.StatusOK {
			out[i] = base
			continue
		}
		// ERR or ILL: ask the corrector. It returns OK with a replacement
		// Number if exactly one valid correction exists, AMB with sorted
		// alternatives if more than one, or the original ERR/ILL if none.
		out[i] = corrector.CorrectAccount(e.Number, e.Lines[:])
	}
	return out
}

// FormatOutput is the CLI-facing wrapper: ProcessAll + one line per account
// joined with '\n'. Empty input produces an empty string (no trailing
// newline).
func FormatOutput(ocrContent string) string {
	accounts := ProcessAll(ocrContent)
	if len(accounts) == 0 {
		return ""
	}

	lines := make([]string, len(accounts))
	for i, acc := range accounts {
		lines[i] = formatLine(acc)
	}
	return strings.Join(lines, "\n")
}

// formatLine renders one Account as a single output line.
//
//	OK  → just the number
//	ERR → "<number> ERR"
//	ILL → "<number> ILL"
//	AMB → "<number> AMB ['alt1', 'alt2', …]" with single-quoted, sorted alts
func formatLine(a types.Account) string {
	switch a.Status {
	case types.StatusOK:
		return a.Number
	case types.StatusAMB:
		quoted := make([]string, len(a.Alternatives))
		for i, alt := range a.Alternatives {
			quoted[i] = "'" + alt + "'"
		}
		return a.Number + " AMB [" + strings.Join(quoted, ", ") + "]"
	default: // ERR, ILL, or anything unexpected — use the status marker.
		return a.Number + " " + a.Status.String()
	}
}
