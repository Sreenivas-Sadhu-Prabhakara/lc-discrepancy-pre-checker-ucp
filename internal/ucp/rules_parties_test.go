package ucp

import "testing"

// goodsCredit is a credit with a description of the goods and the parties.
func goodsCredit() Credit {
	c := minimalCredit()
	c.Amount = 10000
	c.ExpiryDate = MustDate("2026-04-30")
	c.GoodsDescription = "Ring frame spindle assemblies model RSA-4200, 1800 sets, CIF Manila"
	return c
}

func TestRuleInvoiceGoodsFollowsArticle18c(t *testing.T) {
	th := StatutoryThresholds()

	cases := []struct {
		name        string
		description string
		want        Severity
		mentions    []string
		skip        bool
	}{
		{name: "no description on the invoice", description: "", skip: true},
		{
			name:        "the credit's description, word for word",
			description: "Ring frame spindle assemblies model RSA-4200, 1800 sets, CIF Manila",
			want:        SeverityCleared, mentions: []string{"art. 18(c)"},
		},
		{
			name:        "the credit's description in another case and spacing",
			description: "RING FRAME SPINDLE ASSEMBLIES  MODEL RSA-4200, 1800 SETS, CIF MANILA",
			want:        SeverityCleared,
		},
		{
			name:        "the credit's description with a purchase order number added",
			description: "Ring frame spindle assemblies model RSA-4200, 1800 sets, CIF Manila, against PO 88120",
			want:        SeverityCleared, mentions: []string{"says more"},
		},
		{
			name:        "a different model number is a conflict, never a typing error",
			description: "Ring frame spindle assemblies model RSA-4400, 1800 sets, CIF Manila",
			want:        SeverityDiscrepancy, mentions: []string{"model 123", "art. 18(c)"},
		},
		{
			name:        "a misspelling is a caution, not a discrepancy",
			description: "Ring frame spindel assemblies model RSA-4200, 1800 sets, CIF Manila",
			want:        SeverityCaution, mentions: []string{"misspelling or typing error", "Reprint"},
		},
		{
			name:        "a general description on the invoice will not do",
			description: "Spindle assemblies",
			want:        SeverityDiscrepancy, mentions: []string{"article 14(e)", "word for word"},
		},
		{
			name:        "wholly different goods",
			description: "Cotton yarn 40s count, 2000 kg",
			want:        SeverityDiscrepancy, mentions: []string{"no word in common"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := goodsCredit()
			inv := invoiceFor(credit, 10000)
			inv.Date = MustDate("2026-03-10")
			inv.GoodsDescription = c.description
			r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
			if c.skip {
				wantSkipped(t, r, "P1-INVOICE-GOODS")
				return
			}
			wantFinding(t, r, "P1-INVOICE-GOODS", c.want, c.mentions...)
		})
	}

	t.Run("no description in the credit", func(t *testing.T) {
		credit := minimalCredit()
		credit.Amount = 10000
		inv := invoiceFor(credit, 10000)
		inv.GoodsDescription = "Spindle assemblies"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantSkipped(t, r, "P1-INVOICE-GOODS")
	})

	t.Run("no invoice", func(t *testing.T) {
		r := Check(goodsCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "P1-INVOICE-GOODS")
	})
}

func TestRuleOtherGoodsFollowsArticle14e(t *testing.T) {
	th := StatutoryThresholds()

	other := func(description string) Document {
		return Document{Kind: KindPackingList, Title: "packing list", Originals: 1,
			Date: MustDate("2026-03-10"), GoodsDescription: description}
	}

	cases := []struct {
		name        string
		description string
		want        Severity
		mentions    []string
		skip        bool
	}{
		{name: "a document that says nothing about the goods is not tested at all", description: "", skip: true},
		{
			name:        "a general description is expressly allowed",
			description: "Spindle assemblies",
			want:        SeverityCleared, mentions: []string{"art. 14(e)", "Do not have these reprinted"},
		},
		{
			name:        "a general description with the packing added is still allowed",
			description: "Spindle assemblies, 150 cartons",
			want:        SeverityCleared, mentions: []string{"art. 14(e)"},
		},
		{
			name:        "the credit's own description is fine too",
			description: "Ring frame spindle assemblies model RSA-4200, 1800 sets, CIF Manila",
			want:        SeverityCleared,
		},
		{
			name:        "a different model number conflicts even here",
			description: "Spindle assemblies model RSA-4400",
			want:        SeverityDiscrepancy, mentions: []string{"arts. 14(e), 14(d)"},
		},
		{
			name:        "different goods altogether conflict",
			description: "Cotton yarn 40s count",
			want:        SeverityDiscrepancy, mentions: []string{"no word in common"},
		},
		{
			name:        "a misspelling outside the invoice is only a caution",
			description: "Ring frame spindel assemblies model RSA-4200, 1800 sets, CIF Manila",
			want:        SeverityCaution, mentions: []string{"art. 14(e)", "usually pass"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := goodsCredit()
			r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{other(c.description)}}, th)
			if c.skip {
				wantSkipped(t, r, "P2-OTHER-GOODS")
				return
			}
			wantFinding(t, r, "P2-OTHER-GOODS", c.want, c.mentions...)
		})
	}
}

func TestRuleInvoiceParties(t *testing.T) {
	th := StatutoryThresholds()

	cases := []struct {
		name     string
		issuer   string
		madeTo   string
		want     Severity
		mentions []string
	}{
		{
			name:   "both names as the credit gives them",
			issuer: "Coimbatore Spindle Works Private Limited", madeTo: "Sampaguita Textile Mills Inc",
			want: SeverityCleared,
		},
		{
			name:   "the ordinary abbreviations of a company name are the same name",
			issuer: "Coimbatore Spindle Works Pvt Ltd", madeTo: "Sampaguita Textile Mills Inc",
			want: SeverityCleared,
		},
		{
			name:   "a misspelling of your own name",
			issuer: "Coimbatore Spindel Works Private Limited", madeTo: "Sampaguita Textile Mills Inc",
			want: SeverityCaution, mentions: []string{"art. 18(a)(i)", "letter for letter"},
		},
		{
			name:   "a name written at another length",
			issuer: "Coimbatore Spindle Works", madeTo: "Sampaguita Textile Mills Inc",
			want: SeverityCaution, mentions: []string{"says more or less"},
		},
		{
			name:   "an invoice issued by somebody else",
			issuer: "Madurai Spindle Traders", madeTo: "Sampaguita Textile Mills Inc",
			want: SeverityDiscrepancy, mentions: []string{"art. 18(a)(i)", "amended"},
		},
		{
			name:   "an invoice billed to a buying agent",
			issuer: "Coimbatore Spindle Works Private Limited", madeTo: "Pacific Sourcing Agency Inc",
			want: SeverityDiscrepancy, mentions: []string{"art. 18(a)(ii)", "buying agent"},
		},
		{
			name:   "a misspelling of the buyer's name",
			issuer: "Coimbatore Spindle Works Private Limited", madeTo: "Sampaguita Textile Mils Inc",
			want: SeverityCaution, mentions: []string{"art. 18(a)(ii)", "purchase order"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := goodsCredit()
			inv := invoiceFor(credit, 10000)
			inv.Date = MustDate("2026-03-10")
			inv.Issuer = c.issuer
			inv.Invoice.MadeOutTo = Party{Name: c.madeTo, Country: "Philippines"}
			r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
			wantFinding(t, r, "P3-INVOICE-PARTIES", c.want, c.mentions...)
		})
	}

	t.Run("nothing entered for the issuer", func(t *testing.T) {
		credit := goodsCredit()
		inv := invoiceFor(credit, 10000)
		inv.Issuer = ""
		inv.IssuedByBeneficiary = false
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantFinding(t, r, "P3-INVOICE-PARTIES", SeverityVerify, "letterhead")
	})

	t.Run("nothing entered for the party billed", func(t *testing.T) {
		credit := goodsCredit()
		inv := invoiceFor(credit, 10000)
		inv.Invoice.MadeOutTo = Party{}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantFinding(t, r, "P3-INVOICE-PARTIES", SeverityVerify, "art. 18(a)(ii)")
	})

	t.Run("no names in the credit", func(t *testing.T) {
		credit := Credit{Currency: "USD", Amount: 10000, ExpiryDate: MustDate("2026-04-30")}
		inv := Document{Kind: KindInvoice, Title: "invoice", Originals: 1, Invoice: &Invoice{Currency: "USD", Amount: 10000}}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantSkipped(t, r, "P3-INVOICE-PARTIES")
	})
}

func TestRuleInvoiceSignature(t *testing.T) {
	th := StatutoryThresholds()

	cases := []struct {
		name     string
		wording  string
		signed   bool
		want     Severity
		mentions []string
	}{
		{
			name:    "the credit is silent and the invoice is unsigned — not a discrepancy",
			wording: "commercial invoice in triplicate", signed: false,
			want: SeverityCleared, mentions: []string{"need not be signed", "art. 18(a)", "field 46A"},
		},
		{
			name:    "the credit is silent and the invoice is signed anyway",
			wording: "commercial invoice in triplicate", signed: true,
			want: SeverityCleared, mentions: []string{"can do no harm"},
		},
		{
			name:    "the credit asks for a signed invoice and gets one",
			wording: "signed commercial invoice in triplicate", signed: true,
			want: SeverityCleared, mentions: []string{"as the credit asks"},
		},
		{
			name:    "the credit asks for a signed invoice and does not get one",
			wording: "signed commercial invoice in triplicate", signed: false,
			want: SeverityDiscrepancy, mentions: []string{"credit's own term governs", "Sign and stamp"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := goodsCredit()
			credit.RequiredDocuments = []Requirement{{Kind: KindInvoice, Wording: c.wording, Originals: 1, Copies: 2}}
			inv := invoiceFor(credit, 10000)
			inv.Date = MustDate("2026-03-10")
			inv.Originals = 1
			inv.Copies = 2
			inv.Signed = c.signed
			inv.Invoice.Signed = c.signed
			r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
			wantFinding(t, r, "P4-INVOICE-SIGNATURE", c.want, c.mentions...)
		})
	}

	t.Run("no invoice", func(t *testing.T) {
		r := Check(goodsCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "P4-INVOICE-SIGNATURE")
	})
}

func TestRuleAddressesFollowsArticle14j(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("no address to compare", func(t *testing.T) {
		r := Check(goodsCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "P5-ADDRESSES")
	})

	t.Run("a different address in the same country is not a discrepancy", func(t *testing.T) {
		credit := goodsCredit()
		credit.Applicant.Address = "118 Chino Roces Avenue, Makati City"
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.Invoice.MadeOutTo = Party{
			Name: credit.Applicant.Name, Address: "2F Robinsons Place, Cebu City", Country: "Philippines",
		}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantFinding(t, r, "P5-ADDRESSES", SeverityCleared, "art. 14(j)", "same country", "Leave it")
	})

	t.Run("an address in another country is a discrepancy", func(t *testing.T) {
		credit := goodsCredit()
		credit.Applicant.Address = "118 Chino Roces Avenue, Makati City"
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.Invoice.MadeOutTo = Party{
			Name: credit.Applicant.Name, Address: "12 Marina Boulevard, Singapore", Country: "India",
		}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantFinding(t, r, "P5-ADDRESSES", SeverityDiscrepancy, "another country", "art. 14(j)")
	})

	t.Run("a country this tool cannot place is a caution, not a refusal", func(t *testing.T) {
		credit := goodsCredit()
		credit.Applicant.Address = "1 Nguyen Hue, Ho Chi Minh City"
		credit.Applicant.Country = "Viet Nam"
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.Invoice.MadeOutTo = Party{
			Name: credit.Applicant.Name, Address: "88 Le Loi, Hanoi", Country: "Socialist Republic of Vietnam",
		}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantFinding(t, r, "P5-ADDRESSES", SeverityCaution, "could not confirm")
	})

	// The strict half: the applicant's address as consignee or notify party on
	// a transport document must be exactly the credit's.
	t.Run("the applicant's address as notify party must match the credit", func(t *testing.T) {
		credit := goodsCredit()
		credit.Applicant.Address = "118 Chino Roces Avenue, Makati City 1203"
		bl := billOfLading("2026-03-10", "")
		bl.Transport.NotifyParty = "Sampaguita Textile Mills Inc, 9 Other Road, Quezon City"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{bl}}, th)
		wantFinding(t, r, "P5-ADDRESSES", SeverityDiscrepancy, "notify party", "art. 14(j)", "field 50")
	})

	t.Run("the credit's own address as notify party raises nothing", func(t *testing.T) {
		credit := goodsCredit()
		credit.Applicant.Address = "118 Chino Roces Avenue, Makati City 1203"
		bl := billOfLading("2026-03-10", "")
		bl.Transport.NotifyParty = "Sampaguita Textile Mills Inc, 118 Chino Roces Avenue, Makati City 1203"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{bl}}, th)
		wantNoFinding(t, r, "P5-ADDRESSES", SeverityDiscrepancy)
	})

	t.Run("no address in the credit for a consignee box that names the applicant", func(t *testing.T) {
		credit := goodsCredit()
		bl := billOfLading("2026-03-10", "")
		bl.Transport.Consignee = "Sampaguita Textile Mills Inc"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{bl}}, th)
		wantFinding(t, r, "P5-ADDRESSES", SeverityVerify, "consignee", "art. 14(j)")
	})

	t.Run("a consignee to the order of a bank is not the applicant", func(t *testing.T) {
		credit := goodsCredit()
		credit.Applicant.Address = "118 Chino Roces Avenue, Makati City 1203"
		bl := billOfLading("2026-03-10", "")
		bl.Transport.Consignee = "To the order of Banco de Manila"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{bl}}, th)
		wantSkipped(t, r, "P5-ADDRESSES")
	})
}

func TestRuleShipperFollowsArticle14k(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("a shipper who is not the beneficiary is expressly allowed", func(t *testing.T) {
		credit := goodsCredit()
		bl := billOfLading("2026-03-10", "")
		bl.Transport.Shipper = "Chennai Freight Forwarders Private Limited"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{bl}}, th)
		wantFinding(t, r, "P6-SHIPPER", SeverityCleared, "art. 14(k)", "merchant exporter")
	})

	t.Run("the beneficiary as shipper leaves nothing to clear", func(t *testing.T) {
		credit := goodsCredit()
		bl := billOfLading("2026-03-10", "")
		bl.Transport.Shipper = credit.Beneficiary.Name
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{bl}}, th)
		wantSkipped(t, r, "P6-SHIPPER")
	})

	t.Run("no beneficiary name in the credit", func(t *testing.T) {
		credit := Credit{Currency: "USD", ExpiryDate: MustDate("2026-04-30")}
		bl := billOfLading("2026-03-10", "")
		bl.Transport.Shipper = "Somebody Else"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{bl}}, th)
		wantSkipped(t, r, "P6-SHIPPER")
	})
}

func TestRuleIssuerQualityFollowsArticle3(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("the credit asks for nothing independent", func(t *testing.T) {
		credit := goodsCredit()
		credit.RequiredDocuments = []Requirement{{Kind: KindOrigin, Wording: "certificate of origin", Originals: 1}}
		origin := Document{Kind: KindOrigin, Title: "certificate of origin", Date: MustDate("2026-03-10"),
			Originals: 1, Issuer: credit.Beneficiary.Name, IssuedByBeneficiary: true}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{origin}}, th)
		wantSkipped(t, r, "P7-ISSUER-QUALITY")
	})

	t.Run("an independent issuer satisfies the wording", func(t *testing.T) {
		credit := goodsCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindOrigin, Wording: "certificate of origin issued by an independent chamber of commerce", Originals: 1},
		}
		origin := Document{Kind: KindOrigin, Title: "certificate of origin", Date: MustDate("2026-03-10"),
			Originals: 1, Issuer: "Southern India Chamber of Commerce"}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{origin}}, th)
		wantFinding(t, r, "P7-ISSUER-QUALITY", SeverityCleared, "art. 3", "does not have to prove")
	})

	t.Run("the beneficiary's own certificate does not", func(t *testing.T) {
		credit := goodsCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindInspection, Wording: "inspection certificate issued by a first class surveyor", Originals: 1},
		}
		inspection := Document{Kind: KindInspection, Title: "inspection certificate", Date: MustDate("2026-03-10"),
			Originals: 1, Issuer: credit.Beneficiary.Name, IssuedByBeneficiary: true}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inspection}}, th)
		wantFinding(t, r, "P7-ISSUER-QUALITY", SeverityDiscrepancy, "art. 3", "chamber of commerce")
	})

	t.Run("a near-spelling of the beneficiary's name is still the beneficiary", func(t *testing.T) {
		credit := goodsCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindInspection, Wording: "inspection certificate issued by an independent surveyor", Originals: 1},
		}
		inspection := Document{Kind: KindInspection, Title: "inspection certificate", Date: MustDate("2026-03-10"),
			Originals: 1, Issuer: "Coimbatore Spindel Works Private Limited"}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inspection}}, th)
		wantFinding(t, r, "P7-ISSUER-QUALITY", SeverityDiscrepancy)
	})

	t.Run("the document is not in the presentation at all", func(t *testing.T) {
		credit := goodsCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindInspection, Wording: "inspection certificate issued by a first class surveyor", Originals: 1},
		}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "P7-ISSUER-QUALITY")
	})
}

func TestRuleCrossDocumentFollowsArticle14d(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("nothing repeated across documents", func(t *testing.T) {
		r := Check(goodsCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "P8-CROSS-DOCUMENT")
	})

	t.Run("an invoice number quoted wrongly", func(t *testing.T) {
		credit := goodsCredit()
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.InvoiceNumberShown = "CSW/118"
		pl := Document{Kind: KindPackingList, Title: "packing list", Date: MustDate("2026-03-10"),
			Originals: 1, InvoiceNumberShown: "CSW/119"}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv, pl}}, th)
		wantFinding(t, r, "P8-CROSS-DOCUMENT", SeverityDiscrepancy, "CSW/119", "art. 14(d)")
	})

	t.Run("the credit number quoted wrongly", func(t *testing.T) {
		credit := goodsCredit()
		pl := Document{Kind: KindPackingList, Title: "packing list", Date: MustDate("2026-03-10"),
			Originals: 1, LCReferenceShown: "LC-TEST-9"}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{pl}}, th)
		wantFinding(t, r, "P8-CROSS-DOCUMENT", SeverityDiscrepancy, "LC-TEST-9", "LC-TEST-1")
	})

	t.Run("the credit number quoted rightly", func(t *testing.T) {
		credit := goodsCredit()
		pl := Document{Kind: KindPackingList, Title: "packing list", Date: MustDate("2026-03-10"),
			Originals: 1, LCReferenceShown: "lc test 1"}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{pl}}, th)
		wantFinding(t, r, "P8-CROSS-DOCUMENT", SeverityCleared)
	})

	t.Run("a package count that differs", func(t *testing.T) {
		credit := goodsCredit()
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.Packages = 150
		pl := Document{Kind: KindPackingList, Title: "packing list", Date: MustDate("2026-03-10"),
			Originals: 1, Packages: 149}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv, pl}}, th)
		wantFinding(t, r, "P8-CROSS-DOCUMENT", SeverityDiscrepancy, "package count", "149")
	})

	t.Run("a gross weight that differs", func(t *testing.T) {
		credit := goodsCredit()
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.GrossWeightKg = 9450
		pl := Document{Kind: KindPackingList, Title: "packing list", Date: MustDate("2026-03-10"),
			Originals: 1, GrossWeightKg: 9460}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv, pl}}, th)
		wantFinding(t, r, "P8-CROSS-DOCUMENT", SeverityDiscrepancy, "gross weight")
	})

	t.Run("shipping marks that differ", func(t *testing.T) {
		credit := goodsCredit()
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.ShippingMarks = "STM/MANILA/1-150"
		pl := Document{Kind: KindPackingList, Title: "packing list", Date: MustDate("2026-03-10"),
			Originals: 1, ShippingMarks: "STM/CEBU/1-150"}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv, pl}}, th)
		wantFinding(t, r, "P8-CROSS-DOCUMENT", SeverityDiscrepancy, "Shipping marks")
	})

	t.Run("marks differing only in case and punctuation do not", func(t *testing.T) {
		credit := goodsCredit()
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		inv.ShippingMarks = "STM/MANILA/1-150"
		pl := Document{Kind: KindPackingList, Title: "packing list", Date: MustDate("2026-03-10"),
			Originals: 1, ShippingMarks: "stm manila 1-150"}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv, pl}}, th)
		wantNoFinding(t, r, "P8-CROSS-DOCUMENT", SeverityDiscrepancy)
	})
}
