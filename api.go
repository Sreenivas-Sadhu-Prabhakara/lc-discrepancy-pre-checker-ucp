package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sreenivas-sadhu-prabhakara/lc-discrepancy-pre-checker-ucp/internal/ucp"
)

// api serves the JSON interface the page talks to. Every call is same-origin
// and nothing leaves the machine.
type api struct {
	store *store
}

// routes registers every endpoint.
func (a *api) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/state", a.getState)
	mux.HandleFunc("POST /api/check", a.postCheck)
	mux.HandleFunc("POST /api/cases", a.postCase)
	mux.HandleFunc("GET /api/cases/{id}", a.getCase)
	mux.HandleFunc("DELETE /api/cases/{id}", a.deleteCase)
	mux.HandleFunc("PUT /api/thresholds", a.putThresholds)
	mux.HandleFunc("GET /api/sample", a.getSample)
	mux.HandleFunc("POST /api/report.txt", a.postReport)
}

// caseSummary is a saved case as the list shows it, with the headline figures
// from a fresh check so that the list says which cases still have something
// wrong with them.
type caseSummary struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Credit  string      `json:"credit"`
	Expiry  ucp.Date    `json:"expiry"`
	SavedOn ucp.Date    `json:"savedOn"`
	Verdict string      `json:"verdict"`
	Summary ucp.Summary `json:"summary"`
}

// statePayload is everything the page needs to draw itself.
type statePayload struct {
	Today      ucp.Date             `json:"today"`
	Thresholds ucp.Thresholds       `json:"thresholds"`
	Citations  []ucp.Citation       `json:"citations"`
	Catalogue  []ucp.CatalogueEntry `json:"catalogue"`
	Cases      []caseSummary        `json:"cases"`
	DataFile   string               `json:"dataFile"`
}

// getState answers with the figures in force, the catalogue of checks and every
// saved case.
func (a *api) getState(w http.ResponseWriter, r *http.Request) {
	data := a.store.snapshot()
	th := data.Thresholds.Normalise()

	cases := make([]caseSummary, 0, len(data.Cases))
	for _, c := range data.Cases {
		result := ucp.Check(c.Credit, c.Presentation, th)
		cases = append(cases, caseSummary{
			ID:      c.ID,
			Name:    c.Name,
			Credit:  c.Credit.Reference,
			Expiry:  c.Credit.ExpiryDate,
			SavedOn: c.SavedOn,
			Verdict: result.Verdict,
			Summary: result.Summary,
		})
	}

	writeJSON(w, http.StatusOK, statePayload{
		Today:      ucp.Today(),
		Thresholds: th,
		Citations:  th.Citations(),
		Catalogue:  ucp.Catalogue(),
		Cases:      cases,
		DataFile:   a.store.path,
	})
}

// checkRequest is a credit and a presentation to examine, optionally saved
// under a name at the same time.
type checkRequest struct {
	Case Case `json:"case"`
	Save bool `json:"save"`
}

// checkResponse is the result, and the case as saved where it was.
type checkResponse struct {
	Case   Case       `json:"case"`
	Result ucp.Result `json:"result"`
	Saved  bool       `json:"saved"`
}

// postCheck examines a credit and a presentation. Nothing is written unless the
// request asks for it, so the page can check as the user types.
func (a *api) postCheck(w http.ResponseWriter, r *http.Request) {
	var in checkRequest
	if !readJSON(w, r, &in) {
		return
	}

	c := in.Case.normalise()
	if err := c.validate(); err != nil {
		writeProblem(w, http.StatusBadRequest, err)
		return
	}

	if in.Save {
		saved, err := a.store.saveCase(c)
		if err != nil {
			writeProblem(w, http.StatusBadRequest, err)
			return
		}
		c = saved
	}

	writeJSON(w, http.StatusOK, checkResponse{
		Case:   c,
		Result: ucp.Check(c.Credit, c.Presentation, a.store.thresholds()),
		Saved:  in.Save,
	})
}

// postCase saves a case without checking it.
func (a *api) postCase(w http.ResponseWriter, r *http.Request) {
	var in Case
	if !readJSON(w, r, &in) {
		return
	}
	saved, err := a.store.saveCase(in)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, checkResponse{
		Case:   saved,
		Result: ucp.Check(saved.Credit, saved.Presentation, a.store.thresholds()),
		Saved:  true,
	})
}

// getCase answers with one saved case and a fresh check of it.
func (a *api) getCase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	saved, ok := a.store.getCase(id)
	if !ok {
		writeProblem(w, http.StatusNotFound, fmt.Errorf("there is no case %q saved", id))
		return
	}
	writeJSON(w, http.StatusOK, checkResponse{
		Case:   saved,
		Result: ucp.Check(saved.Credit, saved.Presentation, a.store.thresholds()),
		Saved:  true,
	})
}

// deleteCase removes a saved case.
func (a *api) deleteCase(w http.ResponseWriter, r *http.Request) {
	if err := a.store.deleteCase(r.PathValue("id")); err != nil {
		writeProblem(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"removed": r.PathValue("id")})
}

// putThresholds replaces the figures the checks apply. Every one of them is
// either variable by credit or a convention of this tool; none of the articles
// themselves can be edited here.
func (a *api) putThresholds(w http.ResponseWriter, r *http.Request) {
	var in ucp.Thresholds
	if !readJSON(w, r, &in) {
		return
	}
	saved, err := a.store.putThresholds(in)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"thresholds": saved,
		"citations":  saved.Citations(),
	})
}

// getSample answers with the worked example, so that somebody opening the tool
// for the first time can see what it does before typing anything.
func (a *api) getSample(w http.ResponseWriter, r *http.Request) {
	credit, presentation := ucp.Sample()
	c := Case{Name: "Worked example — spindle assemblies to Manila", Credit: credit, Presentation: presentation}.normalise()
	c.ID = "" // it is not saved; the page saves it if the user wants it
	writeJSON(w, http.StatusOK, checkResponse{
		Case:   c,
		Result: ucp.Check(credit, presentation, a.store.thresholds()),
		Saved:  false,
	})
}

// postReport returns the written report for a case as plain text, which is what
// an exporter prints or sends their banker.
func (a *api) postReport(w http.ResponseWriter, r *http.Request) {
	var in checkRequest
	if !readJSON(w, r, &in) {
		return
	}
	c := in.Case.normalise()
	if err := c.validate(); err != nil {
		writeProblem(w, http.StatusBadRequest, err)
		return
	}

	result := ucp.Check(c.Credit, c.Presentation, a.store.thresholds())
	body := ucp.Report(c.Credit, c.Presentation, result)

	name := idSafe(c.Name)
	if name == "" {
		name = "lc-pre-check"
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`-pre-check.txt"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, body)
}

// ---------- plumbing ----------

// readJSON reads a request body into v, capping its size and reporting a
// readable problem rather than a parser's complaint. It answers the request
// itself on failure and returns false.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxJSONBody))
	if err != nil {
		writeProblem(w, http.StatusRequestEntityTooLarge,
			fmt.Errorf("that is more than this tool reads in one request"))
		return false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		writeProblem(w, http.StatusBadRequest, errors.New("the request was empty"))
		return false
	}

	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		writeProblem(w, http.StatusBadRequest, fmt.Errorf("the request could not be read: %w", err))
		return false
	}
	if decoder.More() {
		writeProblem(w, http.StatusBadRequest, errors.New("the request carried more than one object"))
		return false
	}
	return true
}

// writeJSON answers with a JSON body.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(v)
}

// writeProblem answers with an error the page can show to the user as it
// stands.
func writeProblem(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"problem": err.Error()})
}
