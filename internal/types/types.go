// Package types defines the shared contract used across the bank-ocr packages.
//
// Status enum + Account struct are intentionally simple so every package
// (parser, checksum, formatter, corrector, web) speaks the same vocabulary.
package types

// Status describes the validation outcome of an account number.
type Status int

const (
	StatusOK  Status = iota // valid checksum, all digits readable
	StatusERR               // invalid checksum
	StatusILL               // contains '?' (illegible digit)
	StatusAMB               // ambiguous: multiple valid corrections found
)

// String returns the status marker used in formatted output.
// StatusOK returns "" because valid accounts have no marker.
func (s Status) String() string {
	switch s {
	case StatusOK:
		return ""
	case StatusERR:
		return "ERR"
	case StatusILL:
		return "ILL"
	case StatusAMB:
		return "AMB"
	default:
		return "UNKNOWN"
	}
}

// Account is the canonical result of parsing + validating one OCR entry.
type Account struct {
	// Number is the 9-character account number. '?' represents an illegible digit.
	Number string

	// Status is the validation outcome.
	Status Status

	// Alternatives lists valid corrections when Status == StatusAMB.
	// Empty for OK / ERR / ILL.
	Alternatives []string
}
