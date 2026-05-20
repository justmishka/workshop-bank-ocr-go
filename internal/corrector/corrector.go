// Package corrector attempts to repair ERR (bad checksum) and ILL
// (illegible) account numbers by exploring single-character OCR
// modifications.
//
// For each of the 9 digit positions, the corrector takes the underlying
// 3x3 OCR pattern and generates every variant that differs by exactly one
// character (pipe/underscore turned into a space, or a space turned into
// pipe or underscore). Variants that resolve to a known digit are
// substituted into the account number; if the resulting account has no
// '?' and a valid checksum, it is recorded as a candidate alternative.
//
// The number of valid alternatives determines the final status:
//
//   - 1 alternative  → StatusOK, account replaced with the correction
//   - >1 alternatives → StatusAMB, alternatives returned sorted
//   - 0 alternatives  → original status preserved (StatusILL if the input
//     contained '?', otherwise StatusERR)
//
// The algorithm is a direct port of src/corrector.py from the Python
// reference implementation.
package corrector

import (
	"sort"
	"strings"

	"workshop-bank-ocr-go/internal/checksum"
	"workshop-bank-ocr-go/internal/parser"
	"workshop-bank-ocr-go/internal/types"
)

// CorrectAccount attempts to repair an ERR or ILL account number.
//
// The account argument is the 9-character account string already produced by
// the parser (containing digits 0-9 and possibly '?'). entryLines is the
// 3-line OCR entry that produced it; lines shorter than 27 characters are
// right-padded with spaces, matching the Python reference's ljust(27).
//
// The returned Account always has its Number set; Status is one of OK, ERR,
// ILL, or AMB. Alternatives is populated (sorted) only when Status is AMB.
//
// CorrectAccount is safe to call on already-valid accounts — if no
// single-character modification produces a different valid account, the
// original is returned with the appropriate status.
func CorrectAccount(account string, entryLines []string) types.Account {
	// Defensive: if we weren't given 3 lines we cannot inspect any patterns,
	// so report the input as-is.
	if len(entryLines) < 3 {
		return types.Account{
			Number: account,
			Status: statusFromAccount(account),
		}
	}

	top := padRight(entryLines[0], parser.EntryWidth)
	mid := padRight(entryLines[1], parser.EntryWidth)
	bot := padRight(entryLines[2], parser.EntryWidth)

	// Track alternatives in insertion order via a slice + dedup set so the
	// final sort.Strings is deterministic regardless of map iteration order.
	var alternatives []string
	seen := make(map[string]struct{})

	for pos := 0; pos < 9; pos++ {
		start := pos * 3
		end := start + 3
		originalPattern := top[start:end] + mid[start:end] + bot[start:end]

		for _, variant := range generateVariants(originalPattern) {
			digit, ok := parser.DigitPatterns[variant]
			if !ok {
				continue
			}

			candidate := substituteDigit(account, pos, digit)
			if strings.ContainsRune(candidate, '?') {
				continue
			}
			if candidate == account {
				continue
			}
			if _, dup := seen[candidate]; dup {
				continue
			}

			valid, known := checksum.IsValid(candidate)
			if !known || !valid {
				continue
			}

			seen[candidate] = struct{}{}
			alternatives = append(alternatives, candidate)
		}
	}

	switch len(alternatives) {
	case 0:
		return types.Account{
			Number: account,
			Status: statusFromAccount(account),
		}
	case 1:
		return types.Account{
			Number: alternatives[0],
			Status: types.StatusOK,
		}
	default:
		sort.Strings(alternatives)
		return types.Account{
			Number:       account,
			Status:       types.StatusAMB,
			Alternatives: alternatives,
		}
	}
}

// generateVariants returns every single-character modification of a 9-char
// OCR pattern. Pipes and underscores can each be replaced with a space; each
// space can be replaced with either a pipe or an underscore.
//
// The returned slice may contain duplicates if the input did — callers should
// not rely on uniqueness, though the well-formed digit patterns we work with
// in practice do not produce any.
func generateVariants(pattern string) []string {
	variants := make([]string, 0, len(pattern)*2)
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '|', '_':
			variants = append(variants, replaceAt(pattern, i, ' '))
		case ' ':
			variants = append(variants, replaceAt(pattern, i, '|'))
			variants = append(variants, replaceAt(pattern, i, '_'))
		}
	}
	return variants
}

// replaceAt returns s with the byte at index i replaced by c. Safe because
// OCR patterns are pure ASCII.
func replaceAt(s string, i int, c byte) string {
	b := []byte(s)
	b[i] = c
	return string(b)
}

// substituteDigit returns account with the rune at position pos replaced by
// digit. Operates on bytes because account strings are pure ASCII.
func substituteDigit(account string, pos int, digit rune) string {
	if pos < 0 || pos >= len(account) {
		return account
	}
	b := []byte(account)
	b[pos] = byte(digit)
	return string(b)
}

// statusFromAccount returns the natural status of an account when no
// alternatives were found: ILL if it contains '?', otherwise ERR. This is
// intentionally narrow — CorrectAccount only calls it on accounts that have
// already failed checksum or contain '?'.
func statusFromAccount(account string) types.Status {
	if strings.ContainsRune(account, '?') {
		return types.StatusILL
	}
	return types.StatusERR
}

// padRight returns s padded on the right with spaces up to width characters.
// Strings already at or above width are returned unchanged.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
