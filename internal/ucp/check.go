package ucp

import "strings"

// rule is one check: an identifier, what it rests on, and a function of the
// examination. Rules are pure: they read the examination and return findings.
type rule struct {
	ID       string
	Group    string
	Title    string
	Citation string
	Tests    string
	eval     func(e *exam) []Finding
}

// Groups, in the order a report reads best in.
const (
	groupDates     = "Deadlines and dates"
	groupAmounts   = "Amounts, quantities and insurance"
	groupParties   = "Descriptions and parties"
	groupDocuments = "Documents required"
	groupTransport = "Transport document"
	groupProcess   = "Presentation itself"
)

// allRules is the whole catalogue, in order. Adding a rule here is all it takes
// for the interface, the documentation and the coverage test to know about it.
func allRules() []rule {
	var out []rule
	out = append(out, dateRules()...)
	out = append(out, amountRules()...)
	out = append(out, partyRules()...)
	out = append(out, documentRules()...)
	out = append(out, transportRules()...)
	out = append(out, processRules()...)
	return out
}

// exam carries the credit, the presentation, the figures in force and the few
// values every rule needs, worked out once.
type exam struct {
	C Credit
	P Presentation
	T Thresholds

	// shipment is the date of shipment the rules give the presentation.
	shipment    Date
	shipmentWhy string
	hasShipment bool

	// invoice is the first commercial invoice presented, if there is one.
	invoice    Document
	hasInvoice bool

	// drawn is the amount being claimed: the amount stated for the drawing if
	// there is one, otherwise the invoice total.
	drawn    float64
	hasDrawn bool

	gaps []Gap
	id   string // the rule being run, for gap reporting
}

// skip records that the rule now running could not be applied, and why.
func (e *exam) skip(need string) []Finding {
	e.gaps = append(e.gaps, Gap{Rule: e.id, Need: need})
	return nil
}

// find builds a finding for the rule now running.
func (e *exam) find(sev Severity, title, detail, citation, fix string, docs ...string) Finding {
	return Finding{
		Rule:      e.id,
		Severity:  sev,
		Title:     title,
		Detail:    detail,
		Citation:  citation,
		Fix:       fix,
		Documents: docs,
	}
}

// money renders an amount in the credit's currency at the configured precision.
func (e *exam) money(v float64) string { return Money(e.C.Currency, v, e.T.AmountDecimals) }

// Check runs every rule over a credit and a presentation and returns everything
// it found. It is the only entry point the rest of the program uses, and it
// changes nothing: the same inputs give the same result every time, apart from
// the day the report says it was produced on.
func Check(c Credit, p Presentation, t Thresholds) Result {
	t = t.Normalise()
	c = c.Normalise()
	p = p.Normalise()

	e := &exam{C: c, P: p, T: t}
	e.shipment, e.shipmentWhy, e.hasShipment = p.ShipmentDate()
	e.invoice, e.hasInvoice = p.First(KindInvoice)

	switch {
	case p.DrawnAmount > 0:
		e.drawn, e.hasDrawn = p.DrawnAmount, true
	case e.hasInvoice && e.invoice.Invoice != nil && e.invoice.Invoice.Amount > 0:
		e.drawn, e.hasDrawn = e.invoice.Invoice.Amount, true
	}

	rules := allRules()
	order := make(map[string]int, len(rules))
	var findings []Finding
	run := 0

	for i, r := range rules {
		order[r.ID] = i
		e.id = r.ID
		before := len(e.gaps)
		got := r.eval(e)
		if len(e.gaps) == before {
			run++
		}
		findings = append(findings, got...)
	}

	result := Result{
		CheckedOn:          Today(),
		Thresholds:         t,
		ShipmentDate:       e.shipment,
		ShipmentDateReason: e.shipmentWhy,
		Findings:           sortFindings(findings, order),
		NotChecked:         e.gaps,
		Summary: Summary{
			RulesRun:     run,
			RulesSkipped: len(rules) - run,
		},
	}
	result.LastPresentationDay, result.LastPresentationWhy = lastPresentationDay(e)
	if !p.Date.Zero() {
		if by, ok := t.AddBankingDays(p.Date, t.ExaminationBankingDays, c.BankClosedDays); ok {
			result.AnswerExpectedBy = by
		}
	}

	for _, f := range result.Findings {
		switch f.Severity {
		case SeverityDiscrepancy:
			result.Summary.Discrepancies++
		case SeverityCaution:
			result.Summary.Cautions++
		case SeverityVerify:
			result.Summary.ToVerify++
		case SeverityCleared:
			result.Summary.Cleared++
		}
	}
	result.Verdict = verdictLine(result.Summary)
	if result.NotChecked == nil {
		result.NotChecked = []Gap{}
	}
	if result.Findings == nil {
		result.Findings = []Finding{}
	}
	return result
}

// lastPresentationDay works out the last day a presentation may be made on and
// explains it. Two limits can bite and the earlier of them wins:
//
//   - the expiry date of the credit (article 6(e)), extended to the next
//     banking day if the bank is closed on it (article 29(a));
//   - where an original transport document subject to articles 19 to 25 is
//     presented, 21 calendar days after the date of shipment, or the period the
//     credit states instead (article 14(c)) — and that limit is never extended
//     beyond the expiry date, which article 14(c) says in terms.
func lastPresentationDay(e *exam) (Date, string) {
	expiry, extended := e.T.LastDayForPresentation(e.C.ExpiryDate, e.C.BankClosedDays)

	var stale Date
	period := e.C.PresentationPeriodDays
	stated := period > 0
	if !stated {
		period = e.T.PresentationPeriodDays
	}
	if e.hasShipment && e.P.HasTransportUnderArticles19To25() {
		stale = e.shipment.AddDays(period)
	}

	switch {
	case expiry.Zero() && stale.Zero():
		return Date{}, ""
	case expiry.Zero():
		return stale, staleReason(period, stated, e.shipment)
	case stale.Zero():
		return expiry, expiryReason(e.C.ExpiryDate, extended)
	case stale.Before(expiry):
		return stale, staleReason(period, stated, e.shipment) + ", which falls before the expiry date of " + e.C.ExpiryDate.Human()
	default:
		return expiry, expiryReason(e.C.ExpiryDate, extended) +
			"; the " + plural(period, "day", "days") + " after shipment would have run to " + stale.Human() +
			", but art. 14(c) allows no presentation after expiry in any event"
	}
}

func staleReason(period int, stated bool, shipment Date) string {
	source := "art. 14(c)"
	if stated {
		source = "the period the credit states"
	}
	return plural(period, "calendar day", "calendar days") + " after the date of shipment, " + shipment.Human() + " (" + source + ")"
}

func expiryReason(expiry Date, extended bool) string {
	if extended {
		return "the expiry date, " + expiry.Human() +
			", extended to the first following banking day because the bank is closed on it (art. 29(a)) — note that art. 29(c) does not extend the latest shipment date with it"
	}
	return "the expiry date of the credit, " + expiry.Human() + " (art. 6(e))"
}

// Report renders a result as plain text, which is what an exporter sends their
// banker or keeps with the file. It is written here rather than in the web
// layer so that it is covered by the same tests as the rules.
func Report(c Credit, p Presentation, r Result) string {
	c = c.Normalise()
	var b strings.Builder

	line := func(s string) { b.WriteString(s + "\n") }
	rule := func() { line(strings.Repeat("-", 68)) }

	line("LETTER OF CREDIT — PRE-PRESENTATION SELF-CHECK")
	rule()
	if c.Reference != "" {
		line("Credit            " + c.Reference)
	}
	if c.IssuingBank != "" {
		line("Issuing bank      " + c.IssuingBank)
	}
	if c.Amount > 0 {
		line("Credit amount     " + Money(c.Currency, c.Amount, r.Thresholds.AmountDecimals))
	}
	if !c.ExpiryDate.Zero() {
		line("Expires           " + c.ExpiryDate.Human() + " at " + orDash(c.ExpiryPlace))
	}
	if !c.LatestShipmentDate.Zero() {
		line("Latest shipment   " + c.LatestShipmentDate.Human())
	}
	if !p.Date.Zero() {
		line("Presenting on     " + p.Date.Human())
	}
	line("Checked on        " + r.CheckedOn.Human())
	line("")

	line("VERDICT: " + r.Verdict)
	line("")
	if !r.ShipmentDate.Zero() {
		line("Date of shipment taken as " + r.ShipmentDate.Human())
		line("  because it is " + r.ShipmentDateReason)
	}
	if !r.LastPresentationDay.Zero() {
		line("Last day to present       " + r.LastPresentationDay.Human())
		line("  because it is " + r.LastPresentationWhy)
	}
	if !r.AnswerExpectedBy.Zero() {
		line("Answer due from the bank  by the close of " + r.AnswerExpectedBy.Human() +
			" (" + plural(r.Thresholds.ExaminationBankingDays, "banking day", "banking days") + " after presentation, arts. 14(b), 16(d))")
	}
	line("")

	group := ""
	for _, f := range r.Findings {
		if f.Severity.Label() != group {
			group = f.Severity.Label()
			rule()
			line(strings.ToUpper(group))
			rule()
		}
		line("[" + f.Rule + "] " + f.Title)
		line("    " + f.Detail)
		if len(f.Documents) > 0 {
			line("    Documents: " + strings.Join(f.Documents, "; "))
		}
		line("    Rule: " + f.Citation)
		if f.Fix != "" {
			line("    Do: " + f.Fix)
		}
		line("")
	}

	if len(r.NotChecked) > 0 {
		rule()
		line("NOT CHECKED — nothing was entered for these")
		rule()
		for _, g := range r.NotChecked {
			line("[" + g.Rule + "] needs " + g.Need)
		}
		line("")
	}

	rule()
	line("FIGURES APPLIED")
	rule()
	for _, cite := range r.Thresholds.Citations() {
		line(pad(cite.Key, 26) + pad(cite.Figure, 34) + cite.Source)
	}
	line("")
	rule()
	line("This is a self-check made before presentation, from what was typed in.")
	line("It is not a bank's examination and it is not advice. Only the issuing")
	line("bank's determination on the documents themselves decides whether the")
	line("presentation complies (UCP 600 arts. 14(a), 15 and 16). Confirm every")
	line("article and figure above against the current text of UCP 600.")
	return b.String()
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s + " "
	}
	return s + strings.Repeat(" ", n-len(s))
}
