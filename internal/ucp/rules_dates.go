package ucp

import "strings"

// dateRules are the checks on the deadlines. More presentations are refused on
// these than on anything else, and all of them are free to fix before the
// documents go in.
func dateRules() []rule {
	return []rule{
		{
			ID:       "D1-EXPIRY",
			Group:    groupDates,
			Title:    "Presentation within the expiry date",
			Citation: "UCP 600 arts. 6(e), 29(a), 29(b)",
			Tests:    "That you are presenting on or before the expiry date, with the extension article 29(a) gives when the bank is closed on that day.",
			eval:     ruleExpiry,
		},
		{
			ID:       "D2-LATE-SHIPMENT",
			Group:    groupDates,
			Title:    "Shipment on or before the latest shipment date",
			Citation: "UCP 600 arts. 3, 29(c)",
			Tests:    "That the date of shipment is not after the latest shipment date, which article 29(c) never extends even when the expiry date is extended.",
			eval:     ruleLateShipment,
		},
		{
			ID:       "D3-STALE",
			Group:    groupDates,
			Title:    "Presentation within the period after shipment",
			Citation: "UCP 600 art. 14(c)",
			Tests:    "That the documents go in within 21 calendar days of shipment, or the period the credit states — and that the rule is applied only where a transport document under articles 19 to 25 is presented.",
			eval:     ruleStale,
		},
		{
			ID:       "D4-ON-OR-ABOUT",
			Group:    groupDates,
			Title:    "Shipment inside an \"on or about\" window",
			Citation: "UCP 600 art. 3",
			Tests:    "Where the credit words shipment as \"on or about\" a date, that shipment falls in the five days either side, both days included.",
			eval:     ruleOnOrAbout,
		},
		{
			ID:       "D5-DOC-DATES",
			Group:    groupDates,
			Title:    "No document dated after presentation",
			Citation: "UCP 600 art. 14(i)",
			Tests:    "That no document is dated later than the day it is presented, and clears a document dated before the credit was issued, which that sub-article expressly allows.",
			eval:     ruleDocumentDates,
		},
		{
			ID:       "D6-INSURANCE-DATE",
			Group:    groupDates,
			Title:    "Insurance in force from shipment",
			Citation: "UCP 600 art. 28(e)",
			Tests:    "That the insurance document is not dated after the date of shipment, unless it shows cover running from a date no later than shipment.",
			eval:     ruleInsuranceDate,
		},
		{
			ID:       "D7-MATURITY",
			Group:    groupDates,
			Title:    "Maturity date of a usance draft",
			Citation: "UCP 600 art. 3",
			Tests:    "Works out when a term draft matures, counting as article 3 does: \"from\" and \"after\", used to determine a maturity date, exclude the date mentioned.",
			eval:     ruleMaturity,
		},
	}
}

func ruleExpiry(e *exam) []Finding {
	if e.C.ExpiryDate.Zero() {
		return e.skip("the credit's expiry date")
	}
	if e.P.Date.Zero() {
		return e.skip("the date you will present the documents")
	}

	last, extended := e.T.LastDayForPresentation(e.C.ExpiryDate, e.C.BankClosedDays)
	var out []Finding

	if e.P.Date.After(last) {
		late := DaysBetween(last, e.P.Date)
		out = append(out, e.find(SeverityDiscrepancy,
			"Presented after the credit has expired",
			"You are presenting on "+e.P.Date.Human()+", which is "+plural(late, "day", "days")+
				" after the last day for presentation, "+last.Human()+". A presentation by or on behalf of the beneficiary must be made on or before the expiry date.",
			"UCP 600 art. 6(e)",
			"Present on or before "+last.Human()+", or ask the applicant to have the credit amended to extend the expiry date before you present. Nothing in the rules lets a bank waive this on its own.",
		))
		return out
	}

	if extended && e.P.Date.After(e.C.ExpiryDate) {
		out = append(out, e.find(SeverityCleared,
			"Presenting after the stated expiry date, but inside the extension",
			"The credit expires on "+e.C.ExpiryDate.Human()+", a day the bank is closed, so the last day for presentation moves to the first following banking day, "+
				last.Human()+". Presenting on "+e.P.Date.Human()+" is within time.",
			"UCP 600 art. 29(a)",
			"If a nominated bank presents on that extended day, it must state on its covering schedule that the presentation was made within the time limits extended under article 29(a) — ask your bank to include that statement.",
		))
		return out
	}

	left := DaysBetween(e.P.Date, last)
	sev, title := SeverityCleared, "Inside the expiry date"
	fix := ""
	if left <= 2 {
		sev, title = SeverityCaution, "Barely inside the expiry date"
		fix = "Hand the documents in over the counter rather than by courier, and keep the bank's dated receipt. A day lost in transit cannot be recovered."
	}
	out = append(out, e.find(sev, title,
		"Presenting on "+e.P.Date.Human()+", with "+plural(left, "day", "days")+" to spare before the last day for presentation, "+last.Human()+".",
		"UCP 600 art. 6(e)", fix))
	return out
}

func ruleLateShipment(e *exam) []Finding {
	if e.C.LatestShipmentDate.Zero() {
		return e.skip("the latest shipment date the credit allows")
	}
	if !e.hasShipment {
		return e.skip("a transport document, so that the date of shipment can be worked out")
	}

	if e.shipment.After(e.C.LatestShipmentDate) {
		late := DaysBetween(e.C.LatestShipmentDate, e.shipment)
		detail := "Shipment is dated " + e.shipment.Human() + " — " + plural(late, "day", "days") +
			" after the latest shipment date of " + e.C.LatestShipmentDate.Human() + ". That date is taken from " + e.shipmentWhy + "."
		if _, extended := e.T.LastDayForPresentation(e.C.ExpiryDate, e.C.BankClosedDays); extended {
			detail += " The extension that moved the last day for presentation does not help here: article 29(c) says the latest date for shipment is not extended as a result of article 29(a)."
		}
		return []Finding{e.find(SeverityDiscrepancy,
			"Shipped after the latest shipment date",
			detail,
			"UCP 600 arts. 29(c), 14(a)",
			"This one cannot be papered over. Either obtain an amendment extending the latest shipment date before presenting, or present knowing the discrepancy and ask the applicant in advance to waive it — a waiver is the applicant's and the issuing bank's to give, not yours to assume.",
			e.shipmentDocs()...,
		)}
	}

	spare := DaysBetween(e.shipment, e.C.LatestShipmentDate)
	if spare == 0 {
		return []Finding{e.find(SeverityCleared,
			"Shipped on the last day allowed",
			"Shipment is dated "+e.shipment.Human()+", the latest shipment date itself. The words \"to\", \"until\", \"till\" and \"between\", used to determine a period of shipment, include the date mentioned, so the day itself is within time.",
			"UCP 600 art. 3", "", e.shipmentDocs()...)}
	}
	return []Finding{e.find(SeverityCleared,
		"Shipped within the latest shipment date",
		"Shipment is dated "+e.shipment.Human()+", "+plural(spare, "day", "days")+" before the latest shipment date of "+e.C.LatestShipmentDate.Human()+".",
		"UCP 600 art. 14(a)", "", e.shipmentDocs()...)}
}

func ruleStale(e *exam) []Finding {
	if e.P.Date.Zero() {
		return e.skip("the date you will present the documents")
	}
	if !e.hasShipment {
		return e.skip("a transport document, so that the date of shipment can be worked out")
	}

	stated := e.C.PresentationPeriodDays > 0
	period := e.C.PresentationPeriodDays
	if !stated {
		period = e.T.PresentationPeriodDays
	}
	governed := e.P.HasTransportUnderArticles19To25()

	if !governed && !stated {
		return []Finding{e.find(SeverityCleared,
			"The 21-day rule does not reach this presentation",
			"No transport document subject to articles 19 to 25 is being presented, and the credit states no period of its own. The 21-day limit in article 14(c) applies only to a presentation that includes one or more original transport documents under those articles, so the only deadline is the expiry date.",
			"UCP 600 art. 14(c)",
			"Satisfy yourself that what you are presenting really is outside articles 19 to 25 — a forwarder's certificate of receipt is, a house bill of lading that meets article 20 is not.",
		)}
	}

	deadline := e.shipment.AddDays(period)
	source := "article 14(c)"
	if stated {
		source = "the period the credit states"
	}
	days := DaysBetween(e.shipment, e.P.Date)

	if e.P.Date.After(deadline) {
		return []Finding{e.find(SeverityDiscrepancy,
			"Documents presented too long after shipment",
			"Shipment is dated "+e.shipment.Human()+" and you are presenting on "+e.P.Date.Human()+" — "+plural(days, "calendar day", "calendar days")+
				" later, against the "+plural(period, "day", "days")+" allowed by "+source+". The deadline was "+deadline.Human()+".",
			"UCP 600 art. 14(c)",
			"There is no cure after the day has passed; the documents are stale. Present as soon as the set is complete next time, and where a courier will eat the days, arrange for the shipping documents to be released to you earlier.",
			e.shipmentDocs()...,
		)}
	}

	left := DaysBetween(e.P.Date, deadline)
	sev, title, fix := SeverityCleared, "Presented in time after shipment", ""
	if left <= 2 {
		sev, title = SeverityCaution, "Close to the deadline for presentation after shipment"
		fix = "Present over the counter. Article 14(c) counts calendar days, so a weekend costs you two of them."
	}
	return []Finding{e.find(sev, title,
		"Presenting "+plural(days, "calendar day", "calendar days")+" after shipment, against the "+plural(period, "day", "days")+" allowed by "+source+
			". The last day on that count is "+deadline.Human()+".",
		"UCP 600 art. 14(c)", fix, e.shipmentDocs()...)}
}

func ruleOnOrAbout(e *exam) []Finding {
	if e.C.ShipmentOnOrAbout.Zero() {
		return e.skip("an \"on or about\" shipment date in the credit (left blank where the credit words it some other way)")
	}
	if !e.hasShipment {
		return e.skip("a transport document, so that the date of shipment can be worked out")
	}

	window := e.T.OnOrAboutWindow(e.C.ShipmentOnOrAbout)
	if window.Contains(e.shipment) {
		return []Finding{e.find(SeverityCleared,
			"Shipment inside the \"on or about\" window",
			"The credit calls for shipment on or about "+e.C.ShipmentOnOrAbout.Human()+", which is read as the period "+window.String()+
				" — "+plural(e.T.OnOrAboutDays, "calendar day", "calendar days")+" either side, both days included. Shipment is dated "+e.shipment.Human()+".",
			"UCP 600 art. 3", "", e.shipmentDocs()...)}
	}
	return []Finding{e.find(SeverityDiscrepancy,
		"Shipment outside the \"on or about\" window",
		"The credit calls for shipment on or about "+e.C.ShipmentOnOrAbout.Human()+", which is the period "+window.String()+
			". Shipment is dated "+e.shipment.Human()+", outside it.",
		"UCP 600 art. 3",
		"Nothing on the documents can move the date of shipment. Seek an amendment, or an advance waiver from the applicant.",
		e.shipmentDocs()...)}
}

func ruleDocumentDates(e *exam) []Finding {
	if e.P.Date.Zero() {
		return e.skip("the date you will present the documents")
	}
	dated := 0
	for _, d := range e.P.Documents {
		if !d.Date.Zero() {
			dated++
		}
	}
	if dated == 0 {
		return e.skip("a date on at least one of your documents")
	}

	var out []Finding
	var early []string
	for _, d := range e.P.Documents {
		if d.Date.Zero() {
			continue
		}
		if d.Date.After(e.P.Date) {
			out = append(out, e.find(SeverityDiscrepancy,
				"A document is dated after the day it is presented",
				d.Label()+" is dated "+d.Date.Human()+", later than the presentation date of "+e.P.Date.Human()+
					". A document may be dated before the credit was issued, but it must not be dated later than its date of presentation.",
				"UCP 600 art. 14(i)",
				"Re-date the document, or present on or after the date it bears. A post-dated certificate is one of the easiest discrepancies for a bank to spot and one of the hardest to argue away.",
				d.Label()))
		}
		if !e.C.IssueDate.Zero() && d.Date.Before(e.C.IssueDate) {
			early = append(early, d.Label()+" ("+d.Date.Human()+")")
		}
	}

	if len(early) > 0 {
		out = append(out, e.find(SeverityCleared,
			"A document dated before the credit was issued is not a discrepancy",
			"The credit was issued on "+e.C.IssueDate.Human()+" and "+strings.Join(early, ", ")+
				" "+isAre(len(early))+" dated earlier. Article 14(i) says in terms that a document may be dated prior to the issuance date of the credit.",
			"UCP 600 art. 14(i)",
			"Do not have the document reissued for this reason alone. Check instead that the credit itself does not call for a document dated after a particular day.",
			early...))
	}

	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared,
			"Every document is dated on or before the presentation date",
			plural(dated, "document carries a date", "documents carry a date")+", and none of them is later than "+e.P.Date.Human()+".",
			"UCP 600 art. 14(i)", ""))
	}
	return out
}

func ruleInsuranceDate(e *exam) []Finding {
	ins, ok := e.P.First(KindInsurance)
	if !ok {
		return e.skip("an insurance document in the presentation")
	}
	if !e.hasShipment {
		return e.skip("a transport document, so that the date of shipment can be worked out")
	}
	if ins.Date.Zero() && (ins.Insurance == nil || ins.Insurance.CoverEffectiveFrom.Zero()) {
		return e.skip("the date on the insurance document")
	}

	effective := Date{}
	if ins.Insurance != nil {
		effective = ins.Insurance.CoverEffectiveFrom
	}

	if !ins.Date.Zero() && ins.Date.After(e.shipment) {
		if !effective.Zero() && effective.OnOrBefore(e.shipment) {
			return []Finding{e.find(SeverityCleared,
				"Insurance issued after shipment, but cover runs from before it",
				ins.Label()+" is dated "+ins.Date.Human()+", after the date of shipment ("+e.shipment.Human()+
					"), but it shows cover effective from "+effective.Human()+". Article 28(e) allows exactly that.",
				"UCP 600 art. 28(e)",
				"Make sure the words on the document really do say cover is effective from that earlier date — the exception depends on it appearing from the insurance document itself.",
				ins.Label())}
		}
		return []Finding{e.find(SeverityDiscrepancy,
			"Insurance dated after shipment",
			ins.Label()+" is dated "+ins.Date.Human()+", after the date of shipment ("+e.shipment.Human()+
				"), and does not show cover effective from a date no later than shipment. The goods would have sailed uninsured.",
			"UCP 600 art. 28(e)",
			"Ask the insurer to reissue the certificate dated on or before the shipment date, or to endorse it as covering risks from a date not later than the date of shipment.",
			ins.Label())}
	}

	when := ins.Date
	if when.Zero() {
		when = effective
	}
	return []Finding{e.find(SeverityCleared,
		"Insurance in force by the date of shipment",
		ins.Label()+" is dated "+when.Human()+", on or before the date of shipment ("+e.shipment.Human()+").",
		"UCP 600 art. 28(e)", "", ins.Label())}
}

func ruleMaturity(e *exam) []Finding {
	switch e.C.Availability {
	case AvailabilityAcceptance, AvailabilityDeferred:
	default:
		return e.skip("a credit available by acceptance or by deferred payment (a sight credit has no maturity to work out)")
	}
	if e.C.DraftTenorDays <= 0 {
		return e.skip("the tenor the credit states, in days")
	}

	var base Date
	var basis string
	switch e.C.DraftTenorBasis {
	case TenorFromShipment:
		if !e.hasShipment {
			return e.skip("a transport document, so that the date of shipment can be worked out")
		}
		base, basis = e.shipment, "the date of shipment"
	case TenorFromInvoice:
		if !e.hasInvoice || e.invoice.Date.Zero() {
			return e.skip("the date on the commercial invoice")
		}
		base, basis = e.invoice.Date, "the date of the invoice"
	case TenorFromPresentation, TenorFromSight:
		if e.P.Date.Zero() {
			return e.skip("the date you will present the documents")
		}
		base, basis = e.P.Date, "the date of presentation, standing in for sight"
	default:
		return e.skip("what the credit counts the tenor from — shipment, invoice date or sight")
	}

	maturity := base.AddDays(e.C.DraftTenorDays)
	detail := "A tenor of " + plural(e.C.DraftTenorDays, "day", "days") + " counted from " + basis + " (" + base.Human() +
		") matures on " + maturity.Human() + ". Article 3 settles the counting: the words \"from\" and \"after\", when used to determine a maturity date, exclude the date mentioned, so the day counted from is not day one."
	fix := "Check this against the maturity your bank calculates before you accept the draft. If the credit is available by deferred payment there is no draft to accept, and the date is the one the issuing bank undertakes to pay on."
	if e.C.DraftTenorBasis == TenorFromSight {
		fix = "A sight tenor runs from the bank's acceptance, not from your presentation. The date above uses your presentation date only as a guide."
	}
	return []Finding{e.find(SeverityVerify, "Maturity date on the figures entered", detail, "UCP 600 art. 3", fix)}
}

// shipmentDocs names the transport documents a date finding is about.
func (e *exam) shipmentDocs() []string {
	var out []string
	for _, d := range e.P.TransportDocuments() {
		out = append(out, d.Label())
	}
	return out
}

func isAre(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}
