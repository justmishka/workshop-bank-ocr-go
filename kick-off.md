# Project Kick-Off — Bank OCR

_Workshop Product 01_

**Source:** https://codingdojo.org/kata/BankOCR/
**Original authors:** Emmanuel Gaillot and Christophe Thibaut (XP2006)

---

## Product Description

A bank has a machine that scans account numbers from paper documents using OCR (Optical Character Recognition). The machine produces files where each account number is represented as ASCII art using pipes (`|`) and underscores (`_`).

The software needs to:
1. Parse OCR output files into readable account numbers
2. Validate account numbers using a checksum algorithm
3. Generate output files with validation status
4. Attempt error correction for invalid/illegible numbers

### User Story 1: OCR Parsing

Each file entry is 4 lines:
- Lines 1-3: account number in ASCII art (27 characters each)
- Line 4: blank
- Each account number has exactly 9 digits (0-9)

Digit representations (3 chars wide × 3 lines tall):

```
 _     _  _     _  _  _  _  _
| |  | _| _||_||_ |_   ||_||_|
|_|  ||_  _|  | _||_|  ||_| _|

 0  1  2  3  4  5  6  7  8  9
```

### User Story 2: Checksum Validation

Account number checksum (d1 = rightmost digit):

```
(d1 + 2*d2 + 3*d3 + ... + 9*d9) mod 11 = 0
```

**Note:** Digits are numbered right-to-left. Be careful with the ordering.

### User Story 3: Output Formatting

Output file format — one account per line:
- Valid: just the number
- Illegible characters: replace with `?`, mark `ILL`
- Invalid checksum: mark `ERR`

```
457508000
664371495 ERR
86110??36 ILL
```

### User Story 4: Error Correction

For `ERR` or `ILL` accounts, try modifying exactly one pipe or underscore:
- One valid match → use it
- Multiple valid matches → mark `AMB` with possibilities
- No valid match → keep as `ILL`

### Test Cases

- All same digits: `000000000` through `999999999`
- Mixed: `123456789`
- Illegible: characters marked `?` with `ILL` status
- Ambiguous: multiple valid corrections listed

---

## Kick-Off Session Notes

**Date:** 2026-03-16
**Facilitator:** Alex
**Attendees:** Rex, Clara, Nova, Finn, Sage, Dex, Alex, Pixel
**Mode:** Role-play

### Product Vision (Clara)

This is a classic parsing + validation kata. The user is a bank employee (or Miška for demo purposes) who receives OCR output files and needs reliable account number extraction. The product must handle real-world messiness: illegible characters and scanner errors.

Four user stories, progressive complexity:
1. Parse OCR → numbers (core)
2. Validate with checksum (correctness)
3. Format output with status (reporting)
4. Error correction (robustness)

**User:** Miška (she will use and test the software at Sprint Review).

### Architecture (Nova)

Simple CLI application. Python recommended — good string handling, clean for parsing.

```
bank-ocr/
├── src/
│   ├── __init__.py
│   ├── parser.py        ← OCR text → digit recognition
│   ├── checksum.py      ← account number validation
│   ├── formatter.py     ← output formatting with status
│   └── corrector.py     ← error correction logic
├── tests/
│   ├── test_parser.py
│   ├── test_checksum.py
│   ├── test_formatter.py
│   └── test_corrector.py
├── samples/             ← sample OCR input files for testing
├── README.md
└── pyproject.toml
```

**Approach:** Each user story maps to a module. Parser is the foundation — everything depends on it. Checksum is pure math. Formatter combines parser + checksum. Corrector is the most complex (permutations of pipe/underscore changes).

**Tech decision:** Python 3.12+. No external dependencies needed — this is pure string processing + math. pytest for testing.

### Security Assessment (Hugo — via Pixel)

**Scope: Low.** No network, no auth, no user data, no database. Reads files from disk, writes files to disk. Local CLI tool only. No security review needed per story — Hugo batch-reviews at sprint end.

### Design Assessment (Luna — via Pixel)

**No UI component.** CLI tool. Luna notes: output formatting matters for readability — clean columns, clear status markers. Finn should make the terminal output clean.

### Sprint Approach

- **Sprint length:** 1 day
- **Sprint Goal:** "Parse OCR files into validated account numbers with error reporting"
- Stories 1-3 are the core. Story 4 (error correction) is stretch — include if time permits.

---

## Backlog (Clara)

### Epic: Bank OCR (WRKSHP epic)

**Story 1: Parse OCR digits**
_As a user, I want to parse an OCR file so that I get readable account numbers._

**Acceptance Criteria:**
- Given an OCR file with valid digit patterns, When I run the parser, Then I get the correct 9-digit account number as a string
- Given an OCR file with multiple entries, When I run the parser, Then each entry is parsed separately
- Given an OCR file with an unrecognized digit pattern, When I run the parser, Then that digit is represented as `?`

**Story 2: Validate checksum**
_As a user, I want account numbers validated so that I know which ones are correct._

**Acceptance Criteria:**
- Given account number `345882865`, When I calculate the checksum, Then it is valid (result = 0)
- Given account number `664371495`, When I calculate the checksum, Then it is invalid (result ≠ 0)
- Given an account number with `?` characters, When I validate, Then it is marked as illegible (skip checksum)

**Story 3: Format output with status**
_As a user, I want a formatted output file so that I can see which accounts are valid, invalid, or illegible._

**Acceptance Criteria:**
- Given valid account numbers, When I generate output, Then each appears on its own line with no status marker
- Given an account with invalid checksum, When I generate output, Then it shows `ERR` after the number
- Given an account with illegible digits, When I generate output, Then `?` replaces illegible digits and it shows `ILL`

**Story 4: Error correction (stretch)**
_As a user, I want the system to attempt to fix errors so that I recover as many valid numbers as possible._

**Acceptance Criteria:**
- Given an `ERR` account, When I try all single pipe/underscore modifications, Then if exactly one produces a valid checksum, I use that number
- Given an `ILL` account, When I try all single pipe/underscore modifications per illegible digit, Then if exactly one produces a valid checksum, I use that number
- Given an `ERR` or `ILL` account with multiple valid corrections, When I generate output, Then it shows `AMB` followed by the list of possibilities

---

## Refinement Notes

**Alex (facilitator):** Four clean stories, progressive complexity. Story 4 is stretch — don't force it.

**Finn:** Stories 1-3 are straightforward. Story 1 is the foundation — parser needs to be solid. Story 2 is pure math, easy. Story 3 is formatting glue. Story 4 is the interesting one — combinatorial.

**Dex:** All stories are testable. The kata provides test cases. I'll add edge cases: empty files, malformed input, files with wrong line counts.

**Sage:** Clean module separation. Each story = each module = each PR. Easy to review.

**Nova:** No architectural concerns. This is a single-responsibility CLI tool. The module split I proposed maps 1:1 to stories.

### Estimation (Planning Poker)

| Story | Clara | Finn | Sage | Dex | Nova | **Final** |
|-------|-------|------|------|-----|------|-----------|
| S1: Parse OCR | 2 | 2 | 2 | 2 | 2 | **2** |
| S2: Validate checksum | 1 | 1 | 1 | 1 | 1 | **1** |
| S3: Format output | 1 | 2 | 1 | 1 | 1 | **1** |
| S4: Error correction | 3 | 3 | 3 | 5 | 3 | **3** |

**Discussion on S4:** Dex estimated 5 ("lots of permutations to test"), team discussed — the correction logic is well-defined (single pipe/underscore change), not truly complex. Dex agreed to 3 after Finn explained the approach.

**Total: 7 story points** (4 core + 3 stretch)

**Story 5: Web UI**
_As a user, I want a web interface so that I can paste or upload OCR text and see parsed results visually._

**Acceptance Criteria:**
- Given the web UI, When I paste OCR text into the input area, Then I see parsed account numbers with validation status
- Given the web UI, When I upload an OCR file, Then it is processed and results are displayed
- Given the web UI, When results include ERR or ILL accounts, Then they are visually highlighted differently from valid accounts

---

## Sprint Backlog

**Sprint Goal:** "Parse OCR files into validated account numbers with error reporting and web UI"

| Story | Jira | Points | Priority | DoD Categories |
|-------|------|--------|----------|----------------|
| S1: Parse OCR | WRKSHP-2 | 2 | Must | Sage + Dex + Rex |
| S2: Validate checksum | WRKSHP-3 | 1 | Must | Sage + Dex + Rex |
| S3: Format output | WRKSHP-4 | 1 | Must | Sage + Dex + Rex |
| S4: Error correction | WRKSHP-5 | 3 | Stretch | Sage + Dex + Rex |
| S5: Web UI | WRKSHP-6 | 3 | Must | Sage + Dex + Rex + Luna |

**Total: 10 story points.** S1-S3 + S5 are committed (7 points). S4 is stretch.

### Rex's Summary

> Sprint 1 for Bank OCR is active. Sprint Goal: "Parse OCR files into validated account numbers with error reporting and web UI." 5 stories, 10 points. Finn starts with S1 (parser), Mia handles S5 (UI). Go.
