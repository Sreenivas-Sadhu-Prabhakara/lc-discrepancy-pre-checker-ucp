package ucp

import "sort"

// Severity is how much a finding matters. The four levels exist because an
// honest pre-check has to say four different things, and collapsing them into
// "error" and "warning" would make the tool either alarming or useless.
type Severity string

const (
	// SeverityDiscrepancy — on what has been entered, the documents do not
	// comply. A bank examining them would be entitled to refuse.
	SeverityDiscrepancy Severity = "discrepancy"
	// SeverityCaution — it may be refused. Either the article leaves the bank
	// a discretion, or the answer turns on something the exporter has to read
	// off the document rather than type into a form.
	SeverityCaution Severity = "caution"
	// SeverityVerify — this tool cannot see it. Signatures, stamps, whether a
	// sheet is an original: these are examined on the face of the document and
	// a form cannot reproduce them.
	SeverityVerify Severity = "verify-yourself"
	// SeverityCleared — looks wrong, is not. A rule of UCP 600 expressly
	// permits what was entered, and saying so is as useful as finding a
	// discrepancy: it stops a document being reprinted for nothing.
	SeverityCleared Severity = "not-a-discrepancy"
)

// Rank orders the severities for a report: worst first.
func (s Severity) Rank() int {
	switch s {
	case SeverityDiscrepancy:
		return 0
	case SeverityCaution:
		return 1
	case SeverityVerify:
		return 2
	default:
		return 3
	}
}

// Label is the severity in words for a report line.
func (s Severity) Label() string {
	switch s {
	case SeverityDiscrepancy:
		return "Discrepancy"
	case SeverityCaution:
		return "Caution"
	case SeverityVerify:
		return "Check it yourself"
	default:
		return "Not a discrepancy"
	}
}

// Finding is one thing the check has to say.
type Finding struct {
	// Rule is the identifier of the check that produced it, so that the same
	// finding can be recognised across runs.
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Title    string   `json:"title"`
	// Detail says what was found, in the exporter's own figures.
	Detail string `json:"detail"`
	// Citation is the sub-article the finding rests on.
	Citation string `json:"citation"`
	// Fix is what to do about it before presenting. Empty where there is
	// nothing to do.
	Fix string `json:"fix,omitempty"`
	// Documents names the documents the finding is about.
	Documents []string `json:"documents,omitempty"`
}

// Gap is a check that could not be run, and what it would have needed. A
// pre-check that quietly skips half its rules is worse than one that tells you
// which ones it skipped.
type Gap struct {
	Rule string `json:"rule"`
	Need string `json:"need"`
}

// Summary counts the findings.
type Summary struct {
	Discrepancies int `json:"discrepancies"`
	Cautions      int `json:"cautions"`
	ToVerify      int `json:"toVerify"`
	Cleared       int `json:"cleared"`
	RulesRun      int `json:"rulesRun"`
	RulesSkipped  int `json:"rulesSkipped"`
}

// Result is everything the check has to report.
type Result struct {
	CheckedOn  Date       `json:"checkedOn"`
	Thresholds Thresholds `json:"thresholds"`

	// ShipmentDate is the date of shipment the rules give the presentation, and
	// Because says which sub-article gave it. Getting this wrong is the single
	// commonest cause of a refusal for late shipment or late presentation, so
	// the report states it rather than leaving it implied.
	ShipmentDate       Date   `json:"shipmentDate"`
	ShipmentDateReason string `json:"shipmentDateReason"`

	// LastPresentationDay is the last day presentation may be made on, after
	// both the stale-documents rule in article 14(c) and the extension in
	// article 29(a) have been applied, and Why explains it.
	LastPresentationDay Date   `json:"lastPresentationDay"`
	LastPresentationWhy string `json:"lastPresentationWhy"`

	// AnswerExpectedBy is the close of the fifth banking day following the day
	// of presentation — the day by which a refusal must have reached you
	// (articles 14(b) and 16(d)).
	AnswerExpectedBy Date `json:"answerExpectedBy"`

	Findings   []Finding `json:"findings"`
	NotChecked []Gap     `json:"notChecked"`
	Summary    Summary   `json:"summary"`

	// Verdict is the one line of plain English at the top of a report. It is a
	// field rather than a method so that it reaches the interface and the
	// written report as a value, and so that nobody has to compute it twice.
	// It never says the documents comply.
	Verdict string `json:"verdict"`
}

// Clean reports whether nothing was found that would stop the presentation —
// no discrepancy and nothing to be cautious about. It is deliberately not
// called "complying": whether a presentation complies is for the bank to
// determine on the documents themselves (UCP 600 arts. 14(a), 15 and 16).
func (r Result) Clean() bool {
	return r.Summary.Discrepancies == 0 && r.Summary.Cautions == 0
}

// verdictLine writes the one line of plain English a result leads with. It
// never says the documents comply: that is the issuing bank's determination,
// not this tool's (UCP 600 arts. 14(a), 15 and 16).
func verdictLine(s Summary) string {
	switch {
	case s.Discrepancies > 0:
		return plural(s.Discrepancies, "discrepancy to fix", "discrepancies to fix") + " before you present"
	case s.Cautions > 0:
		return "nothing certain to refuse on, " + plural(s.Cautions, "point", "points") + " to look at"
	case s.RulesRun == 0:
		return "nothing could be checked — enter the credit's terms and your documents"
	default:
		return "nothing found that would stop the presentation — the issuing bank still decides"
	}
}

// sortFindings puts the findings in the order they are useful in: worst first,
// then in the catalogue's order so that two runs of the same case read the
// same way.
func sortFindings(in []Finding, order map[string]int) []Finding {
	out := append([]Finding(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Severity.Rank() != out[j].Severity.Rank() {
			return out[i].Severity.Rank() < out[j].Severity.Rank()
		}
		return order[out[i].Rule] < order[out[j].Rule]
	})
	return out
}

// CatalogueEntry describes one check, for the interface and the documentation.
type CatalogueEntry struct {
	ID       string `json:"id"`
	Group    string `json:"group"`
	Title    string `json:"title"`
	Citation string `json:"citation"`
	Tests    string `json:"tests"`
}

// Catalogue lists every check this tool runs, in order. It is generated from
// the rules themselves, so the documentation cannot drift from the code.
func Catalogue() []CatalogueEntry {
	rules := allRules()
	out := make([]CatalogueEntry, 0, len(rules))
	for _, r := range rules {
		out = append(out, CatalogueEntry{
			ID:       r.ID,
			Group:    r.Group,
			Title:    r.Title,
			Citation: r.Citation,
			Tests:    r.Tests,
		})
	}
	return out
}
