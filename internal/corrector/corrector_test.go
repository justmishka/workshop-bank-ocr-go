package corrector

import (
	"reflect"
	"sort"
	"testing"

	"workshop-bank-ocr-go/internal/types"
)

// OCR fixtures for whole-line nine-of-a-kind entries.
var (
	linesAllOnes = []string{
		"                           ",
		"  |  |  |  |  |  |  |  |  |",
		"  |  |  |  |  |  |  |  |  |",
	}
	linesAllEights = []string{
		" _  _  _  _  _  _  _  _  _ ",
		"|_||_||_||_||_||_||_||_||_|",
		"|_||_||_||_||_||_||_||_||_|",
	}
	linesAllZeros = []string{
		" _  _  _  _  _  _  _  _  _ ",
		"| || || || || || || || || |",
		"|_||_||_||_||_||_||_||_||_|",
	}
	linesAllBlank = []string{
		"                           ",
		"                           ",
		"                           ",
	}
	// linesAllSevens depicts "777777777" — invalid checksum
	// (1*7 + 2*7 + ... + 9*7 = 45*7 = 315; 315 mod 11 = 7). The "7" glyph is
	// `" _ "+"  |"+"  |"`; one edit can produce `1` (remove top underscore)
	// or `3` (add an underscore in the mid row), among others, giving the
	// corrector multiple positions to try.
	linesAllSevens = []string{
		" _  _  _  _  _  _  _  _  _ ",
		"  |  |  |  |  |  |  |  |  |",
		"  |  |  |  |  |  |  |  |  |",
	}
)

func TestGenerateVariants(t *testing.T) {
	// "1" pattern: "   " + "  |" + "  |" — 7 spaces, 2 pipes.
	// Each space yields 2 variants (|, _); each pipe yields 1 (space).
	// Expected: 7*2 + 2 = 16 variants.
	got := generateVariants("     |  |")
	if len(got) != 16 {
		t.Fatalf("expected 16 variants for '1' pattern, got %d: %q", len(got), got)
	}

	// Sanity: both pipe-to-space replacements should be present.
	wantPipeReplacements := []string{
		"        |", // first pipe → space (index 5)
		"     |   ", // second pipe → space (index 8)
	}
	for _, w := range wantPipeReplacements {
		if !contains(got, w) {
			t.Errorf("missing variant %q in %q", w, got)
		}
	}
}

func TestGenerateVariantsCountForZero(t *testing.T) {
	// "0" pattern: " _ " + "| |" + "|_|" — 3 spaces, 6 pipes/underscores.
	// Expected: 3*2 + 6 = 12 variants.
	got := generateVariants(" _ | ||_|")
	if len(got) != 12 {
		t.Fatalf("expected 12 variants for '0' pattern, got %d", len(got))
	}
}

func TestCorrectAccount(t *testing.T) {
	tests := []struct {
		name             string
		account          string
		lines            []string
		wantStatus       types.Status
		wantNumber       string   // ignored when wantStatus == AMB (we check Alternatives instead)
		wantAlternatives []string // sorted; only meaningful for AMB
		// minAlternatives lets us assert "more than N" without nailing the exact set
		// (useful for the 888888888 case where many corrections exist).
		minAlternatives int
	}{
		{
			// 000000000 has a valid checksum already. The corrector contract
			// is "find a DIFFERENT valid 1-edit account" — with none, it
			// falls through to statusFromAccount, which has no '?' rule and
			// therefore returns ERR. The pipeline never calls the corrector
			// on already-valid accounts, so this branch documents defensive
			// behaviour, not happy-path output.
			name:       "already-valid account with no different 1-edit alt falls through to ERR",
			account:    "000000000",
			lines:      linesAllZeros,
			wantStatus: types.StatusERR,
			wantNumber: "000000000",
		},
		{
			name:       "ILL account that stays ILL — all blank, no corrections possible",
			account:    "?????????",
			lines:      linesAllBlank,
			wantStatus: types.StatusILL,
			wantNumber: "?????????",
		},
		{
			name:            "888888888 has multiple single-edit corrections (AMB)",
			account:         "888888888",
			lines:           linesAllEights,
			wantStatus:      types.StatusAMB,
			minAlternatives: 2,
		},
		{
			name:       "111111111 is correctable — at least one valid alternative",
			account:    "111111111",
			lines:      linesAllOnes,
			wantStatus: 0, // filled in below — depends on uniqueness; check via custom logic
			// We assert in the body: status is either OK or AMB and at least
			// one alternative contains a '7' substitution (711111111 is valid).
		},
		{
			name:       "777777777 has at least one valid single-edit correction",
			account:    "777777777",
			lines:      linesAllSevens,
			wantStatus: 0, // see custom assertion in test body
		},
		{
			name:       "fewer than 3 entry lines is handled defensively",
			account:    "12345678?",
			lines:      []string{"   ", "   "},
			wantStatus: types.StatusILL,
			wantNumber: "12345678?",
		},
		{
			name:       "empty entryLines for ERR returns ERR unchanged",
			account:    "123456788",
			lines:      nil,
			wantStatus: types.StatusERR,
			wantNumber: "123456788",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CorrectAccount(tt.account, tt.lines)

			// Special-cased assertion: 111111111 — we accept OK or AMB so long as
			// 711111111 (which has a valid checksum) appears as a candidate.
			if tt.name == "111111111 is correctable — at least one valid alternative" {
				if got.Status != types.StatusOK && got.Status != types.StatusAMB {
					t.Fatalf("expected OK or AMB, got %s", got.Status)
				}
				found := got.Number == "711111111"
				for _, alt := range got.Alternatives {
					if alt == "711111111" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected 711111111 among Number/Alternatives, got %+v", got)
				}
				return
			}

			// Special-cased assertion: 777777777 — accept OK or AMB; just verify
			// the corrector produced at least one valid alternative.
			if tt.name == "777777777 has at least one valid single-edit correction" {
				if got.Status != types.StatusOK && got.Status != types.StatusAMB {
					t.Fatalf("expected OK or AMB for 777777777, got %s", got.Status)
				}
				if got.Status == types.StatusAMB && len(got.Alternatives) < 2 {
					t.Errorf("AMB result should have ≥2 alternatives, got %d", len(got.Alternatives))
				}
				return
			}

			if got.Status != tt.wantStatus {
				t.Fatalf("status: got %s, want %s (account=%+v)", got.Status, tt.wantStatus, got)
			}

			if tt.wantStatus != types.StatusAMB && got.Number != tt.wantNumber {
				t.Errorf("number: got %q, want %q", got.Number, tt.wantNumber)
			}

			if tt.wantStatus == types.StatusAMB {
				if tt.minAlternatives > 0 && len(got.Alternatives) < tt.minAlternatives {
					t.Errorf("expected at least %d alternatives, got %d: %v",
						tt.minAlternatives, len(got.Alternatives), got.Alternatives)
				}
				if tt.wantAlternatives != nil && !reflect.DeepEqual(got.Alternatives, tt.wantAlternatives) {
					t.Errorf("alternatives: got %v, want %v", got.Alternatives, tt.wantAlternatives)
				}
				// Alternatives must always be sorted for AMB.
				if !sort.StringsAreSorted(got.Alternatives) {
					t.Errorf("alternatives not sorted: %v", got.Alternatives)
				}
			}

			// Non-AMB results must have empty Alternatives.
			if tt.wantStatus != types.StatusAMB && len(got.Alternatives) != 0 {
				t.Errorf("expected empty Alternatives for non-AMB result, got %v", got.Alternatives)
			}
		})
	}
}

// TestCorrectAccountAlternativesAreSorted explicitly nails down the contract
// that AMB alternatives come back in lexicographic order — the formatter
// relies on this for deterministic output.
func TestCorrectAccountAlternativesAreSorted(t *testing.T) {
	got := CorrectAccount("888888888", linesAllEights)
	if got.Status != types.StatusAMB {
		t.Fatalf("expected AMB for 888888888, got %s", got.Status)
	}
	if !sort.StringsAreSorted(got.Alternatives) {
		t.Errorf("alternatives not sorted: %v", got.Alternatives)
	}
	// Sanity: no duplicates.
	seen := make(map[string]struct{})
	for _, a := range got.Alternatives {
		if _, dup := seen[a]; dup {
			t.Errorf("duplicate alternative %q in %v", a, got.Alternatives)
		}
		seen[a] = struct{}{}
	}
}

// TestCorrectAccountDoesNotReturnOriginal guards the rule that the original
// (already known-bad) account is never offered as its own "correction".
func TestCorrectAccountDoesNotReturnOriginal(t *testing.T) {
	got := CorrectAccount("888888888", linesAllEights)
	if got.Number == "888888888" && got.Status == types.StatusOK {
		t.Fatalf("original account returned as OK alternative")
	}
	for _, alt := range got.Alternatives {
		if alt == "888888888" {
			t.Errorf("original account 888888888 appeared in alternatives")
		}
	}
}

// contains reports whether haystack contains needle.
func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
