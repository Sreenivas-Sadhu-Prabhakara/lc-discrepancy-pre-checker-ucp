package ucp

import "strings"

// amountRules are the checks on figures: what may be drawn, what may be
// shipped, what must be insured. The tolerances in article 30 are narrower and
// more conditional than they are usually remembered, and most of the work here
// is applying them as written rather than as remembered.
func amountRules() []rule {
	return []rule{
		{
			ID:       "A1-AMOUNT",
			Group:    groupAmounts,
			Title:    "Amount drawn within the credit",
			Citation: "UCP 600 arts. 30(a), 18(b)",
			Tests:    "That the amount claimed is not above the credit amount as any tolerance the credit or article 30(a) allows extends it.",
			eval:     ruleAmountCeiling,
		},
		{
			ID:       "A2-SHORT-DRAW",
			Group:    groupAmounts,
			Title:    "Short drawing where partial shipments are forbidden",
			Citation: "UCP 600 art. 30(c)",
			Tests:    "That a drawing for less than the credit amount is one the rules allow — the 5% in article 30(c) runs downward only and only on its own conditions.",
			eval:     ruleShortDraw,
		},
		{
			ID:       "A3-QUANTITY",
			Group:    groupAmounts,
			Title:    "Quantity shipped within tolerance",
			Citation: "UCP 600 arts. 30(a), 30(b)",
			Tests:    "That the quantity is inside the tolerance that actually applies — not 5% whenever the mood takes, but 5% only where the credit does not count the goods in packing units and the drawings stay within the credit amount.",
			eval:     ruleQuantity,
		},
		{
			ID:       "A4-CURRENCY",
			Group:    groupAmounts,
			Title:    "One currency across the credit and the documents",
			Citation: "UCP 600 arts. 18(a)(iii), 28(f)(i)",
			Tests:    "That the invoice and the insurance document are in the credit's currency, and that the draft and the drawing are too.",
			eval:     ruleCurrency,
		},
		{
			ID:       "A5-ARITHMETIC",
			Group:    groupAmounts,
			Title:    "Invoice arithmetic and unit price",
			Citation: "UCP 600 arts. 30(a), 30(c), 14(d)",
			Tests:    "That quantity times unit price comes to the invoice total, and that the unit price is the credit's — a reduced unit price also costs you the short-drawing tolerance in article 30(c).",
			eval:     ruleArithmetic,
		},
		{
			ID:       "A6-INSURANCE-AMOUNT",
			Group:    groupAmounts,
			Title:    "Insured amount at least the required percentage",
			Citation: "UCP 600 art. 28(f)(ii)",
			Tests:    "That the sum insured meets the percentage the credit requires, or 110% of the CIF or CIP value where the credit is silent — with the fallback the sub-article gives when no CIF or CIP value can be worked out from the documents.",
			eval:     ruleInsuranceAmount,
		},
		{
			ID:       "A7-DRAFT",
			Group:    groupAmounts,
			Title:    "Draft agrees with the drawing",
			Citation: "UCP 600 arts. 6(b), 14(d)",
			Tests:    "That a draft is presented where the credit calls for one, for the amount being claimed and on the tenor the credit states.",
			eval:     ruleDraft,
		},
	}
}

func ruleAmountCeiling(e *exam) []Finding {
	if e.C.Amount <= 0 {
		return e.skip("the credit amount")
	}
	if !e.hasDrawn {
		return e.skip("the amount you are drawing, or an invoice total")
	}

	tol := AmountTolerance(e.C, e.T)
	_, high := tol.Band(e.C.Amount, e.T.AmountDecimals)

	if AmountExceeds(e.drawn, high, e.T.AmountDecimals) {
		over := e.drawn - high
		detail := "You are claiming " + e.money(e.drawn) + " against a credit for " + e.money(e.C.Amount) +
			". The ceiling is " + e.money(high) + " (" + tol.Source + "), so the claim is over by " + e.money(over) + "."
		fix := "Reduce the invoice and the drawing to " + e.money(high) + " or less. Article 18(b) does let a bank accept an invoice made out for more than the credit permits, but only if it does not honour or negotiate for more than the credit allows — so the excess is not money you will see, and the invoice is a discrepancy until the bank says otherwise."
		return []Finding{e.find(SeverityDiscrepancy, "Claiming more than the credit allows", detail, "UCP 600 art. 30(a)", fix, e.invoiceLabel()...)}
	}

	headroom := high - e.drawn
	return []Finding{e.find(SeverityCleared, "Amount claimed within the credit",
		"Claiming "+e.money(e.drawn)+" against a credit for "+e.money(e.C.Amount)+", with "+e.money(headroom)+
			" of headroom to the ceiling of "+e.money(high)+" ("+tol.Source+").",
		"UCP 600 art. 30(a)", "", e.invoiceLabel()...)}
}

func ruleShortDraw(e *exam) []Finding {
	if e.C.Amount <= 0 {
		return e.skip("the credit amount")
	}
	if !e.hasDrawn {
		return e.skip("the amount you are drawing, or an invoice total")
	}
	if !AmountBelow(e.drawn, e.C.Amount, e.T.AmountDecimals) {
		return e.skip("a drawing for less than the credit amount (nothing to test — you are drawing the full amount)")
	}

	short := e.C.Amount - e.drawn
	shortPct := PercentOf(short, e.C.Amount)

	// Partial drawings are allowed unless the credit forbids them (article
	// 31(a)), so a short drawing under a credit that permits partial shipment
	// is simply a partial drawing.
	if e.C.PartialShipments != PermissionProhibited {
		return []Finding{e.find(SeverityCleared, "Drawing less than the full amount is a partial drawing",
			"You are claiming "+e.money(e.drawn)+" of a credit for "+e.money(e.C.Amount)+". Partial drawings and shipments are allowed unless the credit says otherwise, and this credit does not forbid them.",
			"UCP 600 art. 31(a)",
			"Keep enough of the credit for the balance of the order, and watch the expiry date — the credit does not reopen.", e.invoiceLabel()...)}
	}

	// Partial shipments are forbidden, so a short drawing has to find a
	// tolerance somewhere. The last sentence of article 30(c) decides where: its
	// own 5% "does not apply when the credit stipulates a specific tolerance or
	// uses the expressions referred to in sub-article 30(a)". A credit that
	// states a tolerance of its own, or that words the amount as "about", is
	// therefore governed by that figure and not by article 30(c) — and the
	// conditions article 30(c) attaches do not travel with it.
	tol := AmountTolerance(e.C, e.T)
	ownTolerance := e.C.TolerancePlusPercent > 0 || e.C.ToleranceMinusPercent > 0
	about := NormaliseQualifier(e.C.AmountQualifier) == QualifierAbout

	allowed := e.T.ShortDrawTolerancePercent
	source := "UCP 600 art. 30(c)"
	var blockers []string

	switch {
	case ownTolerance:
		allowed = tol.MinusPercent
		source = "the tolerance the credit states"
	case about:
		allowed = tol.MinusPercent
		source = "UCP 600 art. 30(a), the credit wording the amount as \"about\""
	default:
		// Article 30(c) proper, with its conditions. The third of them is the one
		// most often read backwards: the tolerance does not apply where article
		// 30(b) is applicable. Article 30(b) does not reach goods the credit
		// counts in packing units or individual items, so goods counted that way
		// satisfy the condition rather than breaking it, and nothing is added to
		// the list below for them.
		if shippedShort, known := e.quantityShortOfCredit(); known && shippedShort {
			blockers = append(blockers, "the quantity of the goods has not been shipped in full")
		}
		if reduced, known := e.unitPriceReduced(); known && reduced {
			blockers = append(blockers, "the unit price stated in the credit has been reduced")
		}
	}

	switch {
	case len(blockers) > 0:
		return []Finding{e.find(SeverityDiscrepancy, "Short drawing where partial shipments are not allowed",
			"Partial shipments are prohibited and you are claiming "+e.money(e.drawn)+" of "+e.money(e.C.Amount)+
				" — "+percentText(RoundTo(shortPct, 2))+" less. The tolerance in article 30(c) is not available here because "+strings.Join(blockers, ", and ")+".",
			"UCP 600 art. 30(c)",
			"Either ship and invoice the credit in full, or obtain an amendment. A short drawing under a credit that forbids partial shipment is refused on its face.",
			e.invoiceLabel()...)}
	case shortPct > allowed:
		return []Finding{e.find(SeverityDiscrepancy, "Short drawing beyond the tolerance",
			"Partial shipments are prohibited and you are claiming "+e.money(e.drawn)+" of "+e.money(e.C.Amount)+
				" — "+percentText(RoundTo(shortPct, 2))+" less, against the "+percentText(allowed)+" that "+source+" allows downward.",
			"UCP 600 art. 30(c)",
			"Bring the drawing to at least "+e.money(RoundTo(e.C.Amount*(1-allowed/100), e.T.AmountDecimals))+", or obtain an amendment reducing the credit amount.",
			e.invoiceLabel()...)}
	default:
		why := "where the full quantity has been shipped and the unit price has not been reduced"
		advice := "The tolerance in article 30(c) runs one way only — there is no matching 5% upward — and it disappears the moment the quantity is short or the unit price is cut."
		if ownTolerance || about {
			why = "under " + source
			advice = "Article 30(c)'s own 5% does not apply here, because its last sentence gives way to a tolerance the credit states and to an amount worded as \"about\". The credit's figure is the one that governs."
		}
		return []Finding{e.find(SeverityCleared, "Short drawing inside the tolerance that applies",
			"Partial shipments are prohibited, but a drawing of up to "+percentText(allowed)+" less than the credit amount is allowed "+why+
				". You are "+percentText(RoundTo(shortPct, 2))+" short, which is inside it.",
			source, advice,
			e.invoiceLabel()...)}
	}
}

func ruleQuantity(e *exam) []Finding {
	if e.C.Quantity <= 0 {
		return e.skip("the quantity the credit calls for")
	}
	shipped, unit, known := e.shippedQuantity()
	if !known {
		return e.skip("the quantity on your invoice, packing list or transport document")
	}

	drawingsExceed := e.hasDrawn && e.C.Amount > 0 && AmountExceeds(e.drawn, e.C.Amount, e.T.AmountDecimals)
	tol := QuantityTolerance(e.C, drawingsExceed, e.T)
	low, high := tol.Band(e.C.Quantity, 3)

	unitNote := ""
	if unit != "" && e.C.QuantityUnit != "" && Compare(e.C.QuantityUnit, unit, e.T) == MatchConflict {
		unitNote = " The credit counts the goods in " + e.C.QuantityUnit + " and the documents in " + unit +
			"; the comparison below assumes they are the same measure, which you must confirm."
	}

	switch {
	case shipped > high:
		return []Finding{e.find(SeverityDiscrepancy, "More shipped than the credit allows",
			"The credit calls for "+Quantity(e.C.Quantity, e.C.QuantityUnit)+" and the documents show "+Quantity(shipped, unit)+
				". The ceiling is "+Quantity(high, e.C.QuantityUnit)+" — tolerance: "+tol.Source+"."+unitNote,
			"UCP 600 art. 30(b)",
			"Invoice only the quantity the credit allows, or obtain an amendment. Shipping more than the credit permits also risks the drawing exceeding the credit amount, which is a second discrepancy.",
			e.quantityDocs()...)}
	case shipped < low:
		return []Finding{e.find(SeverityDiscrepancy, "Less shipped than the credit allows",
			"The credit calls for "+Quantity(e.C.Quantity, e.C.QuantityUnit)+" and the documents show "+Quantity(shipped, unit)+
				". The floor is "+Quantity(low, e.C.QuantityUnit)+" — tolerance: "+tol.Source+"."+unitNote,
			"UCP 600 art. 30(b)",
			"Where partial shipment is allowed, present this as a partial drawing and keep the balance for the next shipment. Where it is not, ship the full quantity or obtain an amendment.",
			e.quantityDocs()...)}
	case shipped == e.C.Quantity:
		return []Finding{e.find(SeverityCleared, "Quantity shipped as the credit calls for",
			"The credit calls for "+Quantity(e.C.Quantity, e.C.QuantityUnit)+" and the documents show the same."+unitNote,
			"UCP 600 art. 14(a)", "", e.quantityDocs()...)}
	default:
		return []Finding{e.find(SeverityCleared, "Quantity inside the tolerance that applies",
			"The credit calls for "+Quantity(e.C.Quantity, e.C.QuantityUnit)+" and the documents show "+Quantity(shipped, unit)+
				", inside "+Quantity(low, "")+" to "+Quantity(high, e.C.QuantityUnit)+" — tolerance: "+tol.Source+"."+unitNote,
			"UCP 600 art. 30(b)",
			"Remember that this tolerance is also conditional on the total of the drawings staying within the credit amount.",
			e.quantityDocs()...)}
	}
}

func ruleCurrency(e *exam) []Finding {
	if e.C.Currency == "" {
		return e.skip("the credit's currency")
	}

	var out []Finding
	checked := 0

	if e.hasInvoice && e.invoice.Invoice != nil && e.invoice.Invoice.Currency != "" {
		checked++
		if e.invoice.Invoice.Currency != e.C.Currency {
			out = append(out, e.find(SeverityDiscrepancy, "Invoice in the wrong currency",
				"The credit is in "+e.C.Currency+" and "+e.invoice.Label()+" is made out in "+e.invoice.Invoice.Currency+
					". A commercial invoice must be made out in the same currency as the credit.",
				"UCP 600 art. 18(a)(iii)",
				"Reissue the invoice in "+e.C.Currency+". A conversion note on the invoice does not cure it.",
				e.invoice.Label()))
		}
	}

	if ins, ok := e.P.First(KindInsurance); ok && ins.Insurance != nil && ins.Insurance.Currency != "" {
		checked++
		if ins.Insurance.Currency != e.C.Currency {
			out = append(out, e.find(SeverityDiscrepancy, "Insurance in the wrong currency",
				"The credit is in "+e.C.Currency+" and "+ins.Label()+" insures in "+ins.Insurance.Currency+
					". The insurance document must indicate the amount of insurance coverage and be in the same currency as the credit.",
				"UCP 600 art. 28(f)(i)",
				"Ask the insurer to reissue in "+e.C.Currency+".", ins.Label()))
		}
	}

	if dr, ok := e.P.First(KindDraft); ok && dr.Draft != nil && dr.Draft.Currency != "" {
		checked++
		if dr.Draft.Currency != e.C.Currency {
			out = append(out, e.find(SeverityDiscrepancy, "Draft in the wrong currency",
				"The credit is in "+e.C.Currency+" and the draft is drawn in "+dr.Draft.Currency+
					", which conflicts with the credit.",
				"UCP 600 art. 14(d)", "Redraw the draft in "+e.C.Currency+".", dr.Label()))
		}
	}

	if e.P.DrawnCurrency != "" {
		checked++
		if e.P.DrawnCurrency != e.C.Currency {
			out = append(out, e.find(SeverityDiscrepancy, "Drawing in the wrong currency",
				"The credit is in "+e.C.Currency+" and the drawing is stated in "+e.P.DrawnCurrency+".",
				"UCP 600 art. 14(d)", "Draw in "+e.C.Currency+"."))
		}
	}

	if checked == 0 {
		return e.skip("a currency on at least one of your documents")
	}
	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "Every figure in the credit's currency",
			"The credit is in "+e.C.Currency+" and so is everything entered.",
			"UCP 600 art. 18(a)(iii)", ""))
	}
	return out
}

func ruleArithmetic(e *exam) []Finding {
	if !e.hasInvoice || e.invoice.Invoice == nil {
		return e.skip("a commercial invoice")
	}
	inv := e.invoice.Invoice
	if inv.Amount <= 0 || inv.Quantity <= 0 || inv.UnitPrice <= 0 {
		return e.skip("the quantity, unit price and total on the invoice")
	}

	var out []Finding
	expected := RoundTo(inv.Quantity*inv.UnitPrice, e.T.AmountDecimals)
	if !AmountsEqual(expected, inv.Amount, e.T.AmountDecimals) {
		out = append(out, e.find(SeverityCaution, "Invoice does not add up",
			Quantity(inv.Quantity, inv.Unit)+" at "+e.money(inv.UnitPrice)+" comes to "+e.money(expected)+
				", but the invoice total is "+e.money(inv.Amount)+" — a difference of "+e.money(expected-inv.Amount)+
				". A bank checks the totals of a document; whether it works through every line is standard banking practice rather than a rule of UCP 600, and practice does not require it to. Either way the figure conflicts with itself, and a figure that conflicts is a discrepancy under article 14(d).",
			"UCP 600 art. 14(d)",
			"Recheck the invoice. If the difference is freight or insurance shown separately, set that out as its own line so the arithmetic reads.",
			e.invoice.Label()))
	}

	if e.C.UnitPrice > 0 {
		tol := Tolerance{Source: "none — the credit states the unit price exactly"}
		if NormaliseQualifier(e.C.UnitPriceQualifier) == QualifierAbout {
			tol = Tolerance{
				PlusPercent:  e.T.AboutTolerancePercent,
				MinusPercent: e.T.AboutTolerancePercent,
				Source:       percentText(e.T.AboutTolerancePercent) + " either way because the credit says \"about\" the unit price — UCP 600 art. 30(a)",
			}
		}
		low, high := tol.Band(e.C.UnitPrice, 4)
		switch {
		case inv.UnitPrice > high || inv.UnitPrice < low:
			out = append(out, e.find(SeverityDiscrepancy, "Unit price is not the credit's",
				"The credit states a unit price of "+e.money(e.C.UnitPrice)+" and the invoice shows "+e.money(inv.UnitPrice)+
					". Tolerance: "+tol.Source+".",
				"UCP 600 arts. 14(d), 30(a)",
				"Invoice at the credit's unit price. A reduced unit price also costs you the short-drawing tolerance in article 30(c), which is only available while the credit's unit price is not reduced.",
				e.invoice.Label()))
		case inv.UnitPrice < e.C.UnitPrice:
			out = append(out, e.find(SeverityCaution, "Unit price lower than the credit's",
				"The credit states "+e.money(e.C.UnitPrice)+" a unit and the invoice shows "+e.money(inv.UnitPrice)+
					", which the tolerance covers. Note the side effect: article 30(c) allows a short drawing where partial shipments are forbidden only while the unit price stated in the credit is not reduced.",
				"UCP 600 art. 30(c)",
				"If you are also drawing short under a credit that forbids partial shipment, raise the unit price back to the credit's or expect the short drawing to be refused.",
				e.invoice.Label()))
		}
	}

	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "Invoice adds up and prices as the credit does",
			Quantity(inv.Quantity, inv.Unit)+" at "+e.money(inv.UnitPrice)+" comes to the invoice total of "+e.money(inv.Amount)+".",
			"UCP 600 art. 14(d)", "", e.invoice.Label()))
	}
	return out
}

func ruleInsuranceAmount(e *exam) []Finding {
	ins, ok := e.P.First(KindInsurance)
	if !ok {
		if e.requiresInsurance() {
			return []Finding{e.find(SeverityDiscrepancy, "No insurance document presented",
				"The credit calls for insurance and none is in the presentation.",
				"UCP 600 art. 14(a)",
				"Obtain the policy or certificate before presenting. A broker's cover note will not do: article 28(c) refuses cover notes outright.")}
		}
		return e.skip("an insurance document in the presentation")
	}
	if ins.Insurance == nil || ins.Insurance.Amount <= 0 {
		return e.skip("the sum insured shown on the insurance document")
	}

	required := e.C.InsurancePercent
	pctSource := "the percentage the credit states"
	if required <= 0 {
		required = e.T.MinInsurancePercent
		pctSource = "UCP 600 art. 28(f)(ii), the credit being silent"
	}

	base, baseText, ok := e.insuranceBase()
	if !ok {
		return e.skip("an invoice value or a drawing amount to measure the insurance against")
	}

	need := RoundTo(base*required/100, e.T.AmountDecimals)
	have := ins.Insurance.Amount

	if AmountBelow(have, need, e.T.AmountDecimals) {
		return []Finding{e.find(SeverityDiscrepancy, "Insured for less than required",
			ins.Label()+" insures "+Money(ins.Insurance.Currency, have, e.T.AmountDecimals)+". The credit needs at least "+
				percentText(required)+" of "+baseText+" ("+e.money(base)+"), which is "+e.money(need)+" — short by "+e.money(need-have)+
				". The percentage comes from "+pctSource+".",
			"UCP 600 art. 28(f)(ii)",
			"Ask the insurer to increase the sum insured to at least "+e.money(need)+
				". A percentage in the credit is a minimum, not a target: insuring for more is never a discrepancy, insuring for a rupee less always is.",
			ins.Label())}
	}
	return []Finding{e.find(SeverityCleared, "Insured for at least the required amount",
		ins.Label()+" insures "+Money(ins.Insurance.Currency, have, e.T.AmountDecimals)+" against a requirement of "+
			percentText(required)+" of "+baseText+" ("+e.money(base)+") = "+e.money(need)+". The percentage comes from "+pctSource+".",
		"UCP 600 art. 28(f)(ii)", "", ins.Label())}
}

func ruleDraft(e *exam) []Finding {
	needs := e.C.Availability == AvailabilityAcceptance
	for _, r := range e.C.RequiredDocuments {
		if r.Kind == KindDraft {
			needs = true
		}
	}
	dr, present := e.P.First(KindDraft)

	switch {
	case !needs && !present:
		return e.skip("a credit calling for a draft, or a draft in the presentation")
	case needs && !present:
		return []Finding{e.find(SeverityDiscrepancy, "No draft presented",
			"The credit is available by acceptance or calls for a draft, and none is in the presentation. A credit available by acceptance is accepted against a draft.",
			"UCP 600 arts. 6(b), 2",
			"Draw the draft on the bank the credit names, for the amount being claimed, on the tenor the credit states.")}
	case !needs && present:
		return []Finding{e.find(SeverityCaution, "A draft is presented that the credit may not call for",
			"A draft is in the presentation but nothing entered shows the credit calling for one. A document presented but not required by the credit is disregarded and may be returned to the presenter.",
			"UCP 600 art. 14(g)",
			"Check field 42 of the credit. If no draft is called for, leave it out rather than risk the data on it conflicting with the rest of the set.",
			dr.Label())}
	}

	var out []Finding
	if dr.Draft != nil && dr.Draft.Amount > 0 && e.hasDrawn && !AmountsEqual(dr.Draft.Amount, e.drawn, e.T.AmountDecimals) {
		out = append(out, e.find(SeverityDiscrepancy, "Draft is not for the amount being claimed",
			"The draft is drawn for "+Money(dr.Draft.Currency, dr.Draft.Amount, e.T.AmountDecimals)+" and the claim is "+e.money(e.drawn)+
				". Data in one document must not conflict with data in another.",
			"UCP 600 art. 14(d)", "Redraw the draft for "+e.money(e.drawn)+".", dr.Label()))
	}
	if dr.Draft != nil && e.C.DraftTenorDays > 0 && dr.Draft.TenorDays > 0 && dr.Draft.TenorDays != e.C.DraftTenorDays {
		out = append(out, e.find(SeverityDiscrepancy, "Draft is on the wrong tenor",
			"The credit states a tenor of "+plural(e.C.DraftTenorDays, "day", "days")+" and the draft is drawn at "+
				plural(dr.Draft.TenorDays, "day", "days")+".",
			"UCP 600 art. 14(d)", "Redraw the draft on the credit's tenor.", dr.Label()))
	}
	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "Draft agrees with the drawing",
			"The draft is drawn for the amount being claimed and on the tenor the credit states.",
			"UCP 600 art. 14(d)", "", dr.Label()))
	}
	return out
}

// ---------- shared readings of the documents ----------

// invoiceLabel names the invoice for a finding, where there is one.
func (e *exam) invoiceLabel() []string {
	if e.hasInvoice {
		return []string{e.invoice.Label()}
	}
	return nil
}

// shippedQuantity reads the quantity shipped off the documents, preferring the
// invoice, then the packing list, then a transport document. The first figure
// found wins rather than a sum, because each document states the whole
// shipment.
func (e *exam) shippedQuantity() (float64, string, bool) {
	for _, k := range []Kind{KindInvoice, KindPackingList, KindTransport, KindWeight} {
		for _, d := range e.P.OfKind(k) {
			if d.Quantity > 0 {
				return d.Quantity, d.QuantityUnit, true
			}
		}
	}
	return 0, "", false
}

// quantityDocs names the documents a quantity finding is about.
func (e *exam) quantityDocs() []string {
	var out []string
	for _, d := range e.P.Documents {
		if d.Quantity > 0 {
			out = append(out, d.Label())
		}
	}
	return out
}

// quantityShortOfCredit reports whether less than the credit's quantity has
// been shipped, and whether that could be determined.
func (e *exam) quantityShortOfCredit() (bool, bool) {
	if e.C.Quantity <= 0 {
		return false, false
	}
	shipped, _, known := e.shippedQuantity()
	if !known {
		return false, false
	}
	return shipped < e.C.Quantity, true
}

// unitPriceReduced reports whether the invoice prices below the credit.
func (e *exam) unitPriceReduced() (bool, bool) {
	if e.C.UnitPrice <= 0 || !e.hasInvoice || e.invoice.Invoice == nil || e.invoice.Invoice.UnitPrice <= 0 {
		return false, false
	}
	return e.invoice.Invoice.UnitPrice < e.C.UnitPrice, true
}

// requiresInsurance reports whether the credit calls for an insurance document.
func (e *exam) requiresInsurance() bool {
	for _, r := range e.C.RequiredDocuments {
		if r.Kind == KindInsurance {
			return true
		}
	}
	return false
}

// insuranceBase works out the value the insurance percentage is measured
// against, following article 28(f)(ii) in the order it is written: the CIF or
// CIP value of the goods where that can be determined from the documents, and
// otherwise the greater of the amount for which honour or negotiation is
// requested and the gross value of the goods shown on the invoice.
func (e *exam) insuranceBase() (float64, string, bool) {
	invoiceTotal := 0.0
	if e.hasInvoice && e.invoice.Invoice != nil {
		invoiceTotal = e.invoice.Invoice.Amount
	}

	if e.C.CIFDeterminable() && invoiceTotal > 0 {
		return invoiceTotal, "the " + e.C.TradeTerm + " value on the invoice", true
	}

	// The CIF or CIP value cannot be worked out from the documents — the term
	// is FOB, CFR or something else that does not include insurance, or there
	// is no invoice total. The sub-article then measures against the greater of
	// the two figures below.
	base, text := 0.0, ""
	if e.hasDrawn {
		base, text = e.drawn, "the amount for which honour is requested"
	}
	if invoiceTotal > base {
		base, text = invoiceTotal, "the gross value of the goods on the invoice"
	}
	if base <= 0 {
		return 0, "", false
	}
	suffix := ""
	if e.C.TradeTerm != "" && !e.C.CIFDeterminable() {
		suffix = " (no CIF or CIP value can be worked out from the documents on " + e.C.TradeTerm + " terms, so article 28(f)(ii) measures against the greater of the drawing and the invoice value)"
	}
	return base, text + suffix, true
}
