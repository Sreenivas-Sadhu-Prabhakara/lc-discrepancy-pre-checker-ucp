package ucp

import "testing"

// transportCredit is a credit with the shipment terms a transport rule needs.
func transportCredit() Credit {
	c := minimalCredit()
	c.Amount = 10000
	c.ExpiryDate = MustDate("2026-04-30")
	c.PortOfLoading = "Tuticorin"
	c.PortOfDischarge = "Manila"
	return c
}

// transportDoc builds a transport document of any mode for a test.
func transportDoc(mode Mode, fn func(*Document)) Document {
	d := Document{
		Kind:      KindTransport,
		Title:     mode.Title(),
		Date:      MustDate("2026-03-10"),
		Originals: 1,
		Transport: &Transport{
			Mode:            mode,
			OriginalsIssued: 1,
			Vessel:          "MV Kaveri Star",
			Voyage:          "118W",
			PortOfLoading:   "Tuticorin",
			PortOfDischarge: "Manila",
		},
	}
	switch mode {
	case ModeBillOfLading, ModeSeaWaybill, ModeCharterParty:
		d.Transport.PreprintedOnBoard = true
	}
	if fn != nil {
		fn(&d)
	}
	return d
}

func TestRuleCleanFollowsArticle27(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("no transport document", func(t *testing.T) {
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "T1-CLEAN")
	})

	t.Run("a clean document, and the word \"clean\" need not appear", func(t *testing.T) {
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantFinding(t, r, "T1-CLEAN", SeverityCleared, "art. 27", "need not appear")
	})

	t.Run("a clause declaring the goods defective", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) {
			d.Transport.AdverseClause = "Three cartons torn and restitched"
		})
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T1-CLEAN", SeverityDiscrepancy, "art. 27", "letter of indemnity", "torn and restitched")
	})
}

func TestRuleOnDeckFollowsArticle26a(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("nothing said about deck stowage", func(t *testing.T) {
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantSkipped(t, r, "T2-ON-DECK")
	})

	t.Run("the goods are on deck", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) { d.Transport.OnDeck = true })
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T2-ON-DECK", SeverityDiscrepancy, "art. 26(a)", "under deck")
	})

	t.Run("a liberty clause saying they may be is acceptable", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) { d.Transport.MayBeOnDeck = true })
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T2-ON-DECK", SeverityCleared, "art. 26(a)", "Leave it")
	})
}

func TestRuleCharterParty(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("a liner bill under a credit that says nothing", func(t *testing.T) {
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantSkipped(t, r, "T3-CHARTER-PARTY")
	})

	t.Run("a charter party bill the credit never asked for", func(t *testing.T) {
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeCharterParty, nil)}}, th)
		wantFinding(t, r, "T3-CHARTER-PARTY", SeverityDiscrepancy, "arts. 20(a)(vi), 22", "liner bill of lading")
	})

	t.Run("a bill that merely says it is subject to a charter party", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) { d.Transport.SubjectToCharterParty = true })
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T3-CHARTER-PARTY", SeverityDiscrepancy, "arts. 20(a)(vi), 22")
	})

	t.Run("a charter party bill the credit calls for", func(t *testing.T) {
		credit := transportCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindTransport, Wording: "charter party bill of lading", Originals: 1},
		}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeCharterParty, nil)}}, th)
		wantFinding(t, r, "T3-CHARTER-PARTY", SeverityCleared, "art. 22")
	})

	t.Run("a liner bill where the credit expected a charter party one", func(t *testing.T) {
		credit := transportCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindTransport, Wording: "charter party bill of lading", Originals: 1},
		}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantFinding(t, r, "T3-CHARTER-PARTY", SeverityCaution, "art. 22")
	})
}

func TestRulePorts(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("no ports on the credit", func(t *testing.T) {
		credit := minimalCredit()
		credit.ExpiryDate = MustDate("2026-04-30")
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantSkipped(t, r, "T4-PORTS")
	})

	t.Run("the credit's ports", func(t *testing.T) {
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantFinding(t, r, "T4-PORTS", SeverityCleared, "2 places were compared")
	})

	t.Run("a port with the country appended is the same port", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) { d.Transport.PortOfLoading = "Tuticorin, India" })
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantNoFinding(t, r, "T4-PORTS", SeverityDiscrepancy)
	})

	t.Run("the wrong port of discharge", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) { d.Transport.PortOfDischarge = "Cebu" })
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T4-PORTS", SeverityDiscrepancy, "art. 20(a)(iii)", "amendment before shipping")
	})

	t.Run("a port spelt slightly differently", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) { d.Transport.PortOfLoading = "Tuticorrin" })
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T4-PORTS", SeverityCaution, "art. 20(a)(ii)")
	})

	t.Run("a multimodal document is judged on its places, not its ports", func(t *testing.T) {
		credit := transportCredit()
		credit.PlaceOfReceipt = "Coimbatore"
		credit.FinalDestination = "Manila"
		doc := transportDoc(ModeMultimodal, func(d *Document) {
			d.Transport.PlaceOfReceipt = "Erode"
			d.Transport.FinalDestination = "Manila"
		})
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T4-PORTS", SeverityDiscrepancy, "art. 19(a)(ii)", "place of taking in charge")
	})
}

func TestRuleFullSet(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("nothing entered about the set", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) {
			d.Originals = 0
			d.Transport.OriginalsIssued = 0
		})
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantSkipped(t, r, "T5-FULL-SET")
	})

	t.Run("the full set is there", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) {
			d.Originals = 3
			d.Transport.OriginalsIssued = 3
		})
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T5-FULL-SET", SeverityCleared, "art. 20(a)(iv)")
	})

	t.Run("two of three originals", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) {
			d.Originals = 2
			d.Transport.OriginalsIssued = 3
		})
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T5-FULL-SET", SeverityDiscrepancy, "art. 20(a)(iv)", "document of title")
	})

	t.Run("an air waybill needs only the shipper's original", func(t *testing.T) {
		credit := transportCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindTransport, Wording: "full set of air waybills", Originals: 3},
		}
		doc := transportDoc(ModeAir, func(d *Document) {
			d.Originals = 1
			d.Transport.OriginalsIssued = 3
		})
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T5-FULL-SET", SeverityCleared, "art. 23(a)(v)", "all three originals")
		wantNoFinding(t, r, "T5-FULL-SET", SeverityDiscrepancy)
	})
}

func TestRuleTranshipmentBitesOnlyWhereTheRulesLetIt(t *testing.T) {
	th := StatutoryThresholds()

	prohibited := func() Credit {
		c := transportCredit()
		c.Transhipment = PermissionProhibited
		return c
	}

	t.Run("a credit that does not prohibit transhipment", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) { d.Transport.Transhipment = TranshipmentWill })
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantSkipped(t, r, "T6-TRANSHIPMENT")
	})

	t.Run("no transhipment shown", func(t *testing.T) {
		r := Check(prohibited(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantFinding(t, r, "T6-TRANSHIPMENT", SeverityCleared, "far weaker than it looks")
	})

	cases := []struct {
		name      string
		mode      Mode
		container bool
		state     Transhipment
		want      Severity
		mentions  []string
	}{
		{
			name: "an air waybill — the prohibition has no effect",
			mode: ModeAir, state: TranshipmentWill,
			want: SeverityCleared, mentions: []string{"art. 23(c)(ii)"},
		},
		{
			name: "a road or rail document — no effect either",
			mode: ModeRoadRail, state: TranshipmentWill,
			want: SeverityCleared, mentions: []string{"art. 24(e)(ii)"},
		},
		{
			name: "a multimodal document — no effect",
			mode: ModeMultimodal, state: TranshipmentWill,
			want: SeverityCleared, mentions: []string{"art. 19(c)"},
		},
		{
			name: "a containerised bill of lading — acceptable under art. 20(c)(ii)",
			mode: ModeBillOfLading, container: true, state: TranshipmentWill,
			want: SeverityCleared, mentions: []string{"art. 20(c)(ii)", "container number"},
		},
		{
			name: "a containerised sea waybill cites its own sub-article",
			mode: ModeSeaWaybill, container: true, state: TranshipmentWill,
			want: SeverityCleared, mentions: []string{"art. 21(c)(ii)"},
		},
		{
			name: "a break-bulk bill of lading that will tranship — a real discrepancy",
			mode: ModeBillOfLading, state: TranshipmentWill,
			want: SeverityDiscrepancy, mentions: []string{"art. 20(c)(ii)", "direct vessel"},
		},
		{
			name: "a break-bulk bill with a liberty to tranship — art. 20(d) disregards the clause",
			mode: ModeBillOfLading, state: TranshipmentMay,
			want: SeverityCaution, mentions: []string{"arts. 20(d), 20(c)(ii)", "printed liberty clause"},
		},
		{
			name: "a document outside arts. 19 to 25 — the credit's own term is read as it stands",
			mode: ModeOther, state: TranshipmentWill,
			want: SeverityCaution, mentions: []string{"outside articles 19 to 25"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := transportDoc(c.mode, func(d *Document) {
				d.Transport.Transhipment = c.state
				d.Transport.Containerised = c.container
			})
			r := Check(prohibited(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
			wantFinding(t, r, "T6-TRANSHIPMENT", c.want, c.mentions...)
		})
	}
}

func TestRulePartialShipmentFollowsArticle31(t *testing.T) {
	th := StatutoryThresholds()

	prohibited := func() Credit {
		c := transportCredit()
		c.PartialShipments = PermissionProhibited
		return c
	}

	t.Run("a credit that allows partial shipment", func(t *testing.T) {
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantSkipped(t, r, "T7-PARTIAL")
	})

	t.Run("one shipment under a credit that forbids partial shipment", func(t *testing.T) {
		r := Check(prohibited(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantFinding(t, r, "T7-PARTIAL", SeverityCleared, "art. 31(a)")
	})

	t.Run("two sets on the same vessel and voyage are not a partial shipment", func(t *testing.T) {
		a := transportDoc(ModeBillOfLading, func(d *Document) { d.Title = "bill of lading one" })
		b := transportDoc(ModeBillOfLading, func(d *Document) {
			d.Title = "bill of lading two"
			d.Date = MustDate("2026-03-11")
		})
		r := Check(prohibited(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{a, b}}, th)
		wantFinding(t, r, "T7-PARTIAL", SeverityCleared, "art. 31(b)", "latest of the shipment dates")
	})

	t.Run("two sets on different vessels are a partial shipment", func(t *testing.T) {
		a := transportDoc(ModeBillOfLading, func(d *Document) { d.Title = "bill of lading one" })
		b := transportDoc(ModeBillOfLading, func(d *Document) {
			d.Title = "bill of lading two"
			d.Transport.Vessel = "MV Other Star"
			d.Transport.PortOfDischarge = "Cebu"
		})
		r := Check(prohibited(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{a, b}}, th)
		wantFinding(t, r, "T7-PARTIAL", SeverityDiscrepancy, "arts. 31(a), 31(b)")
	})

	t.Run("several courier receipts are not a partial shipment", func(t *testing.T) {
		a := transportDoc(ModeCourier, func(d *Document) { d.Title = "courier receipt one" })
		b := transportDoc(ModeCourier, func(d *Document) { d.Title = "courier receipt two" })
		r := Check(prohibited(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{a, b}}, th)
		wantFinding(t, r, "T7-PARTIAL", SeverityCleared, "art. 31(c)", "same place and the same date")
	})

	t.Run("a short quantity is itself a partial shipment", func(t *testing.T) {
		credit := prohibited()
		credit.Quantity = 100
		credit.QuantityUnit = "sets"
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.Quantity = 80
		inv.QuantityUnit = "sets"
		inv.Invoice.Quantity = 80
		inv.Invoice.Unit = "sets"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{inv, transportDoc(ModeBillOfLading, nil)}}, th)
		wantFinding(t, r, "T7-PARTIAL", SeverityDiscrepancy, "art. 31(a)", "80 sets")
	})
}

func TestRuleOnBoardFollowsArticle20(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("an air waybill is not in point", func(t *testing.T) {
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeAir, nil)}}, th)
		wantSkipped(t, r, "T8-ON-BOARD")
	})

	t.Run("a dated on board notation", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) {
			d.Transport.PreprintedOnBoard = false
			d.Transport.OnBoardDate = MustDate("2026-03-12")
		})
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T8-ON-BOARD", SeverityCleared, "art. 20(b)", "12 March 2026", "not the date the document was issued")
	})

	t.Run("pre-printed shipped-on-board wording", func(t *testing.T) {
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{transportDoc(ModeBillOfLading, nil)}}, th)
		wantFinding(t, r, "T8-ON-BOARD", SeverityCleared, "arts. 20(a)(ii), 20(b)", "received-for-shipment")
	})

	t.Run("no on board evidence under a credit that asks for it", func(t *testing.T) {
		credit := transportCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindTransport, Wording: "full set clean on board ocean bills of lading", Originals: 3},
		}
		doc := transportDoc(ModeBillOfLading, func(d *Document) {
			d.Transport.PreprintedOnBoard = false
			d.Originals = 3
			d.Transport.OriginalsIssued = 3
		})
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T8-ON-BOARD", SeverityDiscrepancy, "art. 20(a)(ii)", "latest shipment date")
	})

	t.Run("no on board evidence where the credit does not ask for it", func(t *testing.T) {
		doc := transportDoc(ModeBillOfLading, func(d *Document) { d.Transport.PreprintedOnBoard = false })
		r := Check(transportCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "T8-ON-BOARD", SeverityCaution, "field 46A")
	})
}
