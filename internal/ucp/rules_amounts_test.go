package ucp

import "testing"

// amountCredit is a credit with the figures a money rule needs and nothing else.
func amountCredit() Credit {
	c := minimalCredit()
	c.Amount = 10000
	c.ExpiryDate = MustDate("2026-04-30")
	return c
}

func TestRuleAmountCeiling(t *testing.T) {
	th := StatutoryThresholds()
	cases := []struct {
		name      string
		qualifier Qualifier
		plusPct   float64
		drawn     float64
		want      Severity
		mentions  []string
		skip      bool
	}{
		{name: "no drawing and no invoice", skip: true},
		{name: "drawing the credit amount", drawn: 10000, want: SeverityCleared, mentions: []string{"USD 0.00 of headroom"}},
		{name: "drawing under the amount", drawn: 9000, want: SeverityCleared, mentions: []string{"USD 1000.00 of headroom"}},
		{name: "a penny over the amount", drawn: 10000.01, want: SeverityDiscrepancy, mentions: []string{"Article 18(b)", "art. 30(a)"}},
		{name: "well over the amount", drawn: 12000, want: SeverityDiscrepancy, mentions: []string{"over by USD 2000.00"}},
		{name: "\"about\" allows ten per cent more", qualifier: QualifierAbout, drawn: 11000, want: SeverityCleared, mentions: []string{"art. 30(a)"}},
		{name: "\"about\" does not allow eleven per cent more", qualifier: QualifierAbout, drawn: 11001, want: SeverityDiscrepancy},
		{name: "\"maximum\" allows nothing over", qualifier: QualifierMaximum, drawn: 10000.5, want: SeverityDiscrepancy, mentions: []string{"caps the amount"}},
		{name: "a tolerance the credit states is applied", plusPct: 5, drawn: 10500, want: SeverityCleared, mentions: []string{"stated in the credit"}},
		{name: "and its ceiling is enforced", plusPct: 5, drawn: 10501, want: SeverityDiscrepancy},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := amountCredit()
			credit.AmountQualifier = c.qualifier
			credit.TolerancePlusPercent = c.plusPct
			p := Presentation{Date: MustDate("2026-03-20"), DrawnCurrency: "USD", DrawnAmount: c.drawn}
			r := Check(credit, p, th)
			if c.skip {
				wantSkipped(t, r, "A1-AMOUNT")
				return
			}
			wantFinding(t, r, "A1-AMOUNT", c.want, c.mentions...)
		})
	}

	t.Run("the invoice total stands in for a drawing that was not entered", func(t *testing.T) {
		credit := amountCredit()
		inv := invoiceFor(credit, 12000)
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantFinding(t, r, "A1-AMOUNT", SeverityDiscrepancy, "USD 12000.00")
	})
}

func TestRuleShortDrawFollowsArticle30c(t *testing.T) {
	th := StatutoryThresholds()

	cases := []struct {
		name           string
		partial        Permission
		quantity       float64
		shipped        float64
		unitPrice      float64
		invUnitPrice   float64
		invoiced       float64
		drawn          float64
		qualifier      Qualifier
		tolerancePlus  float64
		toleranceMinus float64
		want           Severity
		mentions       []string
		skip           bool
	}{
		{name: "drawing in full leaves nothing to test", partial: PermissionProhibited, drawn: 10000, skip: true},
		{
			name:    "a short drawing where partial shipment is allowed is just a partial drawing",
			partial: PermissionAllowed, drawn: 4000,
			want: SeverityCleared, mentions: []string{"art. 31(a)", "Partial drawings and shipments are allowed"},
		},
		{
			name:  "a short drawing where the credit is silent is also allowed",
			drawn: 4000, want: SeverityCleared, mentions: []string{"art. 31(a)"},
		},
		{
			name:    "5% less with the full quantity shipped and the price unchanged is allowed",
			partial: PermissionProhibited, quantity: 100, shipped: 100, unitPrice: 100, invoiced: 9500, drawn: 9500,
			want: SeverityCleared, mentions: []string{"art. 30(c)", "one way only"},
		},
		{
			name:    "6% less is beyond the tolerance",
			partial: PermissionProhibited, quantity: 100, shipped: 100, unitPrice: 100, invoiced: 9400, drawn: 9400,
			want: SeverityDiscrepancy, mentions: []string{"6%", "5%"},
		},
		{
			name:    "the tolerance is lost when the quantity is short",
			partial: PermissionProhibited, quantity: 100, shipped: 90, unitPrice: 100, invoiced: 9500, drawn: 9500,
			want: SeverityDiscrepancy, mentions: []string{"not been shipped in full"},
		},
		{
			name:    "the tolerance is lost when the unit price is cut",
			partial: PermissionProhibited, quantity: 100, shipped: 100, unitPrice: 100, invUnitPrice: 95, invoiced: 9500, drawn: 9500,
			want: SeverityDiscrepancy, mentions: []string{"unit price stated in the credit has been reduced"},
		},
		{
			// Article 30(c)'s last sentence gives way to an amount worded as
			// "about", so the 10% of article 30(a) governs instead — and
			// article 30(c)'s conditions do not travel with it.
			name:    "\"about\" gives ten per cent down and displaces article 30(c)",
			partial: PermissionProhibited, quantity: 100, shipped: 100, unitPrice: 100, invoiced: 9500, drawn: 9500,
			qualifier: QualifierAbout,
			want:      SeverityCleared, mentions: []string{"art. 30(a)", "gives way"},
		},
		{
			name:    "and eleven per cent down is outside even that",
			partial: PermissionProhibited, quantity: 100, shipped: 100, unitPrice: 100, invoiced: 8900, drawn: 8900,
			qualifier: QualifierAbout,
			want:      SeverityDiscrepancy, mentions: []string{"10%"},
		},
		{
			// A credit that stipulates a specific tolerance displaces article
			// 30(c) too. One of +5/-0 therefore allows nothing downward.
			name:    "a stated tolerance with nothing downward allows no short drawing",
			partial: PermissionProhibited, quantity: 100, shipped: 100, unitPrice: 100, invoiced: 9900, drawn: 9900,
			tolerancePlus: 5,
			want:          SeverityDiscrepancy, mentions: []string{"the tolerance the credit states", "0%"},
		},
		{
			name:    "a stated downward tolerance is applied as stated",
			partial: PermissionProhibited, quantity: 100, shipped: 100, unitPrice: 100, invoiced: 9800, drawn: 9800,
			toleranceMinus: 3,
			want:           SeverityCleared, mentions: []string{"the tolerance the credit states"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := amountCredit()
			credit.PartialShipments = c.partial
			credit.Quantity = c.quantity
			credit.QuantityUnit = "sets"
			credit.UnitPrice = c.unitPrice
			credit.AmountQualifier = c.qualifier
			credit.TolerancePlusPercent = c.tolerancePlus
			credit.ToleranceMinusPercent = c.toleranceMinus

			var docs []Document
			if c.invoiced > 0 {
				inv := invoiceFor(credit, c.invoiced)
				inv.Date = MustDate("2026-03-10")
				inv.Quantity = c.shipped
				inv.QuantityUnit = "sets"
				inv.Invoice.Quantity = c.shipped
				inv.Invoice.Unit = "sets"
				invoicePrice := c.invUnitPrice
				if invoicePrice == 0 {
					invoicePrice = c.unitPrice
				}
				inv.Invoice.UnitPrice = invoicePrice
				docs = append(docs, inv)
			}

			r := Check(credit, Presentation{
				Date: MustDate("2026-03-20"), DrawnCurrency: "USD", DrawnAmount: c.drawn, Documents: docs,
			}, th)
			if c.skip {
				wantSkipped(t, r, "A2-SHORT-DRAW")
				return
			}
			wantFinding(t, r, "A2-SHORT-DRAW", c.want, c.mentions...)
		})
	}
}

func TestRuleQuantityFollowsArticle30b(t *testing.T) {
	th := StatutoryThresholds()

	cases := []struct {
		name         string
		quantity     float64
		packingUnits bool
		qualifier    Qualifier
		shipped      float64
		drawn        float64
		want         Severity
		mentions     []string
		skip         bool
	}{
		{name: "no quantity in the credit", shipped: 100, skip: true},
		{name: "no quantity on the documents", quantity: 100, skip: true},
		{name: "exactly the quantity", quantity: 100, shipped: 100, want: SeverityCleared, mentions: []string{"show the same"}},
		{name: "5% more is inside the tolerance", quantity: 100, shipped: 105, want: SeverityCleared, mentions: []string{"art. 30(b)"}},
		{name: "5% less is inside the tolerance", quantity: 100, shipped: 95, want: SeverityCleared},
		{name: "6% more is outside", quantity: 100, shipped: 106, want: SeverityDiscrepancy, mentions: []string{"ceiling"}},
		{name: "6% less is outside", quantity: 100, shipped: 94, want: SeverityDiscrepancy, mentions: []string{"floor"}},
		{
			name:     "goods counted in packing units get no tolerance at all",
			quantity: 100, packingUnits: true, shipped: 101,
			want: SeverityDiscrepancy, mentions: []string{"packing units"},
		},
		{
			name:     "but the exact count passes",
			quantity: 100, packingUnits: true, shipped: 100, want: SeverityCleared,
		},
		{
			name:     "\"about\" the quantity gives ten per cent",
			quantity: 100, qualifier: QualifierAbout, shipped: 110,
			want: SeverityCleared, mentions: []string{"art. 30(a)"},
		},
		{
			name:     "drawings over the credit amount cost the tolerance",
			quantity: 100, shipped: 103, drawn: 12000,
			want: SeverityDiscrepancy, mentions: []string{"total of the drawings"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := amountCredit()
			credit.Quantity = c.quantity
			credit.QuantityUnit = "sets"
			credit.QuantityInPackingUnits = c.packingUnits
			credit.QuantityQualifier = c.qualifier

			var docs []Document
			if c.shipped > 0 {
				inv := invoiceFor(credit, 10000)
				inv.Date = MustDate("2026-03-10")
				inv.Quantity = c.shipped
				inv.QuantityUnit = "sets"
				inv.Invoice.Quantity = c.shipped
				inv.Invoice.Unit = "sets"
				docs = append(docs, inv)
			}
			drawn := c.drawn
			if drawn == 0 {
				drawn = 10000
			}

			r := Check(credit, Presentation{
				Date: MustDate("2026-03-20"), DrawnCurrency: "USD", DrawnAmount: drawn, Documents: docs,
			}, th)
			if c.skip {
				wantSkipped(t, r, "A3-QUANTITY")
				return
			}
			wantFinding(t, r, "A3-QUANTITY", c.want, c.mentions...)
		})
	}

	t.Run("a unit that does not match is flagged rather than silently assumed", func(t *testing.T) {
		credit := amountCredit()
		credit.Quantity = 100
		credit.QuantityUnit = "metric tonnes"
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.Quantity = 100
		inv.QuantityUnit = "pieces"
		inv.Invoice.Quantity = 100
		inv.Invoice.Unit = "pieces"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnAmount: 10000, DrawnCurrency: "USD",
			Documents: []Document{inv}}, th)
		wantFinding(t, r, "A3-QUANTITY", SeverityCleared, "assumes they are the same measure")
	})
}

func TestRuleCurrency(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("no currency in the credit", func(t *testing.T) {
		credit := amountCredit()
		credit.Currency = ""
		r := Check(credit, Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "A4-CURRENCY")
	})

	t.Run("no currency anywhere on the documents", func(t *testing.T) {
		credit := amountCredit()
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{
			{Kind: KindPackingList, Title: "packing list", Originals: 1},
		}}, th)
		wantSkipped(t, r, "A4-CURRENCY")
	})

	t.Run("everything in the credit's currency", func(t *testing.T) {
		credit := amountCredit()
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnCurrency: "USD",
			Documents: []Document{invoiceFor(credit, 10000)}}, th)
		wantFinding(t, r, "A4-CURRENCY", SeverityCleared)
	})

	t.Run("an invoice in another currency", func(t *testing.T) {
		credit := amountCredit()
		inv := invoiceFor(credit, 10000)
		inv.Invoice.Currency = "INR"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantFinding(t, r, "A4-CURRENCY", SeverityDiscrepancy, "art. 18(a)(iii)", "INR")
	})

	t.Run("insurance in another currency", func(t *testing.T) {
		credit := amountCredit()
		ins := Document{Kind: KindInsurance, Title: "insurance certificate", Originals: 1,
			Date: MustDate("2026-03-10"), Issuer: "Insurer",
			Insurance: &Insurance{Currency: "INR", Amount: 900000, IssuedBy: "Insurer"}}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{ins}}, th)
		wantFinding(t, r, "A4-CURRENCY", SeverityDiscrepancy, "art. 28(f)(i)")
	})

	t.Run("a draft and a drawing in another currency", func(t *testing.T) {
		credit := amountCredit()
		dr := Document{Kind: KindDraft, Title: "draft", Originals: 1, Date: MustDate("2026-03-10"),
			Draft: &Draft{Currency: "INR", Amount: 900000}}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnCurrency: "INR",
			Documents: []Document{dr}}, th)
		found := findingsFor(r, "A4-CURRENCY")
		if len(found) < 2 {
			t.Fatalf("both the draft and the drawing should be flagged, got %d findings", len(found))
		}
	})
}

func TestRuleArithmetic(t *testing.T) {
	th := StatutoryThresholds()

	build := func(quantity, unitPrice, total, creditUnitPrice float64, qualifier Qualifier) Result {
		credit := amountCredit()
		credit.UnitPrice = creditUnitPrice
		credit.UnitPriceQualifier = qualifier
		inv := invoiceFor(credit, total)
		inv.Date = MustDate("2026-03-10")
		inv.Invoice.Quantity = quantity
		inv.Invoice.Unit = "sets"
		inv.Invoice.UnitPrice = unitPrice
		return Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnCurrency: "USD", DrawnAmount: total,
			Documents: []Document{inv}}, th)
	}

	t.Run("no invoice", func(t *testing.T) {
		r := Check(amountCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "A5-ARITHMETIC")
	})

	t.Run("an invoice without the figures", func(t *testing.T) {
		credit := amountCredit()
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"),
			Documents: []Document{invoiceFor(credit, 10000)}}, th)
		wantSkipped(t, r, "A5-ARITHMETIC")
	})

	t.Run("the arithmetic holds", func(t *testing.T) {
		wantFinding(t, build(100, 100, 10000, 100, QualifierExact), "A5-ARITHMETIC", SeverityCleared)
	})

	t.Run("the arithmetic does not hold", func(t *testing.T) {
		wantFinding(t, build(100, 100, 9500, 100, QualifierExact), "A5-ARITHMETIC", SeverityCaution,
			"USD 10000.00", "art. 14(d)")
	})

	t.Run("a rounding remainder is not a failure", func(t *testing.T) {
		wantFinding(t, build(3, 33.333, 99.999, 0, QualifierExact), "A5-ARITHMETIC", SeverityCleared)
	})

	t.Run("a unit price above the credit's", func(t *testing.T) {
		wantFinding(t, build(100, 110, 11000, 100, QualifierExact), "A5-ARITHMETIC", SeverityDiscrepancy,
			"not the credit's")
	})

	t.Run("a unit price below the credit's warns about article 30(c)", func(t *testing.T) {
		r := build(100, 95, 9500, 100, QualifierAbout)
		wantFinding(t, r, "A5-ARITHMETIC", SeverityCaution, "art. 30(c)", "not reduced")
	})

	t.Run("\"about\" the unit price widens the band", func(t *testing.T) {
		r := build(100, 109, 10900, 100, QualifierAbout)
		wantNoFinding(t, r, "A5-ARITHMETIC", SeverityDiscrepancy)
	})
}

func TestRuleInsuranceAmountFollowsArticle28f(t *testing.T) {
	th := StatutoryThresholds()

	insure := func(amount float64) Document {
		return Document{Kind: KindInsurance, Title: "insurance certificate", Originals: 1,
			Date: MustDate("2026-03-08"), Issuer: "Bharat General Insurance",
			Insurance: &Insurance{Currency: "USD", Amount: amount, IssuedBy: "Bharat General Insurance", Endorsed: true}}
	}

	t.Run("no insurance document and the credit does not ask for one", func(t *testing.T) {
		r := Check(amountCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "A6-INSURANCE-AMOUNT")
	})

	t.Run("the credit asks for insurance and none is presented", func(t *testing.T) {
		credit := amountCredit()
		credit.RequiredDocuments = []Requirement{{Kind: KindInsurance, Wording: "insurance certificate", Originals: 1}}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20")}, th)
		wantFinding(t, r, "A6-INSURANCE-AMOUNT", SeverityDiscrepancy, "cover note")
	})

	t.Run("110% of the CIF value where the credit is silent", func(t *testing.T) {
		credit := amountCredit()
		credit.TradeTerm = "CIF MANILA"
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-08")
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnCurrency: "USD", DrawnAmount: 10000,
			Documents: []Document{inv, insure(11000)}}, th)
		wantFinding(t, r, "A6-INSURANCE-AMOUNT", SeverityCleared, "110%", "art. 28(f)(ii)", "CIF MANILA value")
	})

	t.Run("a rupee under 110% is a discrepancy", func(t *testing.T) {
		credit := amountCredit()
		credit.TradeTerm = "CIF MANILA"
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-08")
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnCurrency: "USD", DrawnAmount: 10000,
			Documents: []Document{inv, insure(10999.99)}}, th)
		wantFinding(t, r, "A6-INSURANCE-AMOUNT", SeverityDiscrepancy, "short by", "minimum, not a target")
	})

	t.Run("a percentage the credit states is used instead", func(t *testing.T) {
		credit := amountCredit()
		credit.TradeTerm = "CIF MANILA"
		credit.InsurancePercent = 120
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-08")
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnCurrency: "USD", DrawnAmount: 10000,
			Documents: []Document{inv, insure(11000)}}, th)
		wantFinding(t, r, "A6-INSURANCE-AMOUNT", SeverityDiscrepancy, "120%", "the percentage the credit states")
	})

	t.Run("on FOB terms no CIF value can be worked out, so the fallback applies", func(t *testing.T) {
		credit := amountCredit()
		credit.TradeTerm = "FOB TUTICORIN"
		inv := invoiceFor(credit, 9000)
		inv.Date = MustDate("2026-03-08")
		// The greater of the drawing (10000) and the invoice value (9000).
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnCurrency: "USD", DrawnAmount: 10000,
			Documents: []Document{inv, insure(11000)}}, th)
		wantFinding(t, r, "A6-INSURANCE-AMOUNT", SeverityCleared, "no CIF or CIP value", "honour is requested")
	})

	t.Run("the invoice value wins the fallback where it is the greater", func(t *testing.T) {
		credit := amountCredit()
		credit.TradeTerm = "FOB TUTICORIN"
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-08")
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnCurrency: "USD", DrawnAmount: 8000,
			Documents: []Document{inv, insure(11000)}}, th)
		wantFinding(t, r, "A6-INSURANCE-AMOUNT", SeverityCleared, "gross value of the goods on the invoice")
	})

	t.Run("nothing to measure against", func(t *testing.T) {
		credit := amountCredit()
		credit.Amount = 0
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{insure(11000)}}, th)
		wantSkipped(t, r, "A6-INSURANCE-AMOUNT")
	})
}

func TestRuleDraft(t *testing.T) {
	th := StatutoryThresholds()

	draft := func(amount float64, tenor int, currency string) Document {
		return Document{Kind: KindDraft, Title: "draft", Originals: 1, Date: MustDate("2026-03-10"),
			Draft: &Draft{Currency: currency, Amount: amount, TenorDays: tenor, DrawnOn: "Banco de Manila"}}
	}

	t.Run("no draft wanted and none presented", func(t *testing.T) {
		r := Check(amountCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "A7-DRAFT")
	})

	t.Run("an acceptance credit with no draft", func(t *testing.T) {
		credit := amountCredit()
		credit.Availability = AvailabilityAcceptance
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnAmount: 10000, DrawnCurrency: "USD"}, th)
		wantFinding(t, r, "A7-DRAFT", SeverityDiscrepancy, "6(b)")
	})

	t.Run("a draft nobody asked for", func(t *testing.T) {
		credit := amountCredit()
		credit.Availability = AvailabilitySight
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnAmount: 10000, DrawnCurrency: "USD",
			Documents: []Document{draft(10000, 0, "USD")}}, th)
		wantFinding(t, r, "A7-DRAFT", SeverityCaution, "art. 14(g)", "field 42")
	})

	t.Run("a draft agreeing with the drawing", func(t *testing.T) {
		credit := amountCredit()
		credit.Availability = AvailabilityAcceptance
		credit.DraftTenorDays = 60
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnAmount: 10000, DrawnCurrency: "USD",
			Documents: []Document{draft(10000, 60, "USD")}}, th)
		wantFinding(t, r, "A7-DRAFT", SeverityCleared)
	})

	t.Run("a draft for the wrong amount", func(t *testing.T) {
		credit := amountCredit()
		credit.Availability = AvailabilityAcceptance
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnAmount: 10000, DrawnCurrency: "USD",
			Documents: []Document{draft(9000, 0, "USD")}}, th)
		wantFinding(t, r, "A7-DRAFT", SeverityDiscrepancy, "USD 9000.00", "art. 14(d)")
	})

	t.Run("a draft on the wrong tenor", func(t *testing.T) {
		credit := amountCredit()
		credit.Availability = AvailabilityAcceptance
		credit.DraftTenorDays = 60
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnAmount: 10000, DrawnCurrency: "USD",
			Documents: []Document{draft(10000, 90, "USD")}}, th)
		wantFinding(t, r, "A7-DRAFT", SeverityDiscrepancy, "90 days", "60 days")
	})

	t.Run("a draft the credit's document list calls for", func(t *testing.T) {
		credit := amountCredit()
		credit.Availability = AvailabilityNegotiation
		credit.RequiredDocuments = []Requirement{{Kind: KindDraft, Wording: "draft at sight", Originals: 1}}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), DrawnAmount: 10000, DrawnCurrency: "USD"}, th)
		wantFinding(t, r, "A7-DRAFT", SeverityDiscrepancy)
	})
}
