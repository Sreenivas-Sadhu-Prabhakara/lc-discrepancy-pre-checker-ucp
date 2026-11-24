package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreenivas-sadhu-prabhakara/lc-discrepancy-pre-checker-ucp/internal/ucp"
)

// testServer builds the whole interface over a temporary data file, exactly as
// the binary does, so these tests exercise the same handlers and the same
// guard.
func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	s, err := openStore(filepath.Join(t.TempDir(), "cases.json"))
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	mux := http.NewServeMux()
	(&api{store: s}).routes(mux)
	server := httptest.NewServer(guard(mux))
	t.Cleanup(server.Close)
	return server
}

// do makes a request and decodes the answer.
func do(t *testing.T, server *httptest.Server, method, path string, body any, into any) int {
	t.Helper()

	var reader *strings.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encoding the request: %v", err)
		}
		reader = strings.NewReader(string(raw))
	} else {
		reader = strings.NewReader("")
	}

	request, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()

	if into != nil {
		if err := json.NewDecoder(response.Body).Decode(into); err != nil {
			t.Fatalf("reading the answer to %s %s: %v", method, path, err)
		}
	}
	return response.StatusCode
}

func TestGetState(t *testing.T) {
	server := testServer(t)

	var state statePayload
	if code := do(t, server, "GET", "/api/state", nil, &state); code != http.StatusOK {
		t.Fatalf("GET /api/state = %d", code)
	}
	if state.Today.Zero() {
		t.Errorf("the state must carry today's date")
	}
	if len(state.Catalogue) != len(ucp.Catalogue()) {
		t.Errorf("%d checks listed, want %d", len(state.Catalogue), len(ucp.Catalogue()))
	}
	if len(state.Citations) == 0 {
		t.Errorf("the state must carry the figures and their sources")
	}
	if state.Thresholds.PresentationPeriodDays != ucp.ArtPresentationPeriodDays {
		t.Errorf("the figures were not normalised: %+v", state.Thresholds)
	}
	if len(state.Cases) != 0 {
		t.Errorf("%d cases on a new store", len(state.Cases))
	}
	if state.DataFile == "" {
		t.Errorf("the state must say where the file is, so the user knows")
	}
}

func TestGetSampleAndCheckIt(t *testing.T) {
	server := testServer(t)

	var sample checkResponse
	if code := do(t, server, "GET", "/api/sample", nil, &sample); code != http.StatusOK {
		t.Fatalf("GET /api/sample = %d", code)
	}
	if sample.Saved {
		t.Errorf("the worked example must not save itself")
	}
	if sample.Case.ID != "" {
		t.Errorf("the worked example must come back unsaved, got id %q", sample.Case.ID)
	}
	if sample.Result.Summary.Discrepancies == 0 {
		t.Errorf("the worked example is built to carry discrepancies")
	}
	if sample.Result.Verdict == "" {
		t.Errorf("the result must carry its verdict as a value the page can show")
	}
	if sample.Result.ShipmentDate.Zero() || sample.Result.ShipmentDateReason == "" {
		t.Errorf("the result must say what it took as the date of shipment, and why")
	}

	// Checking the same case again without saving gives the same answer and
	// writes nothing.
	var again checkResponse
	if code := do(t, server, "POST", "/api/check", checkRequest{Case: sample.Case}, &again); code != http.StatusOK {
		t.Fatalf("POST /api/check = %d", code)
	}
	if again.Saved {
		t.Errorf("a check without save must not save")
	}
	if again.Result.Summary != sample.Result.Summary {
		t.Errorf("the same case gave a different answer: %+v then %+v", sample.Result.Summary, again.Result.Summary)
	}

	var state statePayload
	do(t, server, "GET", "/api/state", nil, &state)
	if len(state.Cases) != 0 {
		t.Errorf("a check without save left %d cases behind", len(state.Cases))
	}
}

func TestCheckSaveListOpenDelete(t *testing.T) {
	server := testServer(t)

	var sample checkResponse
	do(t, server, "GET", "/api/sample", nil, &sample)

	// Save it through the check endpoint.
	var saved checkResponse
	if code := do(t, server, "POST", "/api/check", checkRequest{Case: sample.Case, Save: true}, &saved); code != http.StatusOK {
		t.Fatalf("POST /api/check with save = %d", code)
	}
	if !saved.Saved || saved.Case.ID == "" {
		t.Fatalf("the case was not saved: %+v", saved)
	}

	// It appears in the list, with the verdict beside it.
	var state statePayload
	do(t, server, "GET", "/api/state", nil, &state)
	if len(state.Cases) != 1 {
		t.Fatalf("%d cases listed, want 1", len(state.Cases))
	}
	listed := state.Cases[0]
	if listed.ID != saved.Case.ID || listed.Verdict == "" || listed.Summary.Discrepancies == 0 {
		t.Errorf("the list does not say where the case stands: %+v", listed)
	}
	if listed.Credit != sample.Case.Credit.Reference {
		t.Errorf("the list shows credit %q, want %q", listed.Credit, sample.Case.Credit.Reference)
	}

	// Open it again.
	var opened checkResponse
	if code := do(t, server, "GET", "/api/cases/"+saved.Case.ID, nil, &opened); code != http.StatusOK {
		t.Fatalf("GET /api/cases/%s = %d", saved.Case.ID, code)
	}
	if opened.Case.ID != saved.Case.ID || !opened.Saved {
		t.Errorf("the wrong case came back: %+v", opened.Case)
	}
	if opened.Result.Summary != saved.Result.Summary {
		t.Errorf("reopening gave a different answer")
	}

	// A case that is not there.
	var problem map[string]any
	if code := do(t, server, "GET", "/api/cases/no-such-case", nil, &problem); code != http.StatusNotFound {
		t.Errorf("GET of a missing case = %d, want 404", code)
	}
	if problem["problem"] == nil {
		t.Errorf("a failure must explain itself: %v", problem)
	}

	// Delete it.
	if code := do(t, server, "DELETE", "/api/cases/"+saved.Case.ID, nil, nil); code != http.StatusOK {
		t.Fatalf("DELETE = %d", code)
	}
	do(t, server, "GET", "/api/state", nil, &state)
	if len(state.Cases) != 0 {
		t.Errorf("%d cases left after a delete", len(state.Cases))
	}
	if code := do(t, server, "DELETE", "/api/cases/"+saved.Case.ID, nil, nil); code != http.StatusNotFound {
		t.Errorf("deleting twice = %d, want 404", code)
	}
}

func TestPostCaseSavesWithoutChecking(t *testing.T) {
	server := testServer(t)

	var sample checkResponse
	do(t, server, "GET", "/api/sample", nil, &sample)

	var saved checkResponse
	if code := do(t, server, "POST", "/api/cases", sample.Case, &saved); code != http.StatusCreated {
		t.Fatalf("POST /api/cases = %d", code)
	}
	if !saved.Saved || saved.Case.ID == "" {
		t.Fatalf("the case was not saved: %+v", saved)
	}
}

func TestPutThresholdsChangesWhatTheChecksApply(t *testing.T) {
	server := testServer(t)

	var sample checkResponse
	do(t, server, "GET", "/api/sample", nil, &sample)

	own := ucp.StatutoryThresholds()
	own.PresentationPeriodDays = 7
	var answer struct {
		Thresholds ucp.Thresholds `json:"thresholds"`
		Citations  []ucp.Citation `json:"citations"`
	}
	if code := do(t, server, "PUT", "/api/thresholds", own, &answer); code != http.StatusOK {
		t.Fatalf("PUT /api/thresholds = %d", code)
	}
	if answer.Thresholds.PresentationPeriodDays != 7 || !answer.Thresholds.Edited {
		t.Fatalf("the figure was not taken: %+v", answer.Thresholds)
	}
	if len(answer.Citations) == 0 {
		t.Errorf("the answer must restate where the figures come from")
	}

	// The next check must use the new figure: the worked example presents 15
	// days after shipment, which is inside 21 days and outside 7.
	var again checkResponse
	do(t, server, "POST", "/api/check", checkRequest{Case: sample.Case}, &again)
	stale := false
	for _, f := range again.Result.Findings {
		if f.Rule == "D3-STALE" && f.Severity == ucp.SeverityDiscrepancy {
			stale = true
		}
	}
	if !stale {
		t.Errorf("a 7-day presentation period did not make the example stale")
	}
	if !again.Result.Thresholds.Edited {
		t.Errorf("the result must say it was produced on altered figures")
	}
}

func TestPostReportIsPlainTextAndOffersItself(t *testing.T) {
	server := testServer(t)

	var sample checkResponse
	do(t, server, "GET", "/api/sample", nil, &sample)

	raw, err := json.Marshal(checkRequest{Case: sample.Case})
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	response, err := server.Client().Post(server.URL+"/api/report.txt", "application/json", strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("POST /api/report.txt: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/report.txt = %d", response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("Content-Type = %q, want plain text", got)
	}
	if got := response.Header.Get("Content-Disposition"); !strings.Contains(got, "attachment") ||
		!strings.Contains(got, "pre-check.txt") {
		t.Errorf("Content-Disposition = %q, want an attachment with a readable name", got)
	}

	buffer := make([]byte, 64<<10)
	n, _ := response.Body.Read(buffer)
	body := string(buffer[:n])
	for _, want := range []string{
		"PRE-PRESENTATION SELF-CHECK",
		"VERDICT:",
		"FIGURES APPLIED",
		"UCP 600 art. 14(c)",
		"is not a bank's examination and it is not advice",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the report does not say %q", want)
		}
	}
}

func TestBadRequestsAreRefusedReadably(t *testing.T) {
	server := testServer(t)

	cases := []struct {
		name string
		path string
		body string
		want int
	}{
		{"an empty body", "/api/check", "", http.StatusBadRequest},
		{"not JSON", "/api/check", "{", http.StatusBadRequest},
		{"a field the tool does not know", "/api/check", `{"case":{},"save":false,"invented":1}`, http.StatusBadRequest},
		{"two objects in one request", "/api/check", `{"case":{}} {"case":{}}`, http.StatusBadRequest},
		{"a date that does not exist", "/api/check", `{"case":{"credit":{"expiryDate":"2023-02-29"}}}`, http.StatusBadRequest},
		{"a credit that cannot be saved", "/api/cases", `{"name":"x","credit":{"currency":"DOLLARS"}}`, http.StatusBadRequest},
		{"an impossible threshold", "/api/thresholds", `{"weekendDays":[99]}`, http.StatusOK}, // out-of-range days are dropped
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			method := "POST"
			if c.path == "/api/thresholds" {
				method = "PUT"
			}
			request, err := http.NewRequest(method, server.URL+c.path, strings.NewReader(c.body))
			if err != nil {
				t.Fatalf("building the request: %v", err)
			}
			request.Header.Set("Content-Type", "application/json")
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatalf("%s %s: %v", method, c.path, err)
			}
			defer response.Body.Close()

			if response.StatusCode != c.want {
				t.Fatalf("%s %s = %d, want %d", method, c.path, response.StatusCode, c.want)
			}
			if c.want == http.StatusBadRequest {
				var problem map[string]any
				if err := json.NewDecoder(response.Body).Decode(&problem); err != nil {
					t.Fatalf("the refusal was not readable JSON: %v", err)
				}
				if text, ok := problem["problem"].(string); !ok || strings.TrimSpace(text) == "" {
					t.Errorf("a refusal must say what was wrong, got %v", problem)
				}
			}
		})
	}
}

func TestOversizeRequestIsRefused(t *testing.T) {
	server := testServer(t)

	huge := `{"case":{"name":"` + strings.Repeat("x", maxJSONBody+1024) + `"}}`
	response, err := server.Client().Post(server.URL+"/api/check", "application/json", strings.NewReader(huge))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("an oversize request = %d, want 413", response.StatusCode)
	}
}

func TestGuardRefusesAnotherSiteAndSetsTheUsualHeaders(t *testing.T) {
	server := testServer(t)

	t.Run("a request from another site is refused", func(t *testing.T) {
		request, _ := http.NewRequest("GET", server.URL+"/api/state", nil)
		request.Header.Set("Origin", "https://somewhere-else.example")
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("a cross-site request = %d, want 403", response.StatusCode)
		}
	})

	t.Run("a request from this page is allowed", func(t *testing.T) {
		request, _ := http.NewRequest("GET", server.URL+"/api/state", nil)
		request.Header.Set("Origin", server.URL)
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("a same-origin request = %d, want 200", response.StatusCode)
		}
		for header, want := range map[string]string{
			"X-Content-Type-Options": "nosniff",
			"Referrer-Policy":        "no-referrer",
			"Cache-Control":          "no-store",
		} {
			if got := response.Header.Get(header); got != want {
				t.Errorf("%s = %q, want %q", header, got, want)
			}
		}
	})
}

func TestSameOrigin(t *testing.T) {
	cases := []struct {
		origin, host string
		want         bool
	}{
		{"http://127.0.0.1:8080", "127.0.0.1:8080", true},
		{"HTTP://127.0.0.1:8080", "127.0.0.1:8080", true},
		{"http://127.0.0.1:8080", "127.0.0.1:9000", false},
		{"https://evil.example", "127.0.0.1:8080", false},
		{"null", "127.0.0.1:8080", false},
		{"", "127.0.0.1:8080", false},
		{"not a url at all", "127.0.0.1:8080", false},
	}
	for _, c := range cases {
		if got := sameOrigin(c.origin, c.host); got != c.want {
			t.Errorf("sameOrigin(%q, %q) = %v, want %v", c.origin, c.host, got, c.want)
		}
	}
}

func TestEmbeddedInterfaceIsServedFromTheBinary(t *testing.T) {
	// The whole point of embedding is that the tool is one file. If a file the
	// page needs is missing from the embed, the tool is broken on any machine
	// but the one it was built on.
	for _, name := range []string{
		"web/index.html",
		"web/assets/app.js",
		"web/assets/style.css",
		"web/assets/layout.css",
		"web/assets/accent.css",
		"web/assets/theme-init.js",
		"web/assets/favicon.svg",
	} {
		body, err := webUI.ReadFile(name)
		if err != nil {
			t.Errorf("%s is not built into the binary: %v", name, err)
			continue
		}
		if len(body) == 0 {
			t.Errorf("%s is built in but empty", name)
		}
	}

	// And the page must carry no inline style or script, whatever else changes
	// about it, because its own policy forbids both.
	page, err := webUI.ReadFile("web/index.html")
	if err != nil {
		t.Fatalf("reading the page: %v", err)
	}
	text := string(page)
	for _, forbidden := range []string{"<style", "style=\"", "onclick=", "oninput=", "onchange=", "onsubmit=", "onload="} {
		if strings.Contains(text, forbidden) {
			t.Errorf("the page carries %q, which its own Content-Security-Policy forbids", forbidden)
		}
	}
	policy := "default-src 'self'; base-uri 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; font-src 'self'; form-action 'self'"
	if !strings.Contains(text, policy) {
		t.Errorf("the page does not carry the expected Content-Security-Policy")
	}
}
