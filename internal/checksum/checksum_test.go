package checksum

import "testing"

func TestIsValid(t *testing.T) {
	tests := []struct {
		name      string
		account   string
		wantValid bool
		wantKnown bool
	}{
		{
			name:      "known valid account from Python reference",
			account:   "345882865",
			wantValid: true,
			wantKnown: true,
		},
		{
			name:      "all zeros are valid (sum = 0, 0 mod 11 == 0)",
			account:   "000000000",
			wantValid: true,
			wantKnown: true,
		},
		{
			name:      "123456789 is valid (sum = 165, 165 mod 11 == 0)",
			account:   "123456789",
			wantValid: true,
			wantKnown: true,
		},
		{
			name:      "known invalid account",
			account:   "664371495",
			wantValid: false,
			wantKnown: true,
		},
		{
			name:      "all ones invalid (sum = 45, 45 mod 11 == 1)",
			account:   "111111111",
			wantValid: false,
			wantKnown: true,
		},
		{
			name:      "illegible middle digits return unknown",
			account:   "86110??36",
			wantValid: false,
			wantKnown: false,
		},
		{
			name:      "single trailing '?' returns unknown",
			account:   "12345678?",
			wantValid: false,
			wantKnown: false,
		},
		{
			name:      "single leading '?' returns unknown",
			account:   "?23456789",
			wantValid: false,
			wantKnown: false,
		},
		{
			name:      "all illegible returns unknown",
			account:   "?????????",
			wantValid: false,
			wantKnown: false,
		},
		{
			name:      "empty string is known invalid (wrong length)",
			account:   "",
			wantValid: false,
			wantKnown: true,
		},
		{
			name:      "too short is known invalid",
			account:   "12345",
			wantValid: false,
			wantKnown: true,
		},
		{
			name:      "too long is known invalid",
			account:   "1234567890",
			wantValid: false,
			wantKnown: true,
		},
		{
			name:      "non-digit garbage is known invalid",
			account:   "abcdefghi",
			wantValid: false,
			wantKnown: true,
		},
		{
			name:      "mixed digits with letter is known invalid",
			account:   "12345678X",
			wantValid: false,
			wantKnown: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotValid, gotKnown := IsValid(tc.account)
			if gotValid != tc.wantValid || gotKnown != tc.wantKnown {
				t.Errorf("IsValid(%q) = (%v, %v), want (%v, %v)",
					tc.account, gotValid, gotKnown, tc.wantValid, tc.wantKnown)
			}
		})
	}
}

// TestRightToLeftIndexing pins the most common bug: indexing left-to-right
// instead of right-to-left. If someone "fixes" the implementation to use
// left-to-right weights, this case will start failing.
//
// "123456789" is valid only with right-to-left indexing:
//   - right-to-left: 1*9 + 2*8 + ... + 9*1 = 165, 165 mod 11 == 0 → valid
//   - left-to-right: 1*1 + 2*2 + ... + 9*9 = 285, 285 mod 11 == 9 → invalid
func TestRightToLeftIndexing(t *testing.T) {
	valid, known := IsValid("123456789")
	if !known {
		t.Fatal("expected known=true for 123456789")
	}
	if !valid {
		t.Error("123456789 must be valid — check that indexing is right-to-left (d1 = rightmost)")
	}
}
