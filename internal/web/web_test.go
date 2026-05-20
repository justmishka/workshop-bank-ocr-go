package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- ProcessOCR ---------------------------------------------------------

// validNines is a valid OCR entry whose decoded number is "000000000".
// All zeros is a trivial valid checksum (sum is 0, 0 mod 11 == 0).
const validNines = " _  _  _  _  _  _  _  _  _ \n" +
	"| || || || || || || || || |\n" +
	"|_||_||_||_||_||_||_||_||_|\n" +
	"\n"

// illegibleEntry is the same shape but the first digit's middle row is
// "   " (all spaces), which is not a known pattern → '?'.
const illegibleEntry = " _  _  _  _  _  _  _  _  _ \n" +
	"   | || || || || || || || |\n" +
	"|_||_||_||_||_||_||_||_||_|\n" +
	"\n"

func TestProcessOCR_ValidInputReturnsOK(t *testing.T) {
	got := ProcessOCR(validNines)

	if errs, ok := got["errors"].([]string); !ok || len(errs) != 0 {
		t.Fatalf("expected empty errors, got %#v", got["errors"])
	}

	// The accounts value is concrete-typed inside ProcessOCR. Round-trip
	// through JSON so we can assert against a generic structure without
	// caring about Go's static type.
	round := roundTrip(t, got)
	accounts, ok := round["accounts"].([]any)
	if !ok || len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %#v", round["accounts"])
	}
	first := accounts[0].(map[string]any)
	if first["account"] != "000000000" {
		t.Errorf("account: want 000000000, got %v", first["account"])
	}
	if first["status"] != "OK" {
		t.Errorf("status: want OK, got %v", first["status"])
	}
	if first["valid"] != true {
		t.Errorf("valid: want true, got %v (type %T)", first["valid"], first["valid"])
	}
}

func TestProcessOCR_IllegibleEntryReturnsILLWithNullValid(t *testing.T) {
	got := ProcessOCR(illegibleEntry)
	round := roundTrip(t, got)

	accounts := round["accounts"].([]any)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	first := accounts[0].(map[string]any)
	if first["status"] != "ILL" {
		t.Errorf("status: want ILL, got %v", first["status"])
	}
	// JSON null round-trips to a Go nil interface — that's exactly what the
	// frontend's `c === '?'` / status-check code expects.
	if first["valid"] != nil {
		t.Errorf("valid: want null (nil), got %v (type %T)", first["valid"], first["valid"])
	}
	// Verify the raw JSON literally contains `null` and not `false`/`"null"`.
	raw, _ := json.Marshal(got)
	if !bytes.Contains(raw, []byte(`"valid":null`)) {
		t.Errorf("expected raw JSON to contain \"valid\":null, got: %s", raw)
	}
}

func TestProcessOCR_EmptyInputReturnsErrors(t *testing.T) {
	got := ProcessOCR("")

	errs, ok := got["errors"].([]string)
	if !ok || len(errs) == 0 {
		t.Fatalf("expected non-empty errors, got %#v", got["errors"])
	}

	// "accounts" must marshal to an empty array, not null.
	raw, _ := json.Marshal(got)
	if !bytes.Contains(raw, []byte(`"accounts":[]`)) {
		t.Errorf("expected accounts to marshal as [], got: %s", raw)
	}
}

func TestProcessOCR_InvalidCharactersReturnsLineErrors(t *testing.T) {
	// 'X' on line 2 is invalid; the parser package surfaces a line-numbered
	// error. We only need to confirm errors are populated and accounts is
	// empty — exact message wording is owned by the parser package.
	bad := " _  _  _  _  _  _  _  _  _ \n" +
		"| || || || X || || || || |\n" +
		"|_||_||_||_||_||_||_||_||_|\n"

	got := ProcessOCR(bad)
	errs, ok := got["errors"].([]string)
	if !ok || len(errs) == 0 {
		t.Fatalf("expected validation errors, got %#v", got["errors"])
	}

	round := roundTrip(t, got)
	if accounts, _ := round["accounts"].([]any); len(accounts) != 0 {
		t.Errorf("expected no accounts on validation failure, got %d", len(accounts))
	}
}

// --- Handler / HTTP -----------------------------------------------------

// withStaticDir creates a temp dir containing index.html and returns its
// path. Using a temp dir keeps the test independent of cwd and the repo's
// real static/ contents.
func withStaticDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	html := []byte("<!doctype html><title>bank ocr test</title><p>hello</p>")
	if err := os.WriteFile(filepath.Join(dir, "index.html"), html, 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	return dir
}

func TestHandler_GETIndexReturnsHTML(t *testing.T) {
	srv := httptest.NewServer(Handler(withStaticDir(t)))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: want 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type: want text/html prefix, got %q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("bank ocr test")) {
		t.Errorf("body did not contain expected marker, got: %q", body)
	}
}

func TestHandler_GETIndexHTMLAlsoWorks(t *testing.T) {
	srv := httptest.NewServer(Handler(withStaticDir(t)))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/index.html")
	if err != nil {
		t.Fatalf("GET /index.html: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: want 200, got %d", resp.StatusCode)
	}
}

func TestHandler_UnknownPathReturns404(t *testing.T) {
	srv := httptest.NewServer(Handler(withStaticDir(t)))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatalf("GET /nope: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: want 404, got %d", resp.StatusCode)
	}
}

func TestHandler_POSTParseRoundTripsValidEntry(t *testing.T) {
	srv := httptest.NewServer(Handler(withStaticDir(t)))
	defer srv.Close()

	reqBody, _ := json.Marshal(map[string]string{"text": validNines})
	resp, err := http.Post(srv.URL+"/api/parse", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /api/parse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: want 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}

	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	accounts, ok := got["accounts"].([]any)
	if !ok || len(accounts) != 1 {
		t.Fatalf("accounts: want 1, got %#v", got["accounts"])
	}
	first := accounts[0].(map[string]any)
	if first["status"] != "OK" {
		t.Errorf("status: want OK, got %v", first["status"])
	}
	if first["valid"] != true {
		t.Errorf("valid: want true, got %v", first["valid"])
	}
	if first["account"] != "000000000" {
		t.Errorf("account: want 000000000, got %v", first["account"])
	}
}

func TestHandler_POSTParseILLEntryEmitsJSONNull(t *testing.T) {
	srv := httptest.NewServer(Handler(withStaticDir(t)))
	defer srv.Close()

	reqBody, _ := json.Marshal(map[string]string{"text": illegibleEntry})
	resp, err := http.Post(srv.URL+"/api/parse", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /api/parse: %v", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(raw, []byte(`"valid":null`)) {
		t.Errorf("expected response to contain \"valid\":null, got: %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"status":"ILL"`)) {
		t.Errorf("expected response to contain \"status\":\"ILL\", got: %s", raw)
	}
}

func TestHandler_POSTParseRejectsNonPOST(t *testing.T) {
	srv := httptest.NewServer(Handler(withStaticDir(t)))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/parse")
	if err != nil {
		t.Fatalf("GET /api/parse: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status: want 405, got %d", resp.StatusCode)
	}
}

func TestHandler_POSTParseBadJSONReturnsErrorPayload(t *testing.T) {
	srv := httptest.NewServer(Handler(withStaticDir(t)))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/parse", "application/json", strings.NewReader("{not-json"))
	if err != nil {
		t.Fatalf("POST /api/parse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", resp.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if errs, _ := got["errors"].([]any); len(errs) == 0 {
		t.Errorf("expected errors slice to be populated, got %#v", got["errors"])
	}
}

// --- helpers ------------------------------------------------------------

// roundTrip marshals v to JSON and unmarshals into a map[string]any, so
// tests can assert on a uniform generic structure regardless of which
// concrete types ProcessOCR uses internally.
func roundTrip(t *testing.T, v any) map[string]any {
	t.Helper()
	buf, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}
