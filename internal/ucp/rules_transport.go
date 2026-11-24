package ucp

import "strings"

// transportRules are the checks on the transport document. Two of them —
// transhipment and partial shipment — exist because the rules are much narrower
// than the trade believes, and a prohibition in the credit very often has no
// effect at all on the document in front of you.
func transportRules() []rule {
	return []rule{
		{
			ID:       "T1-CLEAN",
			Group:    groupTransport,
			Title:    "A clean transport document",
			Citation: "UCP 600 art. 27",
			Tests:    "That the document bears no clause expressly declaring a defective condition of the goods or their packaging — and clears the belief that the word \"clean\" has to appear on it.",
			eval:     ruleClean,
		},
		{
			ID:       "T2-ON-DECK",
			Group:    groupTransport,
			Title:    "Goods not loaded on deck",
			Citation: "UCP 600 art. 26(a)",
			Tests:    "That the document does not indicate the goods are or will be loaded on deck, while clearing a clause saying they may be, which is acceptable.",
			eval:     ruleOnDeck,
		},
		{
			ID:       "T3-CHARTER-PARTY",
			Group:    groupTransport,
			Title:    "No charter party bill of lading unless asked for",
			Citation: "UCP 600 arts. 20(a)(vi), 22",
			Tests:    "That a bill of lading contains no indication that it is subject to a charter party, unless the credit calls for a charter party bill of lading.",
			eval:     ruleCharterParty,
		},
		{
			ID:       "T4-PORTS",
			Group:    groupTransport,
			Title:    "Ports and places as the credit states them",
			Citation: "UCP 600 arts. 20(a)(ii), 20(a)(iii), 19(a)(ii), 19(a)(iii)",
			Tests:    "That the port of loading and the port of discharge — or for a multimodal document the place of taking in charge and the place of final destination — are the credit's.",
			eval:     rulePorts,
		},
		{
			ID:       "T5-FULL-SET",
			Group:    groupTransport,
			Title:    "The full set of originals",
			Citation: "UCP 600 arts. 20(a)(iv), 23(a)(v)",
			Tests:    "That as many originals are presented as the document says were issued — and clears an air waybill, where only the original for the consignor or shipper is wanted even if the credit says \"full set\".",
			eval:     ruleFullSet,
		},
		{
			ID:       "T6-TRANSHIPMENT",
			Group:    groupTransport,
			Title:    "Transhipment where the credit forbids it",
			Citation: "UCP 600 arts. 19(c), 20(c), 20(d), 21(c), 23(c), 24(e)",
			Tests:    "Whether a prohibition on transhipment actually bites: for a container shipment, an air waybill, a road or rail document or a multimodal document, the rules accept transhipment even where the credit forbids it.",
			eval:     ruleTranshipment,
		},
		{
			ID:       "T7-PARTIAL",
			Group:    groupTransport,
			Title:    "Partial shipment where the credit forbids it",
			Citation: "UCP 600 arts. 31(a), 31(b), 31(c)",
			Tests:    "Whether more than one set of transport documents is a partial shipment at all — it is not, where the shipment is on the same means of conveyance for the same journey to the same destination.",
			eval:     rulePartialShipment,
		},
		{
			ID:       "T8-ON-BOARD",
			Group:    groupTransport,
			Title:    "Evidence that the goods are on board",
			Citation: "UCP 600 arts. 20(a)(ii), 20(b)",
			Tests:    "That a credit calling for an on board bill of lading gets one: either pre-printed shipped-on-board wording or an on board notation with a date.",
			eval:     ruleOnBoard,
		},
	}
}

// transportDocs returns the transport documents, or records the gap.
func (e *exam) transportDocs() ([]Document, bool) {
	docs := e.P.TransportDocuments()
	if len(docs) == 0 {
		e.skip("a transport document in the presentation")
		return nil, false
	}
	return docs, true
}

func ruleClean(e *exam) []Finding {
	docs, ok := e.transportDocs()
	if !ok {
		return nil
	}
	var out []Finding
	for _, d := range docs {
		if d.Transport == nil {
			continue
		}
		if clause := strings.TrimSpace(d.Transport.AdverseClause); clause != "" {
			out = append(out, e.find(SeverityDiscrepancy, "The transport document is not clean",
				d.Label()+" bears the clause \""+Shorten(clause, 120)+"\". A bank will only accept a clean transport document, and a clean transport document is one bearing no clause or notation expressly declaring a defective condition of the goods or their packaging.",
				"UCP 600 art. 27",
				"Have the carrier remove the clause, which means satisfying them about the packing, or reship. A clause of this kind cannot be waived by you and is not cured by a letter of indemnity to the carrier — the bank is reading the bill of lading, not your indemnity.",
				d.Label()))
			continue
		}
		out = append(out, e.find(SeverityCleared, "The transport document is clean",
			d.Label()+" carries no clause declaring a defective condition of the goods or their packaging. The word \"clean\" need not appear on it, even where the credit asks for a document \"clean on board\".",
			"UCP 600 art. 27",
			"Do not ask the carrier to stamp the word \"clean\" on the bill of lading to satisfy a credit that says \"clean on board\". It is unnecessary, and chasing it delays the set.",
			d.Label()))
	}
	return out
}

func ruleOnDeck(e *exam) []Finding {
	docs, ok := e.transportDocs()
	if !ok {
		return nil
	}
	var out []Finding
	for _, d := range docs {
		if d.Transport == nil {
			continue
		}
		switch {
		case d.Transport.OnDeck:
			out = append(out, e.find(SeverityDiscrepancy, "The goods are shown as loaded on deck",
				d.Label()+" indicates that the goods are or will be loaded on deck. A transport document must not indicate that.",
				"UCP 600 art. 26(a)",
				"Have the goods stowed under deck and the document reissued. An on-deck shipment also usually falls outside ordinary marine cover, so the insurance document is likely to be a second problem.",
				d.Label()))
		case d.Transport.MayBeOnDeck:
			out = append(out, e.find(SeverityCleared, "A clause saying the goods may be carried on deck is acceptable",
				d.Label()+" carries a clause reserving the carrier's liberty to stow on deck. A clause stating that the goods may be loaded on deck is acceptable; only a document indicating that they are or will be is not.",
				"UCP 600 art. 26(a)",
				"Leave it. The liberty clause is printed on most bills of lading and is not a discrepancy.",
				d.Label()))
		}
	}
	if len(out) == 0 {
		return e.skip("anything entered about deck stowage on the transport document")
	}
	return out
}

func ruleCharterParty(e *exam) []Finding {
	docs, ok := e.transportDocs()
	if !ok {
		return nil
	}
	wanted := false
	for _, r := range e.C.RequiredDocuments {
		if r.Kind == KindTransport && ContainsAny(r.Wording, "charter party", "charterparty") {
			wanted = true
		}
	}

	var out []Finding
	for _, d := range docs {
		if d.Transport == nil {
			continue
		}
		isCP := d.Transport.Mode == ModeCharterParty || d.Transport.SubjectToCharterParty
		switch {
		case isCP && !wanted:
			out = append(out, e.find(SeverityDiscrepancy, "The bill of lading is subject to a charter party",
				d.Label()+" is, or indicates that it is subject to, a charter party. A bill of lading under article 20 must contain no indication that it is subject to a charter party; a charter party bill of lading is acceptable only where the credit calls for one, under article 22.",
				"UCP 600 arts. 20(a)(vi), 22",
				"Obtain a liner bill of lading, or have the credit amended to call for a charter party bill of lading. On bulk cargoes a charter party bill is often the only document the trade produces — settle this with the applicant before the vessel is fixed, not after.",
				d.Label()))
		case isCP && wanted:
			out = append(out, e.find(SeverityCleared, "A charter party bill of lading, as the credit allows",
				"The credit calls for a charter party bill of lading and "+d.Label()+" is one. Article 22 governs it, and a bank will not examine the charter party contract itself even where the document says it is presented.",
				"UCP 600 art. 22", "", d.Label()))
		case !isCP && wanted:
			out = append(out, e.find(SeverityCaution, "The credit allows a charter party bill of lading and this is not one",
				"The credit calls for a charter party bill of lading; "+d.Label()+" is "+d.Transport.Mode.Title()+
					". That is usually in your favour, but satisfy yourself the credit does not require a charter party bill specifically.",
				"UCP 600 art. 22", "", d.Label()))
		}
	}
	if len(out) == 0 {
		return e.skip("anything indicating a charter party (nothing to report)")
	}
	return out
}

func rulePorts(e *exam) []Finding {
	docs, ok := e.transportDocs()
	if !ok {
		return nil
	}

	var out []Finding
	comparisons := 0

	for _, d := range docs {
		if d.Transport == nil {
			continue
		}
		multimodal := d.Transport.Mode == ModeMultimodal

		type pair struct {
			what     string
			credit   string
			document string
			citation string
		}
		var pairs []pair
		if multimodal {
			pairs = []pair{
				{"place of taking in charge", e.C.PlaceOfReceipt, d.Transport.PlaceOfReceipt, "UCP 600 art. 19(a)(ii)"},
				{"place of final destination", e.C.FinalDestination, d.Transport.FinalDestination, "UCP 600 art. 19(a)(iii)"},
			}
		} else {
			pairs = []pair{
				{"port of loading", e.C.PortOfLoading, d.Transport.PortOfLoading, "UCP 600 art. 20(a)(ii)"},
				{"port of discharge", e.C.PortOfDischarge, d.Transport.PortOfDischarge, "UCP 600 art. 20(a)(iii)"},
			}
		}

		for _, p := range pairs {
			if p.credit == "" || p.document == "" {
				continue
			}
			comparisons++
			switch Compare(p.credit, p.document, e.T) {
			case MatchExact, MatchEquivalent, MatchGeneral:
			case MatchTypo:
				out = append(out, e.find(SeverityCaution, "The "+p.what+" is spelt differently from the credit",
					d.Label()+" shows the "+p.what+" as \""+p.document+"\" and the credit says \""+p.credit+
						"\". The difference is of a character or two.",
					p.citation,
					"Use the credit's spelling. Port names have several accepted forms and a bank reading two of them side by side has to decide which it is looking at.",
					d.Label()))
			default:
				out = append(out, e.find(SeverityDiscrepancy, "The "+p.what+" is not the one the credit states",
					d.Label()+" shows the "+p.what+" as \""+p.document+"\" and the credit requires \""+p.credit+"\".",
					p.citation,
					"This cannot be corrected on paper after the fact. Where the credit names a port the vessel does not call at, obtain an amendment before shipping.",
					d.Label()))
			}
		}
	}

	if comparisons == 0 {
		return e.skip("ports or places on both the credit and the transport document")
	}
	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "Ports and places as the credit states them",
			plural(comparisons, "place was", "places were")+" compared and all of them agree with the credit.",
			"UCP 600 art. 20(a)", ""))
	}
	return out
}

func ruleFullSet(e *exam) []Finding {
	docs, ok := e.transportDocs()
	if !ok {
		return nil
	}
	var out []Finding
	tested := 0

	for _, d := range docs {
		if d.Transport == nil {
			continue
		}

		if d.Transport.Mode == ModeAir {
			tested++
			out = append(out, e.find(SeverityCleared, "An air waybill needs only the original for the consignor",
				d.Label()+" is an air transport document. It must be the original for the consignor or shipper, even if the credit stipulates a full set of originals — the other two originals of an air waybill belong to the carrier and the consignee and never travel with the documents.",
				"UCP 600 art. 23(a)(v)",
				"Do not chase the carrier for \"all three originals\" of an air waybill because the credit says full set. Present the original for the shipper.",
				d.Label()))
			continue
		}

		if d.Transport.OriginalsIssued <= 0 || d.Originals <= 0 {
			continue
		}
		tested++
		switch {
		case d.Originals < d.Transport.OriginalsIssued:
			out = append(out, e.find(SeverityDiscrepancy, "Not the full set of originals",
				d.Label()+" says it was issued in "+plural(d.Transport.OriginalsIssued, "original", "originals")+
					" and only "+plural(d.Originals, "is", "are")+" in the set. A bill of lading must be the sole original or, if issued in more than one original, be the full set as indicated on the bill of lading.",
				"UCP 600 art. 20(a)(iv)",
				"Recover the missing original from whoever holds it before you present. A bill of lading is a document of title: a bank that releases payment against two of three originals has let a third party collect your goods.",
				d.Label()))
		default:
			out = append(out, e.find(SeverityCleared, "The full set of originals is in the set",
				d.Label()+" says it was issued in "+plural(d.Transport.OriginalsIssued, "original", "originals")+
					" and "+plural(d.Originals, "is", "are")+" being presented.",
				"UCP 600 art. 20(a)(iv)", "", d.Label()))
		}
	}

	if tested == 0 {
		return e.skip("how many originals the transport document says were issued, and how many you hold")
	}
	return out
}

func ruleTranshipment(e *exam) []Finding {
	docs, ok := e.transportDocs()
	if !ok {
		return nil
	}
	if e.C.Transhipment != PermissionProhibited {
		return e.skip("a credit prohibiting transhipment (this one does not, so art. 31(a)'s default applies and transhipment is allowed)")
	}

	var out []Finding
	reported := 0

	for _, d := range docs {
		if d.Transport == nil || d.Transport.Transhipment == TranshipmentNone {
			continue
		}
		reported++
		mode := d.Transport.Mode
		shown := "that transhipment may take place"
		if d.Transport.Transhipment == TranshipmentWill {
			shown = "that transhipment will take place"
		}

		switch mode {
		case ModeAir:
			out = append(out, e.find(SeverityCleared, "A prohibition on transhipment does not bite on an air waybill",
				d.Label()+" indicates "+shown+", and the credit prohibits transhipment — but an air transport document indicating that transhipment will or may take place is acceptable even if the credit prohibits transhipment, provided the entire carriage is covered by one and the same air transport document.",
				"UCP 600 art. 23(c)(ii)",
				"Check only that one air waybill covers the whole carriage. Then leave it: asking the airline to remove the transhipment entry is asking them to misdescribe the routing.",
				d.Label()))
		case ModeRoadRail:
			out = append(out, e.find(SeverityCleared, "A prohibition on transhipment does not bite on a road, rail or inland waterway document",
				d.Label()+" indicates "+shown+". A road, rail or inland waterway transport document indicating that transhipment will or may take place is acceptable even if the credit prohibits transhipment, provided the entire carriage is covered by one and the same transport document.",
				"UCP 600 art. 24(e)(ii)", "", d.Label()))
		case ModeMultimodal:
			out = append(out, e.find(SeverityCleared, "A prohibition on transhipment does not bite on a multimodal document",
				d.Label()+" indicates "+shown+". A transport document covering at least two different modes of transport may indicate that the goods will or may be transhipped, provided the entire carriage is covered by one and the same transport document — and it is acceptable even if the credit prohibits transhipment.",
				"UCP 600 art. 19(c)",
				"A credit that prohibits transhipment while calling for a multimodal document is asking for something the rules will not give it. Nothing to fix on your side.",
				d.Label()))
		case ModeBillOfLading, ModeSeaWaybill, ModeCharterParty:
			citation := "UCP 600 art. 20(c)(ii)"
			if mode == ModeSeaWaybill {
				citation = "UCP 600 art. 21(c)(ii)"
			}
			switch {
			case d.Transport.Containerised:
				out = append(out, e.find(SeverityCleared, "Transhipment is acceptable because the goods are in a container",
					d.Label()+" indicates "+shown+" and the credit prohibits transhipment — but a bill of lading indicating that transhipment will or may take place is acceptable, even if the credit prohibits transhipment, if the goods have been shipped in a container, trailer or LASH barge as evidenced by the bill of lading.",
					citation,
					"Make sure the container is evidenced on the face of the bill of lading itself — the container number in the marks box is what the sub-article depends on.",
					d.Label()))
			case d.Transport.Transhipment == TranshipmentMay:
				out = append(out, e.find(SeverityCaution, "The bill of lading reserves a liberty to tranship",
					d.Label()+" indicates "+shown+", the goods are not shown as containerised, and the credit prohibits transhipment. A clause in a bill of lading stating that the carrier reserves the right to tranship will be disregarded — so a printed liberty clause is harmless. A positive indication that transhipment may take place on this shipment is not.",
					"UCP 600 arts. 20(d), 20(c)(ii)",
					"Read the document. If it is the carrier's standard printed liberty clause, article 20(d) disregards it and there is nothing to do. If the routing boxes actually show a transhipment port, you need either a container shipment, a direct vessel, or an amendment.",
					d.Label()))
			default:
				out = append(out, e.find(SeverityDiscrepancy, "Transhipment where the credit forbids it",
					d.Label()+" shows "+shown+", the goods are not shown as shipped in a container, trailer or LASH barge, and the credit prohibits transhipment.",
					citation,
					"Ship on a direct vessel, ship in a container so that article 20(c)(ii) applies, or obtain an amendment allowing transhipment. The container exception is the usual answer and costs nothing on a full-container shipment.",
					d.Label()))
			}
		default:
			out = append(out, e.find(SeverityCaution, "Transhipment on a document outside articles 19 to 25",
				d.Label()+" indicates "+shown+" and the credit prohibits transhipment. None of the transhipment sub-articles reaches a document outside articles 19 to 25, so the credit's own prohibition is read as it stands.",
				"UCP 600 art. 14(a)",
				"Ask your bank how they will read it before you present.", d.Label()))
		}
	}

	if reported == 0 {
		return []Finding{e.find(SeverityCleared, "No transhipment shown, and the credit forbids it",
			"The credit prohibits transhipment and no transport document in the set indicates that transhipment will or may take place.",
			"UCP 600 art. 20(c)",
			"Worth knowing for next time: a prohibition on transhipment is far weaker than it looks. For a container shipment, an air waybill, a road or rail document or a multimodal document, the rules accept transhipment anyway.")}
	}
	return out
}

func rulePartialShipment(e *exam) []Finding {
	if e.C.PartialShipments != PermissionProhibited {
		return e.skip("a credit prohibiting partial shipment (this one does not — art. 31(a) allows partial drawings and shipments unless the credit says otherwise)")
	}
	docs, ok := e.transportDocs()
	if !ok {
		return nil
	}

	var out []Finding

	if len(docs) > 1 {
		sameConveyance := true
		sameDestination := true
		allCourier := true
		for _, d := range docs[1:] {
			if d.Transport == nil || docs[0].Transport == nil {
				sameConveyance, sameDestination = false, false
				break
			}
			if Compare(docs[0].Transport.Vessel, d.Transport.Vessel, e.T) == MatchConflict ||
				Compare(docs[0].Transport.Voyage, d.Transport.Voyage, e.T) == MatchConflict {
				sameConveyance = false
			}
			dest := func(t *Transport) string {
				if t.FinalDestination != "" {
					return t.FinalDestination
				}
				return t.PortOfDischarge
			}
			if Compare(dest(docs[0].Transport), dest(d.Transport), e.T) == MatchConflict {
				sameDestination = false
			}
		}
		for _, d := range docs {
			if d.Transport == nil || d.Transport.Mode != ModeCourier {
				allCourier = false
			}
		}

		switch {
		case allCourier:
			out = append(out, e.find(SeverityCleared, "Several courier receipts are not a partial shipment",
				plural(len(docs), "courier receipt is", "courier receipts are")+" presented. A presentation of more than one courier receipt, post receipt or certificate of posting is not a partial shipment if they appear to have been stamped or signed by the same courier or postal service at the same place and date and for the same destination.",
				"UCP 600 art. 31(c)",
				"Check the stamps carry the same place and the same date. Two receipts stamped on different days are two shipments.",
				labels(docs)...))
		case sameConveyance && sameDestination:
			out = append(out, e.find(SeverityCleared, "Several sets of transport documents are not a partial shipment here",
				plural(len(docs), "set of transport documents is", "sets of transport documents are")+" presented, on the same means of conveyance for the same journey and to the same destination. A presentation of that kind is not regarded as covering a partial shipment, even if the documents show different dates of shipment or different ports of loading — and where they do, the latest date of shipment on any of them is the date of shipment.",
				"UCP 600 art. 31(b)",
				"Nothing to fix. Note that the latest of the shipment dates is the one the deadlines are counted from, which this check has already applied.",
				labels(docs)...))
		default:
			out = append(out, e.find(SeverityDiscrepancy, "The presentation looks like a partial shipment, which the credit forbids",
				plural(len(docs), "set of transport documents is", "sets of transport documents are")+
					" presented, and they do not appear to be on the same means of conveyance for the same journey to the same destination. Partial shipments are prohibited by this credit.",
				"UCP 600 arts. 31(a), 31(b)",
				"Present one shipment, or obtain an amendment allowing partial shipment. Where the goods did in fact travel on one vessel and one voyage, make sure both documents say so — the exception in article 31(b) depends on that appearing on the documents.",
				labels(docs)...))
		}
	}

	// A short quantity under a credit that forbids partial shipment is itself
	// a partial shipment, whatever the documents say about conveyances.
	if short, known := e.quantityShortOfCredit(); known && short {
		shipped, unit, _ := e.shippedQuantity()
		out = append(out, e.find(SeverityDiscrepancy, "Less than the full quantity under a credit that forbids partial shipment",
			"The credit calls for "+Quantity(e.C.Quantity, e.C.QuantityUnit)+" and the documents show "+Quantity(shipped, unit)+
				". Shipping part of the quantity is a partial shipment, which this credit prohibits. The quantity tolerance in article 30(b), where it applies, is the only slack you have.",
			"UCP 600 art. 31(a)",
			"Ship the balance and present one set, or obtain an amendment. Check the quantity rule above as well: if the shortfall is inside the article 30(b) tolerance, it is not a partial shipment at all.",
			e.quantityDocs()...))
	}

	if len(out) == 0 {
		out = append(out, e.find(SeverityCleared, "One shipment, as the credit requires",
			"The credit prohibits partial shipment and the presentation covers a single shipment.",
			"UCP 600 art. 31(a)", "", labels(docs)...))
	}
	return out
}

func ruleOnBoard(e *exam) []Finding {
	docs, ok := e.transportDocs()
	if !ok {
		return nil
	}

	wanted := false
	for _, r := range e.C.RequiredDocuments {
		if r.Kind == KindTransport && r.RequireOnBoard {
			wanted = true
		}
	}

	var out []Finding
	for _, d := range docs {
		if d.Transport == nil {
			continue
		}
		switch d.Transport.Mode {
		case ModeBillOfLading, ModeSeaWaybill, ModeCharterParty:
		default:
			continue
		}
		evidenced := d.Transport.PreprintedOnBoard || !d.Transport.OnBoardDate.Zero()
		switch {
		case evidenced && !d.Transport.OnBoardDate.Zero():
			out = append(out, e.find(SeverityCleared, "On board, with a dated notation",
				d.Label()+" carries an on board notation dated "+d.Transport.OnBoardDate.Human()+
					", and that date is the date of shipment — not the date the document was issued.",
				"UCP 600 art. 20(b)", "", d.Label()))
		case evidenced:
			out = append(out, e.find(SeverityCleared, "On board by the document's pre-printed wording",
				d.Label()+" says the goods are shipped on board in its printed wording, which article 20(a)(ii) accepts without a separate notation. With no notation, the date of issuance ("+
					d.Date.Human()+") is the date of shipment.",
				"UCP 600 arts. 20(a)(ii), 20(b)",
				"Where the document is a received-for-shipment form, the printed wording is not enough — an on board notation with a date is needed, and its date then becomes the date of shipment.",
				d.Label()))
		case wanted:
			out = append(out, e.find(SeverityDiscrepancy, "No evidence that the goods are on board",
				"The credit calls for an on board bill of lading and "+d.Label()+
					" shows neither pre-printed shipped-on-board wording nor an on board notation. A bill of lading must indicate that the goods have been shipped on board a named vessel at the port of loading stated in the credit, by pre-printed wording or by an on board notation indicating the date on which the goods have been shipped on board.",
				"UCP 600 art. 20(a)(ii)",
				"Have the carrier add a dated on board notation. Do it while the document is still with the agent: the notation's date becomes the date of shipment, and if it falls after the latest shipment date you have turned one problem into two.",
				d.Label()))
		default:
			out = append(out, e.find(SeverityCaution, "Nothing shows the goods on board",
				d.Label()+" shows neither pre-printed shipped-on-board wording nor an on board notation. The credit as entered does not ask for an on board document, so the date of issuance stands as the date of shipment — but most credits do ask.",
				"UCP 600 arts. 20(a)(ii), 20(b)",
				"Read field 46A of the credit again. \"Full set clean on board ocean bills of lading\" is the commonest wording in the trade, and it does require the on board evidence.",
				d.Label()))
		}
	}

	if len(out) == 0 {
		return e.skip("a bill of lading or sea waybill (on board evidence is not in point for other modes)")
	}
	return out
}

// labels names a list of documents.
func labels(docs []Document) []string {
	out := make([]string, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.Label())
	}
	return out
}
