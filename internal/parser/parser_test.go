package parser

import (
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ParseDigit
// ---------------------------------------------------------------------------

func TestParseDigit(t *testing.T) {
	tests := []struct {
		name     string
		top      string
		mid      string
		bot      string
		expected rune
	}{
		{"zero", " _ ", "| |", "|_|", '0'},
		{"one", "   ", "  |", "  |", '1'},
		{"two", " _ ", " _|", "|_ ", '2'},
		{"three", " _ ", " _|", " _|", '3'},
		{"four", "   ", "|_|", "  |", '4'},
		{"five", " _ ", "|_ ", " _|", '5'},
		{"six", " _ ", "|_ ", "|_|", '6'},
		{"seven", " _ ", "  |", "  |", '7'},
		{"eight", " _ ", "|_|", "|_|", '8'},
		{"nine", " _ ", "|_|", " _|", '9'},
		{"all_spaces_returns_question_mark", "   ", "   ", "   ", '?'},
		{"garbage_returns_question_mark", "XXX", "YYY", "ZZZ", '?'},
		{"partial_match_returns_question_mark", " _ ", "| |", "|_ ", '?'},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseDigit(tc.top, tc.mid, tc.bot)
			if got != tc.expected {
				t.Errorf("ParseDigit(%q,%q,%q) = %q; want %q",
					tc.top, tc.mid, tc.bot, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParseEntry
// ---------------------------------------------------------------------------

func TestParseEntry(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected string
	}{
		{
			name: "all_zeros",
			lines: []string{
				" _  _  _  _  _  _  _  _  _ ",
				"| || || || || || || || || |",
				"|_||_||_||_||_||_||_||_||_|",
			},
			expected: "000000000",
		},
		{
			name: "all_ones",
			lines: []string{
				"                           ",
				"  |  |  |  |  |  |  |  |  |",
				"  |  |  |  |  |  |  |  |  |",
			},
			expected: "111111111",
		},
		{
			name: "all_twos",
			lines: []string{
				" _  _  _  _  _  _  _  _  _ ",
				" _| _| _| _| _| _| _| _| _|",
				"|_ |_ |_ |_ |_ |_ |_ |_ |_ ",
			},
			expected: "222222222",
		},
		{
			name: "all_threes",
			lines: []string{
				" _  _  _  _  _  _  _  _  _ ",
				" _| _| _| _| _| _| _| _| _|",
				" _| _| _| _| _| _| _| _| _|",
			},
			expected: "333333333",
		},
		{
			name: "all_fours",
			lines: []string{
				"                           ",
				"|_||_||_||_||_||_||_||_||_|",
				"  |  |  |  |  |  |  |  |  |",
			},
			expected: "444444444",
		},
		{
			name: "all_fives",
			lines: []string{
				" _  _  _  _  _  _  _  _  _ ",
				"|_ |_ |_ |_ |_ |_ |_ |_ |_ ",
				" _| _| _| _| _| _| _| _| _|",
			},
			expected: "555555555",
		},
		{
			name: "sequence_123456789",
			lines: []string{
				"    _  _     _  _  _  _  _ ",
				"  | _| _||_||_ |_   ||_||_|",
				"  ||_  _|  | _||_|  ||_| _|",
			},
			expected: "123456789",
		},
		{
			name: "short_lines_padded_to_27",
			lines: []string{
				" _ ",
				"| |",
				"|_|",
			},
			// First block is "0", remaining 8 blocks become "?" because
			// padding produces all-space 3x3 blocks.
			expected: "0????????",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseEntry(tc.lines)
			if err != nil {
				t.Fatalf("ParseEntry returned unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("ParseEntry = %q; want %q", got, tc.expected)
			}
		})
	}
}

func TestParseEntry_IllegibleDigit(t *testing.T) {
	// Corrupt the first digit of the all-zeros entry.
	lines := []string{
		"___ _  _  _  _  _  _  _  _ ",
		"| || || || || || || || || |",
		"|_||_||_||_||_||_||_||_||_|",
	}
	got, err := ParseEntry(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0] != '?' {
		t.Errorf("first digit = %q; want '?'", got[0])
	}
	if got[1:] != "00000000" {
		t.Errorf("remaining digits = %q; want %q", got[1:], "00000000")
	}
}

func TestParseEntry_MixedIllegibleAndValid(t *testing.T) {
	// Build a known-good middle row of all-zeros, then corrupt digit index 4
	// (byte 13) to produce an unrecognisable middle digit.
	mid := []byte("| || || || || || || || || |")
	mid[13] = 'X'
	lines := []string{
		" _  _  _  _  _  _  _  _  _ ",
		string(mid),
		"|_||_||_||_||_||_||_||_||_|",
	}

	got, err := ParseEntry(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[4] != '?' {
		t.Errorf("digit[4] = %q; want '?'; full result = %q", got[4], got)
	}
	if got[0] != '0' {
		t.Errorf("digit[0] = %q; want '0'", got[0])
	}
}

func TestParseEntry_TooFewLinesReturnsError(t *testing.T) {
	_, err := ParseEntry([]string{"   ", "  |"})
	if err == nil {
		t.Fatal("expected error for 2-line entry, got nil")
	}
	if !errors.Is(err, ErrShortEntry) {
		t.Errorf("error = %v; want errors.Is(err, ErrShortEntry)", err)
	}
}

func TestParseEntry_EmptyLinesReturnsError(t *testing.T) {
	_, err := ParseEntry(nil)
	if !errors.Is(err, ErrShortEntry) {
		t.Errorf("error = %v; want ErrShortEntry", err)
	}
}

// ---------------------------------------------------------------------------
// ParseFile
// ---------------------------------------------------------------------------

func TestParseFile(t *testing.T) {
	zeros := " _  _  _  _  _  _  _  _  _ \n" +
		"| || || || || || || || || |\n" +
		"|_||_||_||_||_||_||_||_||_|\n"
	ones := "                           \n" +
		"  |  |  |  |  |  |  |  |  |\n" +
		"  |  |  |  |  |  |  |  |  |\n"

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty_string",
			content:  "",
			expected: []string{},
		},
		{
			name:     "whitespace_only",
			content:  "   \n\n",
			expected: []string{},
		},
		{
			name:     "single_entry_with_separator",
			content:  zeros + "\n",
			expected: []string{"000000000"},
		},
		{
			name:     "single_entry_no_trailing_separator",
			content:  zeros,
			expected: []string{"000000000"},
		},
		{
			name:     "two_entries",
			content:  zeros + "\n" + ones + "\n",
			expected: []string{"000000000", "111111111"},
		},
		{
			name: "ones_entry_does_not_break_grouping",
			// Digit "1" has an all-spaces first line; strict 4-line grouping
			// must still produce one entry here, not split mid-entry.
			content:  ones + "\n",
			expected: []string{"111111111"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseFile(tc.content)
			if !equalStringSlices(got, tc.expected) {
				t.Errorf("ParseFile = %v; want %v", got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateOCRInput
// ---------------------------------------------------------------------------

func TestValidateOCRInput(t *testing.T) {
	validEntry := " _  _  _  _  _  _  _  _  _ \n" +
		"| || || || || || || || || |\n" +
		"|_||_||_||_||_||_||_||_||_|\n"

	tests := []struct {
		name        string
		content     string
		wantErrs    int
		wantSubstr  string // substring expected in the first error, if any
		wantLineNum int    // 0 = don't check
	}{
		{
			name:       "empty_input",
			content:    "",
			wantErrs:   1,
			wantSubstr: "no OCR input",
		},
		{
			name:       "whitespace_only_input",
			content:    "   \n  \n",
			wantErrs:   1,
			wantSubstr: "no OCR input",
		},
		{
			name:       "too_few_lines",
			content:    " _ \n| |\n",
			wantErrs:   1,
			wantSubstr: "too short",
		},
		{
			name:     "valid_single_entry",
			content:  validEntry,
			wantErrs: 0,
		},
		{
			name: "invalid_char_on_digit_line",
			content: " _ X _  _  _  _  _  _  _  _ \n" +
				"| || || || || || || || || |\n" +
				"|_||_||_||_||_||_||_||_||_|\n",
			wantErrs:    1,
			wantSubstr:  "invalid characters",
			wantLineNum: 1,
		},
		{
			name:     "separator_line_can_be_blank",
			content:  validEntry + "\n" + validEntry,
			wantErrs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateOCRInput(tc.content)
			if len(errs) != tc.wantErrs {
				t.Fatalf("got %d errors, want %d: %v", len(errs), tc.wantErrs, errs)
			}
			if tc.wantErrs == 0 {
				return
			}
			msg := errs[0].Error()
			if tc.wantSubstr != "" && !strings.Contains(msg, tc.wantSubstr) {
				t.Errorf("error %q does not contain %q", msg, tc.wantSubstr)
			}
			if tc.wantLineNum > 0 {
				le, ok := errs[0].(*LineError)
				if !ok {
					t.Fatalf("error is not *LineError: %T", errs[0])
				}
				if le.LineNumber != tc.wantLineNum {
					t.Errorf("LineNumber = %d; want %d", le.LineNumber, tc.wantLineNum)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
