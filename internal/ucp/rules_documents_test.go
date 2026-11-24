package ucp

import "testing"

// docsCredit is a credit with a list of documents required.
func docsCredit(reqs ...Requirement) Credit {
	c := minimalCredit()
	c.Amount = 10000
	c.ExpiryDate = MustDate("2026-04-30")
	c.RequiredDocuments = reqs
	return c
}

func TestRuleMissingDocuments(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("the credit lists nothing", func(t *testing.T) {
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "R1-MISSING")
	})

	t.Run("everything the credit asks for is there", func(t *testing.T) {
		credit := docsCredit(
			Requirement{Kind: KindInvoice, Wording: "commercial invoice", Originals: 1},
			Requirement{Kind: KindPackingList, Wording: "packing list", Originals: 1},
		)
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		pl := Document{Kind: KindPackingList, Title: "packing list", Date: MustDate("2026-03-10"), Originals: 1}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv, pl}}, th)
		wantFinding(t, r, "R1-MISSING", SeverityCleared, "2 documents are required")
	})

	t.Run("one document is missing", func(t *testing.T) {
		credit := docsCredit(
			Requirement{Kind: KindInvoice, Wording: "commercial invoice", Originals: 1},
			Requirement{Kind: KindOrigin, Wording: "certificate of origin in duplicate", Originals: 1, Copies: 1},
		)
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		f := wantFinding(t, r, "R1-MISSING", SeverityDiscrepancy, "certificate of origin in duplicate", "arts. 14(a), 15")
		if len(f.Documents) != 1 {
			t.Fatalf("the finding should name the missing document, got %v", f.Documents)
		}
	})

	t.Run("several documents are missing", func(t *testing.T) {
		credit := docsCredit(
			Requirement{Kind: KindInvoice, Wording: "commercial invoice", Originals: 1},
			Requirement{Kind: KindOrigin, Wording: "certificate of origin", Originals: 1},
			Requirement{Kind: KindInsurance, Wording: "insurance certificate", Originals: 1},
		)
		r := Check(credit, Presentation{Date: MustDate("2026-03-20")}, th)
		f := wantFinding(t, r, "R1-MISSING", SeverityDiscrepancy, "Documents the credit requires are missing")
		if len(f.Documents) != 3 {
			t.Fatalf("all three missing documents should be named, got %v", f.Documents)
		}
	})

	t.Run("a requirement with no wording falls back to the kind", func(t *testing.T) {
		credit := docsCredit(Requirement{Kind: KindWeight, Originals: 1})
		r := Check(credit, Presentation{Date: MustDate("2026-03-20")}, th)
		wantFinding(t, r, "R1-MISSING", SeverityDiscrepancy, "weight certificate")
	})
}

func TestRuleOriginalsFollowsArticle17(t *testing.T) {
	th := StatutoryThresholds()

	present := func(originals, copies int) Document {
		return Document{Kind: KindOrigin, Title: "certificate of origin", Date: MustDate("2026-03-10"),
			Originals: originals, Copies: copies, Issuer: "Chamber of Commerce"}
	}

	cases := []struct {
		name     string
		req      Requirement
		doc      Document
		want     Severity
		mentions []string
		skip     bool
	}{
		{
			name: "one original asked for and one presented",
			req:  Requirement{Kind: KindOrigin, Wording: "certificate of origin", Originals: 1},
			doc:  present(1, 0), want: SeverityCleared,
		},
		{
			name: "only copies presented",
			req:  Requirement{Kind: KindOrigin, Wording: "certificate of origin", Originals: 1},
			doc:  present(0, 3), want: SeverityDiscrepancy, mentions: []string{"art. 17(a)", "article 17(d)"},
		},
		{
			name: "nothing entered about originals or copies",
			req:  Requirement{Kind: KindOrigin, Wording: "certificate of origin", Originals: 1},
			doc:  present(0, 0), want: SeverityVerify, mentions: []string{"art. 17(a)"},
		},
		{
			name: "\"in triplicate\" satisfied by one original and two copies",
			req:  Requirement{Kind: KindOrigin, Wording: "certificate of origin in triplicate", Originals: 3},
			doc:  present(1, 2), want: SeverityCleared, mentions: []string{"art. 17(e)", "in duplicate"},
		},
		{
			name: "but not by one original and nothing else",
			req:  Requirement{Kind: KindOrigin, Wording: "certificate of origin in triplicate", Originals: 3},
			doc:  present(1, 0), want: SeverityDiscrepancy, mentions: []string{"arts. 17(a), 17(e)"},
		},
		{
			name: "three originals presented where three are asked for",
			req:  Requirement{Kind: KindOrigin, Wording: "three original certificates of origin", Originals: 3},
			doc:  present(3, 0), want: SeverityCleared,
		},
		{
			// Article 17(e) reaches a credit that counts in copies, not one
			// that demands originals. "Full set 3/3 original" means three
			// originals, and photocopies do not make up the number.
			name: "a credit demanding originals gets no relief from copies",
			req:  Requirement{Kind: KindOrigin, Wording: "full set 3/3 original certificates of origin", Originals: 3},
			doc:  present(2, 3), want: SeverityDiscrepancy,
			mentions: []string{"counts in originals rather than in copies", "only originals will"},
		},
		{
			name: "but a credit counting in copies does get it",
			req:  Requirement{Kind: KindOrigin, Wording: "certificates of origin in three copies", Originals: 3},
			doc:  present(1, 2), want: SeverityCleared, mentions: []string{"art. 17(e)"},
		},
		{
			name: "fewer sheets than the credit asks for is a caution",
			req:  Requirement{Kind: KindOrigin, Wording: "certificate of origin, 1 original and 2 copies", Originals: 1, Copies: 2},
			doc:  present(1, 1), want: SeverityCaution, mentions: []string{"article 17(d)", "Photocopy"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := docsCredit(c.req)
			r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{c.doc}}, th)
			if c.skip {
				wantSkipped(t, r, "R2-ORIGINALS")
				return
			}
			wantFinding(t, r, "R2-ORIGINALS", c.want, c.mentions...)
		})
	}

	t.Run("the credit lists nothing", func(t *testing.T) {
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{present(1, 0)}}, th)
		wantSkipped(t, r, "R2-ORIGINALS")
	})

	t.Run("the document the credit lists is not presented", func(t *testing.T) {
		credit := docsCredit(Requirement{Kind: KindOrigin, Wording: "certificate of origin", Originals: 1})
		r := Check(credit, Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "R2-ORIGINALS")
	})
}

func TestRuleExtraDocumentsFollowsArticle14g(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("nothing extra", func(t *testing.T) {
		credit := docsCredit(Requirement{Kind: KindInvoice, Wording: "commercial invoice", Originals: 1})
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantSkipped(t, r, "R3-EXTRA")
	})

	t.Run("one document nobody asked for", func(t *testing.T) {
		credit := docsCredit(Requirement{Kind: KindInvoice, Wording: "commercial invoice", Originals: 1})
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		extra := Document{Kind: KindWeight, Title: "weight certificate", Date: MustDate("2026-03-10"), Originals: 1}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv, extra}}, th)
		wantFinding(t, r, "R3-EXTRA", SeverityCaution, "art. 14(g)", "disregarded", "Take it out")
	})

	t.Run("several documents nobody asked for", func(t *testing.T) {
		credit := docsCredit(Requirement{Kind: KindInvoice, Wording: "commercial invoice", Originals: 1})
		inv := invoiceFor(credit, 10000)
		inv.Date = MustDate("2026-03-10")
		a := Document{Kind: KindWeight, Title: "weight certificate", Date: MustDate("2026-03-10"), Originals: 1}
		b := Document{Kind: KindInspection, Title: "inspection certificate", Date: MustDate("2026-03-10"), Originals: 1}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv, a, b}}, th)
		wantFinding(t, r, "R3-EXTRA", SeverityCaution, "Take them out")
	})

	t.Run("the credit lists nothing, so nothing can be extra", func(t *testing.T) {
		extra := Document{Kind: KindWeight, Title: "weight certificate", Date: MustDate("2026-03-10"), Originals: 1}
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{extra}}, th)
		wantSkipped(t, r, "R3-EXTRA")
	})
}

func TestRuleNonDocumentaryFollowsArticle14h(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("no such condition", func(t *testing.T) {
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "R4-NON-DOCUMENTARY")
	})

	t.Run("one condition with no document behind it", func(t *testing.T) {
		credit := minimalCredit()
		credit.NonDocumentaryConditions = []string{"Goods to be shipped on a vessel not more than 20 years old"}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20")}, th)
		f := wantFinding(t, r, "R4-NON-DOCUMENTARY", SeverityCleared,
			"art. 14(h)", "deem such condition as not stated", "article 4")
		if len(f.Documents) != 1 {
			t.Fatalf("the condition should be quoted back, got %v", f.Documents)
		}
	})

	t.Run("several conditions", func(t *testing.T) {
		credit := minimalCredit()
		credit.NonDocumentaryConditions = []string{"Vessel not over 20 years old", "Goods of prime quality"}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20")}, th)
		wantFinding(t, r, "R4-NON-DOCUMENTARY", SeverityCleared, "Conditions of the credit will be disregarded")
	})
}

func TestRuleInsuranceForm(t *testing.T) {
	th := StatutoryThresholds()

	insurance := func(fn func(*Document)) Document {
		d := Document{Kind: KindInsurance, Title: "insurance certificate", Date: MustDate("2026-03-08"),
			Originals: 1, Issuer: "Bharat General Insurance Company Limited",
			Insurance: &Insurance{Currency: "USD", Amount: 11000,
				IssuedBy: "Bharat General Insurance Company Limited", OriginalsIssued: 1, Endorsed: true}}
		if fn != nil {
			fn(&d)
		}
		return d
	}

	t.Run("no insurance document", func(t *testing.T) {
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "R5-INSURANCE-FORM")
	})

	t.Run("a policy in order", func(t *testing.T) {
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{insurance(nil)}}, th)
		wantFinding(t, r, "R5-INSURANCE-FORM", SeverityCleared, "arts. 28(a), 28(c)")
	})

	t.Run("a broker's cover note is refused outright", func(t *testing.T) {
		doc := insurance(func(d *Document) { d.Insurance.IsCoverNote = true })
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "R5-INSURANCE-FORM", SeverityDiscrepancy, "art. 28(c)", "art. 28(d)")
	})

	t.Run("not every original presented", func(t *testing.T) {
		doc := insurance(func(d *Document) {
			d.Insurance.OriginalsIssued = 2
			d.Originals = 1
		})
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "R5-INSURANCE-FORM", SeverityDiscrepancy, "art. 28(b)", "2 originals")
	})

	t.Run("issued by the beneficiary rather than an insurer", func(t *testing.T) {
		credit := minimalCredit()
		doc := insurance(func(d *Document) { d.Insurance.IssuedBy = credit.Beneficiary.Name })
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "R5-INSURANCE-FORM", SeverityDiscrepancy, "art. 28(a)", "open policy")
	})

	t.Run("nothing entered for the issuer", func(t *testing.T) {
		doc := insurance(func(d *Document) { d.Insurance.IssuedBy = "" })
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		wantFinding(t, r, "R5-INSURANCE-FORM", SeverityVerify, "art. 28(a)")
	})

	t.Run("an \"all risks\" requirement is read generously", func(t *testing.T) {
		credit := minimalCredit()
		credit.InsuranceRisks = "Institute Cargo Clauses (A), all risks, warehouse to warehouse"
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{insurance(nil)}}, th)
		wantFinding(t, r, "R5-INSURANCE-FORM", SeverityCleared, "art. 28(h)", "exclusions printed on the back")
	})

	t.Run("an endorsement the credit's wording asks for", func(t *testing.T) {
		credit := minimalCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindInsurance, Wording: "insurance certificate for 110 pct CIF value blank endorsed", Originals: 1},
		}
		doc := insurance(func(d *Document) { d.Insurance.Endorsed = false })
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{doc}}, th)
		f := wantFinding(t, r, "R5-INSURANCE-FORM", SeverityCaution, "blank endorsed")
		// The honesty that matters: UCP 600 has no article on endorsement, and
		// the finding has to say so rather than invent one.
		if !contains(f.Detail, "no article requiring an endorsement") {
			t.Fatalf("the finding must admit that UCP 600 does not require an endorsement: %q", f.Detail)
		}
	})

	t.Run("an endorsed document raises nothing", func(t *testing.T) {
		credit := minimalCredit()
		credit.RequiredDocuments = []Requirement{
			{Kind: KindInsurance, Wording: "insurance certificate blank endorsed", Originals: 1},
		}
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{insurance(nil)}}, th)
		wantNoFinding(t, r, "R5-INSURANCE-FORM", SeverityCaution)
	})
}

func TestRuleOriginality(t *testing.T) {
	th := StatutoryThresholds()

	t.Run("no documents", func(t *testing.T) {
		r := Check(minimalCredit(), Presentation{Date: MustDate("2026-03-20")}, th)
		wantSkipped(t, r, "R6-ORIGINALITY")
	})

	t.Run("with documents it explains the test the bank applies", func(t *testing.T) {
		credit := minimalCredit()
		inv := invoiceFor(credit, 10000)
		r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv}}, th)
		wantFinding(t, r, "R6-ORIGINALITY", SeverityVerify,
			"arts. 17(b), 17(c)", "apparently original signature", "marked \"copy\"")
	})
}

// contains is a tiny helper so the assertions above read as sentences.
func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
