// Package parser converts ASCII-art OCR account entries into account-number
// strings. A single digit is encoded in a 3x3 character block, and a full
// entry is three 27-character lines yielding a 9-character account number.
//
// Unrecognized 3x3 patterns become '?'. The parser performs no checksum
// validation — that is the checksum package's job.
package parser

import (
	"errors"
	"fmt"
	"strings"
)

// ErrShortEntry is returned by ParseEntry when fewer than 3 lines are supplied.
var ErrShortEntry = errors.New("entry must have at least 3 lines")

// EntryWidth is the canonical width of an OCR entry line (9 digits × 3 chars).
//
// Exported so other packages (corrector, tests, fixtures) can pad / slice
// against the same constant instead of redeclaring it and risking drift.
const EntryWidth = 27

// DigitPatterns maps a 9-character concatenation of (top + mid + bot) to its
// digit rune. This is the canonical OCR alphabet for the whole product —
// every package that needs to recognise a glyph depends on this single
// table. Exported deliberately so the corrector (and any future glyph
// consumer) can share the same source of truth without copy-paste drift.
var DigitPatterns = map[string]rune{
	" _ " + "| |" + "|_|": '0',
	"   " + "  |" + "  |": '1',
	" _ " + " _|" + "|_ ": '2',
	" _ " + " _|" + " _|": '3',
	"   " + "|_|" + "  |": '4',
	" _ " + "|_ " + " _|": '5',
	" _ " + "|_ " + "|_|": '6',
	" _ " + "  |" + "  |": '7',
	" _ " + "|_|" + "|_|": '8',
	" _ " + "|_|" + " _|": '9',
}

// ParseDigit converts a single 3x3 OCR block (three 3-char strings) into a
// digit rune. If the pattern is not one of the ten known digits, it returns
// the rune '?'.
func ParseDigit(top, mid, bot string) rune {
	pattern := top + mid + bot
	if r, ok := DigitPatterns[pattern]; ok {
		return r
	}
	return '?'
}

// ParseEntry parses a 3-line OCR entry into a 9-character account-number
// string. Lines shorter than 27 characters are right-padded with spaces to
// match the Python reference's ljust(27) behaviour. Lines longer than 27
// characters are tolerated — only the first 27 characters are inspected.
//
// Returns ErrShortEntry wrapped with context if fewer than 3 lines are given.
func ParseEntry(lines []string) (string, error) {
	if len(lines) < 3 {
		return "", fmt.Errorf("%w: got %d", ErrShortEntry, len(lines))
	}

	top := padRight(lines[0], EntryWidth)
	mid := padRight(lines[1], EntryWidth)
	bot := padRight(lines[2], EntryWidth)

	var b strings.Builder
	b.Grow(9)
	for i := 0; i < 9; i++ {
		start := i * 3
		end := start + 3
		b.WriteRune(ParseDigit(top[start:end], mid[start:end], bot[start:end]))
	}
	return b.String(), nil
}

// ParseFile parses a full OCR file into a slice of 9-character account
// numbers. Each entry occupies 4 lines: 3 digit lines plus 1 blank separator
// line. We use strict 4-line grouping — never blank-line detection — because
// digit "1" has an all-spaces first line that would otherwise be
// indistinguishable from a separator.
//
// A trailing newline (which produces a trailing empty element after splitting)
// is tolerated. An empty or whitespace-only input returns an empty slice.
func ParseFile(content string) []string {
	entries := ParseEntries(content)
	accounts := make([]string, len(entries))
	for i, e := range entries {
		accounts[i] = e.Number
	}
	return accounts
}

// Entry is one parsed OCR entry carrying both the decoded account number and
// the three raw OCR lines that produced it. The Lines are right-padded to
// EntryWidth before being stored so consumers (the corrector, in particular)
// can index them by digit position without re-padding.
type Entry struct {
	// Number is the 9-character account string ('?' for unrecognised digits).
	Number string
	// Lines is the three OCR lines (top, mid, bot), each padded to EntryWidth.
	Lines [3]string
}

// ParseEntries parses a full OCR file and returns one Entry per account,
// preserving the raw OCR lines alongside the decoded number. This is the
// richer counterpart to ParseFile and is what the correction pipeline uses
// to map an ERR/ILL account back to the 3×3 glyph grid that produced it.
//
// Grouping rules and trailing-newline tolerance match ParseFile.
func ParseEntries(content string) []Entry {
	if strings.TrimSpace(content) == "" {
		return []Entry{}
	}

	lines := strings.Split(content, "\n")
	// Drop trailing empty lines so a trailing newline doesn't trick the loop.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	entries := make([]Entry, 0, len(lines)/4+1)
	for i := 0; i+2 < len(lines); i += 4 {
		group := lines[i : i+3]
		number, err := ParseEntry(group)
		if err != nil {
			// Defensive: a 3-line slice cannot trigger ErrShortEntry.
			continue
		}
		entries = append(entries, Entry{
			Number: number,
			Lines: [3]string{
				padRight(group[0], EntryWidth),
				padRight(group[1], EntryWidth),
				padRight(group[2], EntryWidth),
			},
		})
	}
	return entries
}

// LineError describes a validation problem found in OCR input. LineNumber is
// 1-indexed; a value of 0 indicates the error is not tied to a specific line.
type LineError struct {
	Message    string
	LineNumber int
}

// Error implements the error interface.
func (e *LineError) Error() string {
	if e.LineNumber > 0 {
		return fmt.Sprintf("line %d: %s", e.LineNumber, e.Message)
	}
	return e.Message
}

// ValidateOCRInput inspects OCR input and returns a slice of errors describing
// any problems found. The slice is nil/empty when the input is valid.
//
// Checks performed:
//   - Empty / whitespace-only input
//   - Fewer than 3 lines after stripping trailing blanks
//   - Invalid characters on digit lines (anything other than space, pipe,
//     underscore). Separator lines (every 4th line, 0-indexed line 3) are
//     skipped — they may legitimately be blank.
func ValidateOCRInput(content string) []error {
	var errs []error

	if strings.TrimSpace(content) == "" {
		errs = append(errs, &LineError{Message: "no OCR input provided"})
		return errs
	}

	lines := strings.Split(content, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) < 3 {
		errs = append(errs, &LineError{
			Message: fmt.Sprintf(
				"input too short — need at least 3 lines for one entry, got %d",
				len(lines),
			),
		})
		return errs
	}

	for i, line := range lines {
		// Skip the 4th line of each entry — it's the separator and may be blank.
		if i%4 == 3 {
			continue
		}
		if bad := invalidChars(line); bad != "" {
			errs = append(errs, &LineError{
				Message:    fmt.Sprintf("invalid characters found: %q", bad),
				LineNumber: i + 1,
			})
		}
	}

	return errs
}

// padRight returns s padded on the right with spaces up to width characters.
// Strings already at or above width are returned unchanged. Operates on bytes
// because OCR characters are pure ASCII.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// invalidChars returns a string containing each unique invalid character (in
// order of first occurrence) found in s. The valid set is {' ', '|', '_'}.
// Returns "" when every character is valid.
func invalidChars(s string) string {
	var bad strings.Builder
	seen := make(map[rune]struct{})
	for _, r := range s {
		switch r {
		case ' ', '|', '_':
			continue
		}
		if _, dup := seen[r]; dup {
			continue
		}
		seen[r] = struct{}{}
		bad.WriteRune(r)
	}
	return bad.String()
}
