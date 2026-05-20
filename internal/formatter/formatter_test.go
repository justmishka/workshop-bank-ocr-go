package formatter

import (
	"testing"

	"github.com/justmishka/workshop-bank-ocr-go/internal/types"
)

// --- ClassifyAccount ---------------------------------------------------------

func TestClassifyAccount(t *testing.T) {
	tests := []struct {
		name    string
		account string
		want    string
	}{
		{
			name:    "valid account returns just the number",
			account: "345882865",
			want:    "345882865",
		},
		{
			name:    "invalid checksum gets ERR marker",
			account: "664371495",
			want:    "664371495 ERR",
		},
		{
			name:    "illegible digits get ILL marker",
			account: "86110??36",
			want:    "86110??36 ILL",
		},
		{
			name:    "all zeros is valid (sum=0, 0 mod 11 == 0)",
			account: "000000000",
			want:    "000000000",
		},
		{
			name:    "all ones fails checksum",
			account: "111111111",
			want:    "111111111 ERR",
		},
		{
			name:    "single '?' anywhere wins over checksum (ILL beats ERR)",
			account: "1234?6789",
			want:    "1234?6789 ILL",
		},
		{
			name:    "known valid 123456789",
			account: "123456789",
			want:    "123456789",
		},
		{
			name:    "known valid 490867715",
			account: "490867715",
			want:    "490867715",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyAccount(tt.account)
			if got != tt.want {
				t.Errorf("ClassifyAccount(%q) = %q, want %q", tt.account, got, tt.want)
			}
		})
	}
}

// --- ClassifyToAccount -------------------------------------------------------

func TestClassifyToAccount(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   types.Account
	}{
		{
			name:   "valid -> StatusOK",
			number: "345882865",
			want:   types.Account{Number: "345882865", Status: types.StatusOK},
		},
		{
			name:   "invalid checksum -> StatusERR",
			number: "664371495",
			want:   types.Account{Number: "664371495", Status: types.StatusERR},
		},
		{
			name:   "illegible -> StatusILL",
			number: "86110??36",
			want:   types.Account{Number: "86110??36", Status: types.StatusILL},
		},
		{
			name:   "Alternatives is never populated by formatter",
			number: "123456789",
			want:   types.Account{Number: "123456789", Status: types.StatusOK},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyToAccount(tt.number)
			if got.Number != tt.want.Number {
				t.Errorf("Number = %q, want %q", got.Number, tt.want.Number)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.want.Status)
			}
			if len(got.Alternatives) != 0 {
				t.Errorf("Alternatives = %v, want empty (formatter must not populate)", got.Alternatives)
			}
		})
	}
}

// --- FormatOutput ------------------------------------------------------------

func TestFormatOutput(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "single valid entry",
			content: "    _  _     _  _  _  _  _ \n" +
				"  | _| _||_||_ |_   ||_||_|\n" +
				"  ||_  _|  | _||_|  ||_| _|\n" +
				"\n",
			want: "123456789",
		},
		{
			name: "mixed entries: valid then invalid",
			content: " _  _  _  _  _  _  _  _  _ \n" +
				"| || || || || || || || || |\n" +
				"|_||_||_||_||_||_||_||_||_|\n" +
				"\n" +
				"                           \n" +
				"  |  |  |  |  |  |  |  |  |\n" +
				"  |  |  |  |  |  |  |  |  |\n" +
				"\n",
			want: "000000000\n111111111 ERR",
		},
		{
			name:    "empty input returns empty string",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatOutput(tt.content)
			if got != tt.want {
				t.Errorf("FormatOutput() mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}
