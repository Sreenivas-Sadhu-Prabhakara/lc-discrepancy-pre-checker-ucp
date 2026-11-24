package ucp

import (
	"strings"
	"testing"
)

// findingFor returns the finding a named rule produced, and whether it produced
// one. Where a rule produces several, the first is returned; tests that care
// about the rest use findingsFor.
func findingFor(r Result, ruleID string) (Finding, bool) {
	for _, f := range r.Findings {
		if f.Rule == ruleID {
			return f, true
		}
	}
	return Finding{}, false
}

// findingsFor returns every finding a named rule produced.
func findingsFor(r Result, ruleID string) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Rule == ruleID {
			out = append(out, f)
		}
	}
	return out
}

// skipped reports whether a rule recorded that it could not run.
func skipped(r Result, ruleID string) bool {
	for _, g := range r.NotChecked {
		if g.Rule == ruleID {
			return true
		}
	}
	return false
}

// wantFinding asserts that a rule produced a finding of a severity, and that
// the finding is properly formed: it says something, cites something, and names
// a severity that matches.
func wantFinding(t *testing.T, r Result, ruleID string, sev Severity, mustMention ...string) Finding {
	t.Helper()

	matching := findingsFor(r, ruleID)
	if len(matching) == 0 {
		t.Fatalf("%s produced no finding; skipped = %v, findings = %v", ruleID, skipped(r, ruleID), ruleIDs(r))
	}

	for _, f := range matching {
		if f.Severity != sev {
			continue
		}
		if f.Title == "" || f.Detail == "" || f.Citation == "" {
			t.Fatalf("%s produced a finding with an empty title, detail or citation: %+v", ruleID, f)
		}
		missing := ""
		for _, m := range mustMention {
			if !strings.Contains(f.Title+" "+f.Detail+" "+f.Fix+" "+f.Citation, m) {
				missing = m
			}
		}
		if missing != "" {
			t.Fatalf("%s (%s) never mentions %q:\n  %s\n  %s\n  %s", ruleID, sev, missing, f.Title, f.Detail, f.Fix)
		}
		return f
	}

	got := make([]string, 0, len(matching))
	for _, f := range matching {
		got = append(got, string(f.Severity)+": "+f.Title)
	}
	t.Fatalf("%s produced no %q finding, only:\n  %s", ruleID, sev, strings.Join(got, "\n  "))
	return Finding{}
}

// wantNoFinding asserts that a rule produced no finding of a severity.
func wantNoFinding(t *testing.T, r Result, ruleID string, sev Severity) {
	t.Helper()
	for _, f := range findingsFor(r, ruleID) {
		if f.Severity == sev {
			t.Fatalf("%s should not have produced a %q finding, but said: %s — %s", ruleID, sev, f.Title, f.Detail)
		}
	}
}

// wantSkipped asserts that a rule recorded a gap rather than guessing.
func wantSkipped(t *testing.T, r Result, ruleID string) {
	t.Helper()
	if !skipped(r, ruleID) {
		t.Fatalf("%s should have reported that it could not run; findings were %v", ruleID, ruleIDs(r))
	}
	if len(findingsFor(r, ruleID)) > 0 {
		t.Fatalf("%s reported a gap and a finding at the same time", ruleID)
	}
	for _, g := range r.NotChecked {
		if g.Rule == ruleID && strings.TrimSpace(g.Need) == "" {
			t.Fatalf("%s reported a gap without saying what it needed", ruleID)
		}
	}
}

// ruleIDs lists the rules that produced a finding, for a failure message.
func ruleIDs(r Result) []string {
	out := make([]string, 0, len(r.Findings))
	for _, f := range r.Findings {
		out = append(out, f.Rule+"/"+string(f.Severity))
	}
	return out
}

// minimalCredit and minimalPresentation are the smallest pair that lets one
// rule be tested without every other rule firing noise into the result. Each
// rule's test adds only the fields it needs.
func minimalCredit() Credit {
	return Credit{
		Reference:   "LC-TEST-1",
		Currency:    "USD",
		Beneficiary: Party{Name: "Coimbatore Spindle Works Private Limited", Country: "India"},
		Applicant:   Party{Name: "Sampaguita Textile Mills Inc", Country: "Philippines"},
	}
}

// billOfLading builds a transport document for a test.
func billOfLading(issued, onBoard string) Document {
	d := Document{
		Kind:      KindTransport,
		Title:     "bill of lading",
		Originals: 3,
		Transport: &Transport{
			Mode:            ModeBillOfLading,
			OriginalsIssued: 3,
			Vessel:          "MV Kaveri Star",
			Voyage:          "118W",
		},
	}
	if issued != "" {
		d.Date = MustDate(issued)
	}
	if onBoard != "" {
		d.Transport.OnBoardDate = MustDate(onBoard)
	} else {
		d.Transport.PreprintedOnBoard = true
	}
	return d
}

// invoiceFor builds a commercial invoice for a test.
func invoiceFor(c Credit, amount float64) Document {
	return Document{
		Kind:                KindInvoice,
		Title:               "commercial invoice",
		Issuer:              c.Beneficiary.Name,
		IssuedByBeneficiary: true,
		Originals:           1,
		GoodsDescription:    c.GoodsDescription,
		Invoice: &Invoice{
			Currency:  c.Currency,
			Amount:    amount,
			MadeOutTo: c.Applicant,
		},
	}
}
