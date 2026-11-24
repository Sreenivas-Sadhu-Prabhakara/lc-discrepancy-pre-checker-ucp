package ucp

import "strings"

// partyRules are the checks on what the documents say about the goods and the
// people. Three of them exist mainly to stop an exporter reprinting a document
// they did not need to: UCP 600 is a good deal more forgiving about addresses,
// about the shipper's identity and about the description on a packing list than
// the trade's folklore has it.
func partyRules() []rule {
	return []rule{
		{
			ID:       "P1-INVOICE-GOODS",
			Group:    groupParties,
			Title:    "Goods description on the invoice",
			Citation: "UCP 600 art. 18(c)",
			Tests:    "That the description of the goods on the commercial invoice corresponds with the credit — the strictest description test in the rules, and it applies to the invoice alone.",
			eval:     ruleInvoiceGoods,
		},
		{
			ID:       "P2-OTHER-GOODS",
			Group:    groupParties,
			Title:    "Goods description on the other documents",
			Citation: "UCP 600 arts. 14(e), 14(d)",
			Tests:    "That a description on any document other than the invoice does not conflict with the credit — general terms are expressly allowed there, and a packing list need not repeat the credit word for word.",
			eval:     ruleOtherGoods,
		},
		{
			ID:       "P3-INVOICE-PARTIES",
			Group:    groupParties,
			Title:    "Invoice issued by you, made out to the applicant",
			Citation: "UCP 600 arts. 18(a)(i), 18(a)(ii)",
			Tests:    "That the invoice appears to be issued by the beneficiary and is made out in the name of the applicant, comparing names the way standard practice does rather than character by character.",
			eval:     ruleInvoiceParties,
		},
		{
			ID:       "P4-INVOICE-SIGNATURE",
			Group:    groupParties,
			Title:    "Whether the invoice has to be signed",
			Citation: "UCP 600 art. 18(a)",
			Tests:    "Clears the myth: a commercial invoice need not be signed unless the credit asks for a signed invoice — and flags it where the credit does ask.",
			eval:     ruleInvoiceSignature,
		},
		{
			ID:       "P5-ADDRESSES",
			Group:    groupParties,
			Title:    "Addresses of the applicant and the beneficiary",
			Citation: "UCP 600 art. 14(j)",
			Tests:    "That addresses need only be in the same country as the credit's — except the applicant's address and contact details shown as consignee or notify party on a transport document, which must be exactly as the credit states.",
			eval:     ruleAddresses,
		},
		{
			ID:       "P6-SHIPPER",
			Group:    groupParties,
			Title:    "Who the shipper is",
			Citation: "UCP 600 art. 14(k)",
			Tests:    "Clears another myth: the shipper or consignor shown on any document need not be the beneficiary of the credit.",
			eval:     ruleShipper,
		},
		{
			ID:       "P7-ISSUER-QUALITY",
			Group:    groupParties,
			Title:    "Documents the credit wants from an independent issuer",
			Citation: "UCP 600 art. 3",
			Tests:    "That a document the credit describes as \"independent\", \"first class\", \"official\", \"competent\", \"qualified\", \"well known\" or \"local\" is not issued by the beneficiary — those words allow any issuer except you.",
			eval:     ruleIssuerQuality,
		},
		{
			ID:       "P8-CROSS-DOCUMENT",
			Group:    groupParties,
			Title:    "The same data across the documents",
			Citation: "UCP 600 art. 14(d)",
			Tests:    "That the invoice number, the credit reference, the package count, the gross weight and the shipping marks repeated from one document to another do not conflict.",
			eval:     ruleCrossDocument,
		},
	}
}

func ruleInvoiceGoods(e *exam) []Finding {
	if e.C.GoodsDescription == "" {
		return e.skip("the goods description as the credit words it")
	}
	if !e.hasInvoice {
		return e.skip("a commercial invoice")
	}
	if e.invoice.GoodsDescription == "" {
		return e.skip("the goods description as it appears on your invoice")
	}

	match, why := CompareDescription(e.C.GoodsDescription, e.invoice.GoodsDescription, e.T)
	credit := Shorten(e.C.GoodsDescription, 150)
	doc := Shorten(e.invoice.GoodsDescription, 150)

	switch match {
	case DescriptionCorresponds:
		return []Finding{e.find(SeverityCleared, "Invoice describes the goods as the credit does",
			"The description on "+e.invoice.Label()+" says everything the credit's description says. Where it says more — a purchase order number, a container number — that is not a discrepancy, so long as none of it conflicts.",
			"UCP 600 art. 18(c)", "", e.invoice.Label())}

	case DescriptionTypo:
		return []Finding{e.find(SeverityCaution, "Invoice description differs from the credit by what looks like a typing error",
			"Credit: \""+credit+"\". Invoice: \""+doc+"\". The difference is "+why+
				". Standard banking practice is that a misspelling or typing error which does not affect the meaning of a word does not make a document discrepant — but it is practice, applied by a human examiner, and not a rule you can insist on.",
			"UCP 600 art. 18(c)",
			"Reprint the invoice with the credit's wording. It costs you one sheet of paper and removes the argument entirely.",
			e.invoice.Label())}

	case DescriptionGeneral:
		return []Finding{e.find(SeverityDiscrepancy, "Invoice describes the goods in more general terms than the credit",
			"Credit: \""+credit+"\". Invoice: \""+doc+"\" — "+why+
				". General terms are allowed in documents other than the invoice (article 14(e)); the invoice itself must correspond with the credit.",
			"UCP 600 art. 18(c)",
			"Copy the credit's goods description onto the invoice word for word. You may add to it — a model number, a purchase order reference — but you may not say less than the credit does.",
			e.invoice.Label())}

	default:
		return []Finding{e.find(SeverityDiscrepancy, "Invoice description does not correspond with the credit",
			"Credit: \""+credit+"\". Invoice: \""+doc+"\" — "+why+
				". A difference in a figure or a model code is never read as a typing error: \"model 123\" where the credit says \"model 321\" is a discrepancy, not a slip.",
			"UCP 600 art. 18(c)",
			"Reissue the invoice with the credit's description of the goods. This is the most frequently cited discrepancy in documentary credit practice and the cheapest of all to avoid.",
			e.invoice.Label())}
	}
}

func ruleOtherGoods(e *exam) []Finding {
	if e.C.GoodsDescription == "" {
		return e.skip("the goods description as the credit words it")
	}

	var out []Finding
	checked := 0
	var general []string

	for _, d := range e.P.Documents {
		if d.Kind == KindInvoice || d.GoodsDescription == "" {
			continue // the invoice has its own, stricter test; silence is allowed
		}
		checked++
		match, why := CompareDescription(e.C.GoodsDescription, d.GoodsDescription, e.T)
		switch match {
		case DescriptionCorresponds:
			// nothing to say
		case DescriptionGeneral:
			general = append(general, d.Label())
		case DescriptionTypo:
			out = append(out, e.find(SeverityCaution, "A document's description differs from the credit by what looks like a typing error",
				d.Label()+" describes the goods as \""+Shorten(d.GoodsDescription, 120)+"\" against the credit's \""+
					Shorten(e.C.GoodsDescription, 120)+"\" — "+why+
					". Outside the invoice the test is only that the description must not conflict with the credit, so a slip of this size will usually pass — but it does not have to.",
				"UCP 600 art. 14(e)",
				"Correct it if the document is yours to reprint. If it is a third party's certificate, leave it and be ready to explain.",
				d.Label()))
		default:
			out = append(out, e.find(SeverityDiscrepancy, "A document's description of the goods conflicts with the credit",
				d.Label()+" describes the goods as \""+Shorten(d.GoodsDescription, 120)+"\" against the credit's \""+
					Shorten(e.C.GoodsDescription, 120)+"\" — "+why+
					". In a document other than the invoice the description may be in general terms, but it must not conflict with the credit.",
				"UCP 600 arts. 14(e), 14(d)",
				"Have the document corrected, or cut the description back to general terms that cannot conflict — \"textile machinery parts\" rather than a different model number.",
				d.Label()))
		}
	}

	if checked == 0 {
		return e.skip("a goods description on a document other than the invoice")
	}
	if len(general) > 0 {
		out = append(out, e.find(SeverityCleared, "A general description outside the invoice is allowed",
			strings.Join(general, ", ")+" "+isAre(len(general))+" describing the goods in more general terms than the credit. "+
				"In documents other than the commercial invoice the description may be in general terms, so long as it does not conflict with the credit's.",
			"UCP 600 art. 14(e)",
			"Do not have these reprinted to match the credit word for word. Keeping them general is in fact the safer practice: the less a document says, the less of it can conflict.",
			general...))
	}
	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "No document's description conflicts with the credit",
			plural(checked, "document outside the invoice describes", "documents outside the invoice describe")+" the goods, and none of them conflicts with the credit.",
			"UCP 600 art. 14(e)", ""))
	}
	return out
}

func ruleInvoiceParties(e *exam) []Finding {
	if !e.hasInvoice {
		return e.skip("a commercial invoice")
	}
	if e.C.Beneficiary.Name == "" && e.C.Applicant.Name == "" {
		return e.skip("the applicant's and the beneficiary's names as the credit states them")
	}

	var out []Finding
	inv := e.invoice

	// Article 18(a)(i): the invoice must appear to have been issued by the
	// beneficiary. Article 38 is the exception, for a transferred credit, and
	// this tool does not model transfer.
	if e.C.Beneficiary.Name != "" && inv.Issuer != "" {
		switch Compare(e.C.Beneficiary.Name, inv.Issuer, e.T) {
		case MatchExact, MatchEquivalent:
		case MatchGeneral:
			out = append(out, e.find(SeverityCaution, "Your own name on the invoice is not written at the same length as the credit",
				"The credit names the beneficiary as \""+e.C.Beneficiary.Name+"\" and the invoice is issued by \""+inv.Issuer+
					"\". Nothing in the two conflicts — one simply says more or less than the other — but the party being paid is the name an examiner looks at hardest.",
				"UCP 600 art. 18(a)(i)",
				"Write your name on every document exactly as the credit writes it, suffix and all.",
				inv.Label()))
		case MatchTypo:
			out = append(out, e.find(SeverityCaution, "Your own name on the invoice differs slightly from the credit",
				"The credit names the beneficiary as \""+e.C.Beneficiary.Name+"\" and the invoice is issued by \""+inv.Issuer+
					"\". The difference is of a few characters only. Standard practice treats a misspelling that does not affect meaning as no discrepancy, but an examiner is entitled to look twice at the name of the party being paid.",
				"UCP 600 art. 18(a)(i)",
				"Use the credit's spelling of your own name on every document, letter for letter, including the legal suffix. This is the single cheapest discrepancy to eliminate for good: set it once in your invoice template.",
				inv.Label()))
		default:
			out = append(out, e.find(SeverityDiscrepancy, "Invoice is not issued in the beneficiary's name",
				"The credit names the beneficiary as \""+e.C.Beneficiary.Name+"\" and the invoice is issued by \""+inv.Issuer+
					"\". A commercial invoice must appear to have been issued by the beneficiary.",
				"UCP 600 art. 18(a)(i)",
				"Reissue the invoice in the exact name the credit names as beneficiary. If the credit names the wrong entity — a sister concern, an old trading name — the credit has to be amended; you cannot fix it on the invoice.",
				inv.Label()))
		}
	} else if e.C.Beneficiary.Name != "" && inv.Issuer == "" && !inv.IssuedByBeneficiary {
		out = append(out, e.find(SeverityVerify, "Who issued the invoice was not entered",
			"The invoice must appear to have been issued by the beneficiary, and nothing was entered for its issuer.",
			"UCP 600 art. 18(a)(i)", "Check that the invoice is on your own letterhead in the name the credit uses.", inv.Label()))
	}

	// Article 18(a)(ii): the invoice must be made out in the name of the
	// applicant. Sub-article 38(g) is the exception, again for transfer.
	madeOut := ""
	if inv.Invoice != nil {
		madeOut = inv.Invoice.MadeOutTo.Name
	}
	if e.C.Applicant.Name != "" && madeOut != "" {
		switch Compare(e.C.Applicant.Name, madeOut, e.T) {
		case MatchExact, MatchEquivalent:
		case MatchGeneral:
			out = append(out, e.find(SeverityCaution, "The buyer's name on the invoice is not written at the same length as the credit",
				"The credit names the applicant as \""+e.C.Applicant.Name+"\" and the invoice is made out to \""+madeOut+
					"\". Nothing conflicts, but article 18(a)(ii) wants the invoice made out in the name of the applicant.",
				"UCP 600 art. 18(a)(ii)",
				"Copy the applicant's name from the credit, suffix and all.",
				inv.Label()))
		case MatchTypo:
			out = append(out, e.find(SeverityCaution, "The buyer's name on the invoice differs slightly from the credit",
				"The credit names the applicant as \""+e.C.Applicant.Name+"\" and the invoice is made out to \""+madeOut+
					"\". The difference looks like a typing error rather than a different party.",
				"UCP 600 art. 18(a)(ii)",
				"Copy the applicant's name from the credit, not from the purchase order. The two often differ, and the credit is what the bank reads.",
				inv.Label()))
		default:
			out = append(out, e.find(SeverityDiscrepancy, "Invoice is not made out to the applicant",
				"The credit names the applicant as \""+e.C.Applicant.Name+"\" and the invoice is made out to \""+madeOut+
					"\". A commercial invoice must be made out in the name of the applicant.",
				"UCP 600 art. 18(a)(ii)",
				"Reissue the invoice to the applicant exactly as the credit names it. Billing a buying agent, a group company or a consignee instead is a discrepancy even where it is what the trade actually does.",
				inv.Label()))
		}
	} else if e.C.Applicant.Name != "" && madeOut == "" {
		out = append(out, e.find(SeverityVerify, "Who the invoice is made out to was not entered",
			"The invoice must be made out in the name of the applicant, and nothing was entered for the party it is made out to.",
			"UCP 600 art. 18(a)(ii)", "Check the invoice names the applicant exactly as the credit does.", inv.Label()))
	}

	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "Invoice issued by you and made out to the applicant",
			"The names on the invoice agree with the credit.",
			"UCP 600 art. 18(a)", "", inv.Label()))
	}
	return out
}

func ruleInvoiceSignature(e *exam) []Finding {
	if !e.hasInvoice {
		return e.skip("a commercial invoice")
	}

	required := false
	for _, r := range e.C.RequiredDocuments {
		if r.Kind == KindInvoice && r.MustBeSigned {
			required = true
		}
	}

	if required && !e.invoice.Signed {
		return []Finding{e.find(SeverityDiscrepancy, "The credit calls for a signed invoice and this one is not signed",
			"An invoice need not be signed under article 18(a) — but where the credit asks for a signed commercial invoice, the credit's own term governs and the invoice has to be signed.",
			"UCP 600 arts. 18(a), 14(a)",
			"Sign and stamp the invoice. Where the credit says \"signed and stamped\", do both.",
			e.invoice.Label())}
	}
	if required {
		return []Finding{e.find(SeverityCleared, "Signed invoice, as the credit asks",
			"The credit calls for a signed commercial invoice and the invoice is signed.",
			"UCP 600 art. 14(a)", "", e.invoice.Label())}
	}
	if !e.invoice.Signed {
		return []Finding{e.find(SeverityCleared, "An unsigned invoice is not a discrepancy here",
			"Nothing in the credit as entered calls for the invoice to be signed, and article 18(a) says in terms that a commercial invoice need not be signed.",
			"UCP 600 art. 18(a)",
			"Read field 46A of the credit again to be sure it does not say \"signed commercial invoice\". If it does, sign it — the credit's term beats the rule's default.",
			e.invoice.Label())}
	}
	return []Finding{e.find(SeverityCleared, "Invoice is signed, which can do no harm",
		"The credit as entered does not require a signed invoice, and article 18(a) does not either. Signing it anyway is not a discrepancy.",
		"UCP 600 art. 18(a)", "", e.invoice.Label())}
}

func ruleAddresses(e *exam) []Finding {
	var out []Finding
	checked := 0

	// The strict half of article 14(j) first: where the applicant's address and
	// contact details appear as part of the consignee or notify party details
	// on a transport document subject to articles 19 to 25, they must be as
	// stated in the credit. This is the one address test that is exact, and it
	// is the one exporters are least likely to know about.
	if e.C.Applicant.Name != "" || e.C.Applicant.Address != "" {
		for _, d := range e.P.TransportDocuments() {
			if d.Transport == nil {
				continue
			}
			if _, governed := d.Transport.Mode.Article(); !governed {
				continue
			}
			for _, field := range []struct{ what, text string }{
				{"consignee", d.Transport.Consignee},
				{"notify party", d.Transport.NotifyParty},
			} {
				if field.text == "" || !mentionsApplicant(field.text, e.C.Applicant, e.T) {
					continue
				}
				checked++
				if e.C.Applicant.Address == "" {
					out = append(out, e.find(SeverityVerify, "Applicant's address shown as "+field.what+" — compare it with the credit",
						"The applicant appears as "+field.what+" on "+d.Label()+". Where the applicant's address and contact details appear as part of the consignee or notify party details on a transport document, they must be as stated in the credit — this is the exception to the rule that an address need only be in the same country.",
						"UCP 600 art. 14(j)",
						"Put the applicant's address on the transport document exactly as field 50 of the credit gives it, including the telephone and fax numbers if the credit carries them.",
						d.Label()))
					continue
				}
				if !addressAsStated(e.C.Applicant.Address, field.text) {
					out = append(out, e.find(SeverityDiscrepancy, "Applicant's address as "+field.what+" is not as the credit states it",
						"On "+d.Label()+" the "+field.what+" reads \""+Shorten(field.text, 120)+"\". The credit gives the applicant's address as \""+
							Shorten(e.C.Applicant.Address, 120)+"\". Where the applicant's address and contact details appear as part of the consignee or notify party details on a transport document, they must be as stated in the credit.",
						"UCP 600 art. 14(j)",
						"Have the carrier correct the consignee and notify party boxes to the applicant's address exactly as field 50 of the credit gives it, before the document is released. Afterwards it means an amended bill of lading.",
						d.Label()))
				}
			}
		}
	}

	// The forgiving half: everywhere else an address need only be in the same
	// country, and contact details forming part of an address are disregarded.
	for _, pair := range []struct {
		who    string
		credit Party
		doc    Party
		label  string
	}{
		{"applicant", e.C.Applicant, e.invoiceMadeOutTo(), e.invoiceLabelOr("the invoice")},
	} {
		if pair.credit.Address == "" || pair.doc.Address == "" {
			continue
		}
		checked++
		if Compare(pair.credit.Address, pair.doc.Address, e.T) != MatchConflict {
			continue
		}
		same, certain := SameCountry(pair.credit.Country, pair.doc.Country)
		switch {
		case same:
			out = append(out, e.find(SeverityCleared, "A different address for the "+pair.who+" is not a discrepancy",
				"The address on "+pair.label+" differs from the credit's, but both are in the same country ("+pair.credit.Country+
					"). Article 14(j) requires only that the addresses of the beneficiary and the applicant be within the same country as those in the credit — not that they be the same address. Contact details given as part of an address are disregarded.",
				"UCP 600 art. 14(j)",
				"Leave it. Reprinting a document because a branch address differs from a head office address is wasted work.",
				pair.label))
		case certain:
			out = append(out, e.find(SeverityDiscrepancy, "The "+pair.who+"'s address is in another country",
				"The credit gives the "+pair.who+"'s country as "+pair.credit.Country+" and "+pair.label+" gives "+pair.doc.Country+
					". An address may differ from the credit's, but it must be within the same country.",
				"UCP 600 art. 14(j)",
				"Use an address in the country the credit states, or have the credit amended.",
				pair.label))
		default:
			out = append(out, e.find(SeverityCaution, "The "+pair.who+"'s address differs — check it is the same country",
				"The address on "+pair.label+" differs from the credit's. That is allowed so long as it is in the same country ("+
					orDash(pair.credit.Country)+" against "+orDash(pair.doc.Country)+"), which this tool could not confirm from the names entered.",
				"UCP 600 art. 14(j)",
				"Satisfy yourself the two addresses are in the same country. If they are, there is nothing to fix.",
				pair.label))
		}
	}

	if checked == 0 {
		return e.skip("an address on the credit and on a document to compare it with")
	}
	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "Addresses raise nothing",
			"The addresses entered either match the credit or are in the same country, which is all article 14(j) asks outside the consignee and notify party boxes of a transport document.",
			"UCP 600 art. 14(j)", ""))
	}
	return out
}

func ruleShipper(e *exam) []Finding {
	if e.C.Beneficiary.Name == "" {
		return e.skip("the beneficiary's name as the credit states it")
	}
	var differing []string
	for _, d := range e.P.TransportDocuments() {
		if d.Transport == nil || d.Transport.Shipper == "" {
			continue
		}
		if Compare(e.C.Beneficiary.Name, d.Transport.Shipper, e.T) == MatchConflict {
			differing = append(differing, d.Label()+" (shipper: "+Shorten(d.Transport.Shipper, 60)+")")
		}
	}
	if len(differing) == 0 {
		return e.skip("a transport document naming a shipper other than you (nothing to clear)")
	}
	return []Finding{e.find(SeverityCleared, "The shipper need not be you",
		strings.Join(differing, ", ")+" "+isAre(len(differing))+" shipped by someone other than the beneficiary named in the credit. Article 14(k) says in terms that the shipper or consignor of the goods indicated on any document need not be the beneficiary of the credit.",
		"UCP 600 art. 14(k)",
		"Leave it. A merchant exporter shipping in a manufacturer's name, or a freight forwarder shown as shipper, is not a discrepancy — unless the credit itself demands that the shipper be the beneficiary, in which case the credit's own term governs.",
		differing...)}
}

func ruleIssuerQuality(e *exam) []Finding {
	var demanding []Requirement
	for _, r := range e.C.RequiredDocuments {
		if r.IssuerMustBeIndependent() {
			demanding = append(demanding, r)
		}
	}
	if len(demanding) == 0 {
		return e.skip("a document the credit describes as from an \"independent\", \"first class\", \"official\", \"competent\", \"qualified\", \"well known\" or \"local\" issuer")
	}

	var out []Finding
	for _, r := range demanding {
		doc, present := e.P.First(r.Kind)
		if !present {
			continue // the missing-document rule reports this
		}
		byBeneficiary := doc.IssuedByBeneficiary
		if !byBeneficiary && doc.Issuer != "" && e.C.Beneficiary.Name != "" {
			switch Compare(e.C.Beneficiary.Name, doc.Issuer, e.T) {
			case MatchExact, MatchEquivalent, MatchTypo:
				byBeneficiary = true
			}
		}
		if byBeneficiary {
			out = append(out, e.find(SeverityDiscrepancy, "A document the credit wants from an independent issuer is your own",
				"The credit calls for \""+Shorten(r.Wording, 110)+"\" and "+doc.Label()+" appears to be issued by the beneficiary. Terms such as \"first class\", \"well known\", \"qualified\", \"independent\", \"official\", \"competent\" or \"local\", used to describe the issuer of a document, allow any issuer except the beneficiary to issue that document.",
				"UCP 600 art. 3",
				"Have it issued by a third party — a chamber of commerce, a surveyor, a laboratory. Your own certificate on your own letterhead cannot satisfy a credit worded this way, however true its contents.",
				doc.Label()))
			continue
		}
		out = append(out, e.find(SeverityCleared, "An independently issued document, as the credit asks",
			"The credit calls for \""+Shorten(r.Wording, 110)+"\" and "+doc.Label()+" is issued by "+
				orDash(doc.Issuer)+", not by the beneficiary. Any issuer other than the beneficiary satisfies wording of this kind — the issuer does not have to prove it is \"first class\".",
			"UCP 600 art. 3", "", doc.Label()))
	}
	if len(out) == 0 {
		return e.skip("the document the credit wants independently issued (it is not in the presentation)")
	}
	return out
}

func ruleCrossDocument(e *exam) []Finding {
	var out []Finding
	comparisons := 0

	// An invoice number repeated on another document must agree with the
	// invoice's own number, which is read off the invoice's title or its
	// InvoiceNumberShown field.
	invoiceNo := ""
	if e.hasInvoice {
		invoiceNo = e.invoice.InvoiceNumberShown
	}
	if invoiceNo != "" {
		for _, d := range e.P.Documents {
			if d.Kind == KindInvoice || d.InvoiceNumberShown == "" {
				continue
			}
			comparisons++
			if Normalised(d.InvoiceNumberShown) != Normalised(invoiceNo) {
				out = append(out, e.find(SeverityDiscrepancy, "Invoice number differs between documents",
					d.Label()+" quotes invoice number \""+d.InvoiceNumberShown+"\" and the invoice is numbered \""+invoiceNo+
						"\". Data in a document must not conflict with data in any other stipulated document.",
					"UCP 600 art. 14(d)",
					"Correct the quoting document. An invoice number is a figure, and a figure that differs is a conflict, never a typing error.",
					d.Label(), e.invoice.Label()))
			}
		}
	}

	// The credit's own reference, where documents quote it.
	if e.C.Reference != "" {
		for _, d := range e.P.Documents {
			if d.LCReferenceShown == "" {
				continue
			}
			comparisons++
			if Normalised(d.LCReferenceShown) != Normalised(e.C.Reference) {
				out = append(out, e.find(SeverityDiscrepancy, "Credit number quoted wrongly",
					d.Label()+" quotes the credit as \""+d.LCReferenceShown+"\" and the credit is \""+e.C.Reference+"\".",
					"UCP 600 art. 14(d)",
					"Correct the document. A wrong credit number also delays the bank matching your presentation to the credit at all.",
					d.Label()))
			}
		}
	}

	// Package counts, gross weights and shipping marks repeated across
	// documents.
	out = append(out, e.compareNumberAcrossDocuments("package count", func(d Document) float64 { return d.Packages }, &comparisons)...)
	out = append(out, e.compareNumberAcrossDocuments("gross weight in kg", func(d Document) float64 { return d.GrossWeightKg }, &comparisons)...)

	var marks []struct {
		label string
		text  string
	}
	for _, d := range e.P.Documents {
		if d.ShippingMarks != "" {
			marks = append(marks, struct {
				label string
				text  string
			}{d.Label(), d.ShippingMarks})
		}
	}
	for i := 1; i < len(marks); i++ {
		comparisons++
		if Compare(marks[0].text, marks[i].text, e.T) == MatchConflict {
			out = append(out, e.find(SeverityDiscrepancy, "Shipping marks differ between documents",
				marks[0].label+" shows marks \""+Shorten(marks[0].text, 80)+"\" and "+marks[i].label+" shows \""+
					Shorten(marks[i].text, 80)+"\".",
				"UCP 600 art. 14(d)",
				"Make the marks identical across the set, or leave them off the documents that do not have to carry them — a box not filled in cannot conflict with one that is.",
				marks[0].label, marks[i].label))
		}
	}

	if comparisons == 0 {
		return e.skip("the same figure on two documents to compare — an invoice number, a package count, a weight or shipping marks")
	}
	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "The documents agree with one another",
			plural(comparisons, "figure was", "figures were")+" compared across the set and none of them conflicts.",
			"UCP 600 art. 14(d)", ""))
	}
	return out
}

// compareNumberAcrossDocuments compares one numeric field wherever two or more
// documents state it.
func (e *exam) compareNumberAcrossDocuments(what string, read func(Document) float64, comparisons *int) []Finding {
	type stated struct {
		label string
		value float64
	}
	var all []stated
	for _, d := range e.P.Documents {
		if v := read(d); v > 0 {
			all = append(all, stated{d.Label(), v})
		}
	}
	var out []Finding
	for i := 1; i < len(all); i++ {
		*comparisons++
		if !AmountsEqual(all[0].value, all[i].value, 3) {
			out = append(out, e.find(SeverityDiscrepancy, "The "+what+" differs between documents",
				all[0].label+" shows "+Quantity(all[0].value, "")+" and "+all[i].label+" shows "+Quantity(all[i].value, "")+
					". Data in a document must not conflict with data in any other stipulated document.",
				"UCP 600 art. 14(d)",
				"Reconcile the two before presenting. A packing list that counts 500 cartons against a bill of lading that counts 499 is refused on sight.",
				all[0].label, all[i].label))
		}
	}
	return out
}

// addressAsStated reports whether a consignee or notify party box repeats the
// credit's address in full. Article 14(j) requires the applicant's address and
// contact details, where they appear in those boxes on a transport document, to
// be "as stated in the credit" — so every part of the credit's address has to
// be there. A box that happens to share the word "city" with it does not
// satisfy that, which is why this is a subset test and not a similarity one.
func addressAsStated(creditAddress, box string) bool {
	want := Tokens(creditAddress)
	if len(want) == 0 {
		return true
	}
	return subsetStrings(want, Tokens(box))
}

// mentionsApplicant reports whether a consignee or notify party box appears to
// name the applicant, so that the strict half of article 14(j) is applied only
// where it bites.
func mentionsApplicant(text string, applicant Party, t Thresholds) bool {
	if applicant.Name == "" {
		return false
	}
	switch Compare(applicant.Name, text, t) {
	case MatchExact, MatchEquivalent, MatchTypo, MatchGeneral:
		return true
	}
	return ContainsAny(text, applicant.Name)
}

// invoiceMadeOutTo returns the party the invoice is made out to.
func (e *exam) invoiceMadeOutTo() Party {
	if e.hasInvoice && e.invoice.Invoice != nil {
		return e.invoice.Invoice.MadeOutTo
	}
	return Party{}
}

// invoiceLabelOr names the invoice, or the given fallback where there is none.
func (e *exam) invoiceLabelOr(fallback string) string {
	if e.hasInvoice {
		return e.invoice.Label()
	}
	return fallback
}
