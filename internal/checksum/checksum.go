// Package checksum validates bank account numbers using the Bank OCR checksum rule.
//
// Algorithm:
//
//	(d1 + 2*d2 + 3*d3 + ... + 9*d9) mod 11 == 0
//
// IMPORTANT: d1 is the RIGHTMOST digit of the account number, d9 is the
// leftmost. The index increases right-to-left. This is the most common
// place implementers go wrong — read it twice.
//
// Example for account "123456789" (left-to-right as written):
//
//	d1 = 9 (rightmost), d2 = 8, d3 = 7, ..., d9 = 1 (leftmost)
//	sum = 1*9 + 2*8 + 3*7 + 4*6 + 5*5 + 6*4 + 7*3 + 8*2 + 9*1 = 165
//	165 mod 11 == 0 → valid
package checksum

import "strings"

// accountLen is the expected length of an account number.
const accountLen = 9

// IsValid validates an account number's checksum.
//
// Return values mirror Python's Optional[bool] semantic:
//
//   - (true,  true)  — account is well-formed and the checksum is valid
//   - (false, true)  — account is well-formed but the checksum is invalid
//   - (false, false) — account contains '?' (illegible), so the checksum
//     cannot be decided; the caller should treat this as ILL, not ERR
//
// Malformed input (wrong length, non-digit characters that are not '?')
// is treated as "known and invalid" — (false, true). It is not the
// checksum layer's job to surface parser errors; the parser is the
// gatekeeper for shape, and an unrecognisable string here is simply not
// a valid account.
func IsValid(account string) (valid bool, known bool) {
	// '?' anywhere → illegible, cannot decide.
	if strings.ContainsRune(account, '?') {
		return false, false
	}

	// Wrong length → not a valid account, but the answer is known.
	if len(account) != accountLen {
		return false, true
	}

	// Compute sum with right-to-left indexing.
	// For position p in the string (0 = leftmost), the checksum weight is
	// (accountLen - p), because d1 is the rightmost digit.
	sum := 0
	for p := 0; p < accountLen; p++ {
		c := account[p]
		if c < '0' || c > '9' {
			// Non-digit, non-'?' → malformed, treat as known invalid.
			return false, true
		}
		digit := int(c - '0')
		weight := accountLen - p
		sum += weight * digit
	}

	return sum%11 == 0, true
}
