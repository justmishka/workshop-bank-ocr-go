// Package web provides the HTTP server + JSON API for the Bank OCR tool.
//
// The server has two routes:
//
//   - GET  /            or /index.html  → serves the static UI (text/html)
//   - POST /api/parse                   → JSON in, JSON out (see ProcessOCR)
//
// All other paths return 404. The JSON contract is shared with the Python
// reference implementation so the existing frontend (static/index.html) can
// be reused unchanged.
package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/justmishka/workshop-bank-ocr-go/internal/formatter"
	"github.com/justmishka/workshop-bank-ocr-go/internal/parser"
	"github.com/justmishka/workshop-bank-ocr-go/internal/types"
)

// ProcessOCR runs validation + parsing + classification over OCR text and
// returns a response map ready to be marshalled to JSON.
//
// The shape matches the Python reference exactly:
//
//	{
//	  "accounts": [
//	    {"account": "123456789", "status": "OK",  "valid": true},
//	    {"account": "664371495", "status": "ERR", "valid": false},
//	    {"account": "86110??36", "status": "ILL", "valid": null}
//	  ],
//	  "errors": []
//	}
//
// When validation fails, "accounts" is an empty slice and "errors" carries
// one or more human-readable strings.
//
// The "valid" field is a *bool so that ILL entries marshal to JSON null
// (and not the string "null" or the boolean false).
func ProcessOCR(ocrText string) map[string]any {
	// Stage 1: input validation. If anything is wrong, surface it and stop.
	if errs := parser.ValidateOCRInput(ocrText); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return map[string]any{
			"accounts": []any{},
			"errors":   msgs,
		}
	}

	// Stage 2: parse into account-number strings.
	numbers := parser.ParseFile(ocrText)
	if len(numbers) == 0 {
		return map[string]any{
			"accounts": []any{},
			"errors":   []string{"No accounts found in input"},
		}
	}

	// Stage 3: classify each account. Build the response as concrete-typed
	// records so the *bool marshals to true / false / null as required.
	type record struct {
		Account string `json:"account"`
		Status  string `json:"status"`
		Valid   *bool  `json:"valid"`
	}

	results := make([]record, 0, len(numbers))
	for _, number := range numbers {
		acc := formatter.ClassifyToAccount(number)
		rec := record{Account: number}

		switch acc.Status {
		case types.StatusOK:
			rec.Status = "OK"
			t := true
			rec.Valid = &t
		case types.StatusERR:
			rec.Status = "ERR"
			f := false
			rec.Valid = &f
		case types.StatusILL:
			rec.Status = "ILL"
			rec.Valid = nil // JSON null
		default:
			// StatusAMB or anything unexpected: treat as ERR for the JSON
			// contract (frontend has no AMB lane today).
			rec.Status = acc.Status.String()
			f := false
			rec.Valid = &f
		}
		results = append(results, rec)
	}

	return map[string]any{
		"accounts": results,
		"errors":   []string{},
	}
}

// parseRequest is the JSON body shape for POST /api/parse.
type parseRequest struct {
	Text string `json:"text"`
}

// maxBodyBytes caps request bodies so a misbehaving client can't exhaust
// memory. 1 MiB is far more than any realistic OCR payload.
const maxBodyBytes = 1 << 20

// Handler returns an http.Handler wired to the two routes. staticDir is the
// directory containing index.html — it is taken as a parameter so the binary
// can be launched from any working directory.
func Handler(staticDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex(staticDir))
	mux.HandleFunc("/index.html", serveIndex(staticDir))
	mux.HandleFunc("/api/parse", handleParse)
	return mux
}

// serveIndex returns a handler that serves index.html on "/" and "/index.html"
// and 404s everything else. The "/" pattern in http.ServeMux is a catch-all,
// so the explicit path check is what produces 404s for unknown routes.
func serveIndex(staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := filepath.Join(staticDir, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, path)
	}
}

// handleParse implements POST /api/parse.
func handleParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the body with a hard cap and tolerate either application/json or
	// no content-type (the reference Python server doesn't enforce one).
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"accounts": []any{},
			"errors":   []string{fmt.Sprintf("failed to read request body: %v", err)},
		})
		return
	}

	var req parseRequest
	if len(strings.TrimSpace(string(body))) == 0 {
		// Empty body — treat as empty text (so the validator produces the
		// "no OCR input provided" error rather than a JSON parse failure).
		req.Text = ""
	} else if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"accounts": []any{},
			"errors":   []string{fmt.Sprintf("invalid JSON body: %v", err)},
		})
		return
	}

	result := ProcessOCR(req.Text)
	writeJSON(w, http.StatusOK, result)
}

// writeJSON marshals v as JSON and writes it with the given status code.
// On marshal error it logs and falls back to a 500 with a plain-text body
// because we cannot meaningfully recover the failed response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	buf, err := json.Marshal(v)
	if err != nil {
		log.Printf("web: json marshal failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf)))
	w.WriteHeader(status)
	_, _ = w.Write(buf)
}

// Run starts the HTTP server on the given port and blocks until the server
// exits. It prints a startup banner to stdout to match the Python reference.
// Access logging is suppressed (the default net/http server doesn't log
// requests, so there's nothing extra to silence).
func Run(port int, staticDir string) error {
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Bank OCR Web UI running at http://localhost:%d\n", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: Handler(staticDir),
		// Silence net/http's internal error log so we don't leak per-request
		// noise (e.g. broken-pipe writes) to stderr. The Python reference is
		// equally quiet.
		ErrorLog: log.New(io.Discard, "", 0),
	}
	return srv.ListenAndServe()
}
