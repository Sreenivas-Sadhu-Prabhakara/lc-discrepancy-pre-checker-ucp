package ucp

import (
	"strings"
	"testing"
)

func TestCatalogueIsWellFormed(t *testing.T) {
	entries := Catalogue()
	if len(entries) < 25 {
		t.Fatalf("the catalogue has %d checks; the tool is meant to run around 25", len(entries))
	}

	seen := map[string]bool{}
	groups := map[string]int{}
	for _, c := range entries {
		if c.ID == "" || c.Title == "" || c.Citation == "" || c.Tests == "" || c.Group == "" {
			t.Errorf("an incomplete catalogue entry: %+v", c)
		}
		if seen[c.ID] {
			t.Errorf("%s appears twice in the catalogue", c.ID)
		}
		seen[c.ID] = true
		groups[c.Group]++

		// Every check must cite its source, and the citation must be either a
		// UCP 600 article or an honest statement that it is not one.
		if !strings.HasPrefix(c.Citation, "UCP 600 ") {
			t.Errorf("%s cites %q, which is neither a UCP 600 article nor marked as something else", c.ID, c.Citation)
		}
	}

	for _, want := range []string{groupDates, groupAmounts, groupParties, groupDocuments, groupTransport, groupProcess} {
		if groups[want] == 0 {
			t.Errorf("no check belongs to the group %q", want)
		}
	}
}

func TestEveryRuleInTheCatalogueCanFire(t *testing.T) {
	// Every scenario the rule tests exercise, gathered. If a rule in the
	// catalogue never produces a finding across all of them, either the rule is
	// unreachable or it is untested — both are failures.
	fired := map[string]bool{}
	for _, r := range coverageScenarios() {
		for _, f := range r.Findings {
			fired[f.Rule] = true
		}
	}
	for _, c := range Catalogue() {
		if !fired[c.ID] {
			t.Errorf("%s (%s) never produced a finding in any scenario — it is unreachable or untested", c.ID, c.Title)
		}
	}
}

// coverageScenarios runs the sample and a spread of deliberately broken cases,
// so that the coverage test above sees every rule fire.
func coverageScenarios() []Result {
	th := StatutoryThresholds()
	var out []Result

	sampleCredit, samplePresentation := Sample()
	out = append(out, Check(sampleCredit, samplePresentation, th))

	// A clean sea shipment with everything the credit asks for.
	clean := minimalCredit()
	clean.Amount = 10000
	clean.ExpiryDate = MustDate("2026-03-31")
	clean.ExpiryPlace = "India"
	clean.AvailableWith = "Any bank"
	clean.IssueDate = MustDate("2026-01-01")
	clean.LatestShipmentDate = MustDate("2026-03-10")
	clean.ShipmentOnOrAbout = MustDate("2026-03-08")
	clean.GoodsDescription = "Spindle assemblies model RSA-4200"
	clean.Quantity = 100
	clean.QuantityUnit = "sets"
	clean.UnitPrice = 100
	clean.TradeTerm = "CIF MANILA"
	clean.PortOfLoading = "Tuticorin"
	clean.PortOfDischarge = "Manila"
	clean.InsurancePercent = 110
	clean.InsuranceRisks = "Institute Cargo Clauses (A) all risks"
	clean.Availability = AvailabilityAcceptance
	clean.DraftTenorDays = 60
	clean.DraftTenorBasis = TenorFromShipment
	clean.RequiredDocuments = []Requirement{
		{Kind: KindInvoice, Wording: "signed commercial invoice in triplicate", Originals: 1, Copies: 2},
		{Kind: KindTransport, Wording: "full set 3/3 clean on board ocean bills of lading", Originals: 3},
		{Kind: KindInsurance, Wording: "insurance certificate blank endorsed", Originals: 1},
		{Kind: KindDraft, Wording: "draft at 60 days from B/L date", Originals: 1},
		{Kind: KindOrigin, Wording: "certificate of origin issued by an independent chamber of commerce", Originals: 1},
	}
	clean.NonDocumentaryConditions = []string{"Vessel not over 20 years old"}

	inv := invoiceFor(clean, 10000)
	inv.Date = MustDate("2026-03-09")
	inv.Signed = true
	inv.Quantity = 100
	inv.QuantityUnit = "sets"
	inv.Packages = 10
	inv.Invoice.Quantity = 100
	inv.Invoice.Unit = "sets"
	inv.Invoice.UnitPrice = 100
	inv.Invoice.Signed = true

	bl := billOfLading("2026-03-08", "2026-03-09")
	bl.GoodsDescription = "Spindle assemblies"
	bl.Packages = 10
	bl.Transport.PortOfLoading = "Tuticorin"
	bl.Transport.PortOfDischarge = "Manila"
	bl.Transport.Shipper = "Chennai Freight Forwarders Private Limited"
	bl.Transport.Consignee = "To the order of the issuing bank"

	insurance := Document{
		Kind: KindInsurance, Title: "insurance certificate", Date: MustDate("2026-03-09"),
		Issuer: "Bharat General Insurance Company Limited", Originals: 1,
		Insurance: &Insurance{
			Currency: "USD", Amount: 11000, IssuedBy: "Bharat General Insurance Company Limited",
			OriginalsIssued: 1, Endorsed: true,
		},
	}
	origin := Document{
		Kind: KindOrigin, Title: "certificate of origin", Date: MustDate("2026-03-10"),
		Issuer: "Southern India Chamber of Commerce", Originals: 1,
	}
	draft := Document{
		Kind: KindDraft, Title: "draft", Date: MustDate("2026-03-12"), Originals: 1,
		Draft: &Draft{Currency: "USD", Amount: 10000, TenorDays: 60, DrawnOn: "Banco de Manila"},
	}

	out = append(out, Check(clean, Presentation{
		Date: MustDate("2026-03-12"), Place: "Any bank", DrawnCurrency: "USD", DrawnAmount: 10000,
		Documents: []Document{inv, bl, insurance, origin, draft},
	}, th))

	// Late everything, in the wrong currency, short-insured, unclean, on deck,
	// transhipped without a container, under a charter party, at the wrong
	// ports, with a missing document and an extra one.
	bad := clean
	bad.Transhipment = PermissionProhibited
	bad.PartialShipments = PermissionProhibited
	bad.PortOfLoading = "Chennai"
	bad.PortOfDischarge = "Cebu"
	bad.RequiredDocuments = append(bad.RequiredDocuments, Requirement{
		Kind: KindInspection, Wording: "inspection certificate issued by a first class surveyor", Originals: 1,
	})

	badInvoice := inv
	badInvoice.Date = MustDate("2026-04-05")
	badInvoice.GoodsDescription = "Spindle assemblies model RSA-4400"
	badInvoice.Issuer = "Madurai Spindle Traders"
	badInvoice.IssuedByBeneficiary = false
	badInvoice.Signed = false
	badInvoice.Invoice.Signed = false
	badInvoice.Invoice.Currency = "INR"
	badInvoice.Invoice.Amount = 12000
	badInvoice.Invoice.UnitPrice = 90
	badInvoice.Invoice.MadeOutTo = Party{Name: "Pacific Sourcing Agency Inc", Country: "Philippines"}
	badInvoice.Quantity = 80
	badInvoice.Packages = 10
	badInvoice.InvoiceNumberShown = "CSW/118"
	badInvoice.LCReferenceShown = "LC-TEST-9"
	badInvoice.ShippingMarks = "STM/MANILA/1-10"
	badInvoice.Originals = 0
	badInvoice.Copies = 2

	badBL := bl
	badBL.Date = MustDate("2026-03-20")
	badBL.Originals = 2
	badBL.Packages = 9
	badBL.InvoiceNumberShown = "CSW/119"
	badBL.ShippingMarks = "STM/CEBU/1-10"
	badBL.GoodsDescription = "Cotton yarn 40s count"
	badBL.Transport = &Transport{
		Mode: ModeBillOfLading, OriginalsIssued: 3, Vessel: "MV Kaveri Star", Voyage: "118W",
		PortOfLoading: "Tuticorin", PortOfDischarge: "Manila",
		AdverseClause: "Three cartons torn and restitched", OnDeck: true, SubjectToCharterParty: true,
		Transhipment: TranshipmentWill, Containerised: false,
		Consignee: "Sampaguita Textile Mills Inc, 9 Other Road, Quezon City", Packages: 9,
	}

	badInsurance := insurance
	badInsurance.Date = MustDate("2026-03-25")
	badInsurance.Insurance = &Insurance{
		Currency: "INR", Amount: 5000, IsCoverNote: true,
		IssuedBy: "Coimbatore Spindle Works Private Limited", OriginalsIssued: 2, Endorsed: false,
	}
	badInsurance.Originals = 1

	badDraft := draft
	badDraft.Draft = &Draft{Currency: "INR", Amount: 9000, TenorDays: 90}

	extra := Document{Kind: KindWeight, Title: "weight certificate", Date: MustDate("2026-03-09"), Originals: 1}
	inspection := Document{
		Kind: KindInspection, Title: "inspection certificate", Date: MustDate("2026-03-09"),
		Issuer: "Coimbatore Spindle Works Private Limited", IssuedByBeneficiary: true, Originals: 1,
	}

	bad.Applicant.Address = "118 Chino Roces Avenue, Makati City"
	badInvoice.Invoice.MadeOutTo.Address = "7F Ayala Triangle, Makati City"

	out = append(out, Check(bad, Presentation{
		Date: MustDate("2026-04-02"), Place: "State Bank of Nowhere, Erode", DrawnCurrency: "INR", DrawnAmount: 12000,
		Documents: []Document{badInvoice, badBL, badInsurance, badDraft, extra, inspection},
	}, th))

	// An air shipment: the transhipment prohibition does not bite, no full set
	// is wanted, and the stale-documents rule still applies.
	air := clean
	air.Transhipment = PermissionProhibited
	airWaybill := Document{
		Kind: KindTransport, Title: "air waybill", Date: MustDate("2026-03-09"), Originals: 1,
		Transport: &Transport{
			Mode: ModeAir, Transhipment: TranshipmentWill, MayBeOnDeck: false,
			PortOfLoading: "Coimbatore", PortOfDischarge: "Manila",
		},
	}
	out = append(out, Check(air, Presentation{
		Date: MustDate("2026-03-12"), DrawnCurrency: "USD", DrawnAmount: 10000,
		Documents: []Document{inv, airWaybill, insurance, origin, draft},
	}, th))

	// A road consignment note, containerised sea carriage, a multimodal
	// document and two courier receipts, each of which clears a prohibition.
	for _, mode := range []Mode{ModeRoadRail, ModeMultimodal, ModeSeaWaybill, ModeOther} {
		doc := Document{
			Kind: KindTransport, Title: string(mode), Date: MustDate("2026-03-09"), Originals: 1,
			Transport: &Transport{Mode: mode, Transhipment: TranshipmentWill, MayBeOnDeck: true},
		}
		out = append(out, Check(air, Presentation{
			Date: MustDate("2026-03-12"), Documents: []Document{inv, doc},
		}, th))
	}

	containerised := Document{
		Kind: KindTransport, Title: "containerised bill of lading", Date: MustDate("2026-03-09"), Originals: 3,
		Transport: &Transport{Mode: ModeBillOfLading, OriginalsIssued: 3, Transhipment: TranshipmentWill, Containerised: true},
	}
	out = append(out, Check(air, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{inv, containerised}}, th))

	liberty := containerised
	liberty.Transport = &Transport{Mode: ModeBillOfLading, OriginalsIssued: 3, Transhipment: TranshipmentMay, Containerised: false}
	out = append(out, Check(air, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{inv, liberty}}, th))

	// Two courier receipts under a credit forbidding partial shipment.
	couriers := air
	couriers.PartialShipments = PermissionProhibited
	couriers.Quantity = 0
	c1 := Document{Kind: KindTransport, Title: "courier receipt one", Date: MustDate("2026-03-09"), Originals: 1,
		Transport: &Transport{Mode: ModeCourier, FinalDestination: "Manila"}}
	c2 := c1
	c2.Title = "courier receipt two"
	out = append(out, Check(couriers, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{c1, c2}}, th))

	// Two bills of lading on the same vessel and voyage: not a partial
	// shipment at all.
	twoSets := couriers
	b1 := billOfLading("2026-03-08", "2026-03-09")
	b2 := billOfLading("2026-03-09", "2026-03-10")
	b1.Transport.PortOfDischarge, b2.Transport.PortOfDischarge = "Manila", "Manila"
	out = append(out, Check(twoSets, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{b1, b2}}, th))

	// Two bills of lading on different vessels: a partial shipment.
	b3 := billOfLading("2026-03-09", "2026-03-10")
	b3.Transport.Vessel = "MV Other Star"
	b3.Transport.PortOfDischarge = "Cebu"
	out = append(out, Check(twoSets, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{b1, b3}}, th))

	// A presentation after expiry, and one on an extended day.
	late := clean
	late.ExpiryDate = MustDate("2026-03-10")
	out = append(out, Check(late, Presentation{Date: MustDate("2026-03-20"), Documents: []Document{inv, bl}}, th))

	weekend := clean
	weekend.ExpiryDate = MustDate("2026-03-28") // a Saturday
	out = append(out, Check(weekend, Presentation{Date: MustDate("2026-03-30"), Documents: []Document{inv, bl}}, th))

	// A received-for-shipment bill of lading under a credit wanting on board.
	received := clean
	noOnBoard := billOfLading("2026-03-08", "")
	noOnBoard.Transport.PreprintedOnBoard = false
	out = append(out, Check(received, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{inv, noOnBoard}}, th))

	// A presentation with no transport document at all: the 21-day rule does
	// not reach it, and a forwarder's receipt is not one either.
	noTransport := clean
	noTransport.RequiredDocuments = []Requirement{{Kind: KindInvoice, Wording: "commercial invoice", Originals: 1}}
	forwarder := Document{Kind: KindTransport, Title: "forwarder's certificate of receipt",
		Date: MustDate("2026-03-09"), Originals: 1, Transport: &Transport{Mode: ModeOther}}
	out = append(out, Check(noTransport, Presentation{
		Date: MustDate("2026-04-20"), Documents: []Document{inv, forwarder},
	}, th))

	// A short drawing under a credit forbidding partial shipment, within and
	// outside the article 30(c) tolerance.
	short := clean
	short.PartialShipments = PermissionProhibited
	short.Quantity = 100
	shortInv := inv
	shortInv.Invoice.Amount = 9800
	shortInv.Invoice.UnitPrice = 100
	shortInv.Invoice.Quantity = 98
	out = append(out, Check(short, Presentation{
		Date: MustDate("2026-03-12"), DrawnAmount: 9800, DrawnCurrency: "USD",
		Documents: []Document{shortInv, bl},
	}, th))

	// A short drawing a partial-shipment credit simply permits.
	partialOK := clean
	partialOK.PartialShipments = PermissionAllowed
	out = append(out, Check(partialOK, Presentation{
		Date: MustDate("2026-03-12"), DrawnAmount: 4000, DrawnCurrency: "USD",
		Documents: []Document{inv, bl},
	}, th))

	// An insurance certificate dated after shipment but effective earlier.
	effective := clean
	rescued := insurance
	rescued.Date = MustDate("2026-03-20")
	rescued.Insurance.CoverEffectiveFrom = MustDate("2026-03-08")
	out = append(out, Check(effective, Presentation{
		Date: MustDate("2026-03-22"), Documents: []Document{inv, bl, rescued},
	}, th))

	// A credit over-drawn with an "about" amount, and a quantity over the top.
	about := clean
	about.AmountQualifier = QualifierAbout
	about.QuantityInPackingUnits = false
	overInv := inv
	overInv.Invoice.Amount = 20000
	overInv.Quantity = 200
	out = append(out, Check(about, Presentation{
		Date: MustDate("2026-03-12"), DrawnAmount: 20000, DrawnCurrency: "USD",
		Documents: []Document{overInv, bl},
	}, th))

	// A quantity under the floor.
	under := clean
	underInv := inv
	underInv.Quantity = 50
	underInv.Invoice.Quantity = 50
	out = append(out, Check(under, Presentation{
		Date: MustDate("2026-03-12"), DrawnAmount: 10000, DrawnCurrency: "USD",
		Documents: []Document{underInv, bl},
	}, th))

	// A credit wanting a charter party bill where a liner bill is presented,
	// and one where a charter party bill is presented and allowed.
	cp := clean
	cp.RequiredDocuments = []Requirement{{Kind: KindTransport, Wording: "charter party bill of lading", Originals: 1}}
	out = append(out, Check(cp, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{bl}}, th))
	cpDoc := Document{Kind: KindTransport, Title: "charter party bill of lading", Date: MustDate("2026-03-09"),
		Originals: 1, Transport: &Transport{Mode: ModeCharterParty, OriginalsIssued: 1}}
	out = append(out, Check(cp, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{cpDoc}}, th))

	// A draft presented where none is called for, and none where one is.
	noDraft := clean
	noDraft.Availability = AvailabilityAcceptance
	out = append(out, Check(noDraft, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{inv, bl}}, th))
	sightCredit := clean
	sightCredit.Availability = AvailabilitySight
	sightCredit.RequiredDocuments = nil
	out = append(out, Check(sightCredit, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{inv, draft}}, th))

	// A credit requiring insurance with none presented.
	noIns := clean
	out = append(out, Check(noIns, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{inv, bl}}, th))

	// An FOB credit, where no CIF value can be worked out.
	fob := clean
	fob.TradeTerm = "FOB TUTICORIN"
	fob.InsurancePercent = 0
	thinIns := insurance
	thinIns.Insurance.Amount = 10500
	out = append(out, Check(fob, Presentation{
		Date: MustDate("2026-03-12"), DrawnAmount: 10000, DrawnCurrency: "USD",
		Documents: []Document{inv, bl, thinIns},
	}, th))

	// A multimodal document with the credit's places.
	mm := clean
	mm.PlaceOfReceipt = "Coimbatore"
	mm.FinalDestination = "Manila"
	mmDoc := Document{Kind: KindTransport, Title: "multimodal transport document", Date: MustDate("2026-03-09"),
		Originals: 1, Transport: &Transport{Mode: ModeMultimodal, PlaceOfReceipt: "Coimbatore", FinalDestination: "Manila"}}
	out = append(out, Check(mm, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{inv, mmDoc}}, th))

	// A document dated before the credit was issued, and one dated after
	// presentation.
	dates := clean
	dates.IssueDate = MustDate("2026-03-05")
	earlyDoc := origin
	earlyDoc.Date = MustDate("2026-02-20")
	out = append(out, Check(dates, Presentation{
		Date: MustDate("2026-03-12"), Documents: []Document{inv, bl, earlyDoc},
	}, th))

	// An empty case: nothing can be checked, and the tool must say so.
	out = append(out, Check(Credit{}, Presentation{}, th))

	return out
}

func TestEmptyCaseChecksNothingAndSaysSo(t *testing.T) {
	r := Check(Credit{}, Presentation{}, Thresholds{})

	if r.Summary.RulesRun != 0 {
		t.Errorf("%d rules claimed to run on an empty case", r.Summary.RulesRun)
	}
	if r.Summary.RulesSkipped != len(Catalogue()) {
		t.Errorf("%d rules were skipped, want all %d", r.Summary.RulesSkipped, len(Catalogue()))
	}
	if len(r.Findings) != 0 {
		t.Errorf("an empty case produced findings: %v", ruleIDs(r))
	}
	if len(r.NotChecked) != len(Catalogue()) {
		t.Errorf("%d gaps reported, want one for each of the %d checks", len(r.NotChecked), len(Catalogue()))
	}
	for _, g := range r.NotChecked {
		if strings.TrimSpace(g.Need) == "" {
			t.Errorf("%s reported a gap without saying what it needed", g.Rule)
		}
	}
	if !strings.Contains(r.Verdict, "nothing could be checked") {
		t.Errorf("verdict on an empty case = %q", r.Verdict)
	}
	if !r.Clean() {
		t.Errorf("an empty case has nothing wrong with it, so Clean() must be true")
	}
	// The thresholds must still come back filled in, so a report can state
	// them.
	if r.Thresholds.PresentationPeriodDays != ArtPresentationPeriodDays {
		t.Errorf("the thresholds were not normalised: %+v", r.Thresholds)
	}
}

func TestCheckIsDeterministic(t *testing.T) {
	credit, presentation := Sample()
	first := Check(credit, presentation, StatutoryThresholds())
	second := Check(credit, presentation, StatutoryThresholds())

	if first.Summary != second.Summary {
		t.Fatalf("two runs gave different summaries: %+v then %+v", first.Summary, second.Summary)
	}
	if len(first.Findings) != len(second.Findings) {
		t.Fatalf("two runs gave %d and %d findings", len(first.Findings), len(second.Findings))
	}
	for i := range first.Findings {
		if first.Findings[i].Rule != second.Findings[i].Rule || first.Findings[i].Detail != second.Findings[i].Detail {
			t.Fatalf("finding %d differed between runs: %q then %q", i, first.Findings[i].Rule, second.Findings[i].Rule)
		}
	}
}

func TestFindingsAreOrderedWorstFirst(t *testing.T) {
	credit, presentation := Sample()
	r := Check(credit, presentation, StatutoryThresholds())
	if len(r.Findings) < 5 {
		t.Fatalf("the sample should produce a spread of findings, got %d", len(r.Findings))
	}
	last := -1
	for _, f := range r.Findings {
		if f.Severity.Rank() < last {
			t.Fatalf("findings are out of order: %q (%s) came after a worse one", f.Title, f.Severity)
		}
		last = f.Severity.Rank()
	}
	if r.Findings[0].Severity != SeverityDiscrepancy {
		t.Fatalf("the first finding is %q; the sample is built to carry discrepancies", r.Findings[0].Severity)
	}
}

func TestSummaryCountsMatchTheFindings(t *testing.T) {
	for i, r := range coverageScenarios() {
		counted := Summary{}
		for _, f := range r.Findings {
			switch f.Severity {
			case SeverityDiscrepancy:
				counted.Discrepancies++
			case SeverityCaution:
				counted.Cautions++
			case SeverityVerify:
				counted.ToVerify++
			case SeverityCleared:
				counted.Cleared++
			default:
				t.Fatalf("scenario %d produced a finding with an unknown severity %q", i, f.Severity)
			}
		}
		if r.Summary.Discrepancies != counted.Discrepancies || r.Summary.Cautions != counted.Cautions ||
			r.Summary.ToVerify != counted.ToVerify || r.Summary.Cleared != counted.Cleared {
			t.Errorf("scenario %d: summary %+v does not match the findings %+v", i, r.Summary, counted)
		}
		if r.Summary.RulesRun+r.Summary.RulesSkipped != len(Catalogue()) {
			t.Errorf("scenario %d: %d run plus %d skipped is not the %d checks in the catalogue",
				i, r.Summary.RulesRun, r.Summary.RulesSkipped, len(Catalogue()))
		}
	}
}

func TestEveryFindingIsUsable(t *testing.T) {
	for i, r := range coverageScenarios() {
		for _, f := range r.Findings {
			if f.Rule == "" || f.Title == "" || f.Detail == "" || f.Citation == "" {
				t.Errorf("scenario %d: an unusable finding %+v", i, f)
			}
			// A discrepancy without anything to do about it is useless to an
			// exporter, which is the whole point of the tool.
			if f.Severity == SeverityDiscrepancy && strings.TrimSpace(f.Fix) == "" {
				t.Errorf("scenario %d: %s reports a discrepancy with no remedy: %q", i, f.Rule, f.Title)
			}
			if strings.Contains(f.Detail, "%!") {
				t.Errorf("scenario %d: %s has a broken format string: %q", i, f.Rule, f.Detail)
			}
		}
	}
}

func TestSampleCarriesTheDiscrepanciesItAdvertises(t *testing.T) {
	credit, presentation := Sample()
	r := Check(credit, presentation, StatutoryThresholds())

	// The deliberate faults.
	wantFinding(t, r, "D2-LATE-SHIPMENT", SeverityDiscrepancy, "12 March 2026", "10 March 2026")
	wantFinding(t, r, "P1-INVOICE-GOODS", SeverityDiscrepancy)
	wantFinding(t, r, "P3-INVOICE-PARTIES", SeverityDiscrepancy, "Pacific Sourcing Agency")
	wantFinding(t, r, "A6-INSURANCE-AMOUNT", SeverityDiscrepancy, "110%")
	wantFinding(t, r, "T5-FULL-SET", SeverityDiscrepancy, "3 originals")

	// The things that look wrong and are not.
	wantFinding(t, r, "R1-MISSING", SeverityCleared)
	wantFinding(t, r, "P2-OTHER-GOODS", SeverityCleared, "art. 14(e)")
	wantFinding(t, r, "T6-TRANSHIPMENT", SeverityCleared, "container")
	wantFinding(t, r, "R4-NON-DOCUMENTARY", SeverityCleared, "art. 14(h)")
	wantFinding(t, r, "D1-EXPIRY", SeverityCleared, "art. 29(a)")

	if r.Clean() {
		t.Fatalf("the sample is built to be dirty")
	}
	if !strings.Contains(r.Verdict, "discrepanc") {
		t.Fatalf("verdict = %q, want it to count the discrepancies", r.Verdict)
	}
}

func TestVerdictNeverClaimsCompliance(t *testing.T) {
	for i, r := range coverageScenarios() {
		v := strings.ToLower(r.Verdict)
		for _, forbidden := range []string{"complies", "complying", "compliant", "will be accepted", "guaranteed"} {
			if strings.Contains(v, forbidden) {
				t.Errorf("scenario %d: the verdict %q claims compliance, which only the issuing bank determines", i, r.Verdict)
			}
		}
	}

	// A case with nothing found still has to defer to the bank.
	clean := minimalCredit()
	clean.ExpiryDate = MustDate("2026-03-31")
	r := Check(clean, Presentation{Date: MustDate("2026-03-12")}, StatutoryThresholds())
	if !strings.Contains(r.Verdict, "issuing bank still decides") {
		t.Errorf("a clean verdict = %q, want it to defer to the issuing bank", r.Verdict)
	}
}

func TestSeverityLabelsAndRanks(t *testing.T) {
	order := []Severity{SeverityDiscrepancy, SeverityCaution, SeverityVerify, SeverityCleared}
	for i, s := range order {
		if s.Rank() != i {
			t.Errorf("%q ranks %d, want %d", s, s.Rank(), i)
		}
		if s.Label() == "" {
			t.Errorf("%q has no label", s)
		}
	}
	if Severity("invented").Rank() != 3 || Severity("invented").Label() == "" {
		t.Errorf("an unknown severity must still sort and label rather than panicking")
	}
}

func TestLastPresentationDayTakesTheEarlierLimit(t *testing.T) {
	th := StatutoryThresholds()
	inv := invoiceFor(minimalCredit(), 100)
	inv.Date = MustDate("2026-03-09")

	cases := []struct {
		name    string
		credit  Credit
		docs    []Document
		want    Date
		because string
	}{
		{
			name:    "expiry bites first",
			credit:  Credit{ExpiryDate: MustDate("2026-03-20")},
			docs:    []Document{billOfLading("2026-03-10", "2026-03-10")},
			want:    MustDate("2026-03-20"),
			because: "expiry date of the credit",
		},
		{
			name:    "the 21 days bite first",
			credit:  Credit{ExpiryDate: MustDate("2026-04-30")},
			docs:    []Document{billOfLading("2026-03-10", "2026-03-10")},
			want:    MustDate("2026-03-31"),
			because: "21 calendar days after the date of shipment",
		},
		{
			name:    "a period the credit states displaces the 21 days",
			credit:  Credit{ExpiryDate: MustDate("2026-04-30"), PresentationPeriodDays: 10},
			docs:    []Document{billOfLading("2026-03-10", "2026-03-10")},
			want:    MustDate("2026-03-20"),
			because: "the period the credit states",
		},
		{
			name:    "no transport document means only the expiry date",
			credit:  Credit{ExpiryDate: MustDate("2026-04-30")},
			docs:    []Document{inv},
			want:    MustDate("2026-04-30"),
			because: "expiry date of the credit",
		},
		{
			name:    "an expiry on a Saturday moves to the Monday",
			credit:  Credit{ExpiryDate: MustDate("2026-03-28")},
			docs:    []Document{inv},
			want:    MustDate("2026-03-30"),
			because: "art. 29(a)",
		},
		{
			name:    "no expiry date at all leaves the stale limit",
			credit:  Credit{},
			docs:    []Document{billOfLading("2026-03-10", "2026-03-10")},
			want:    MustDate("2026-03-31"),
			because: "21 calendar days",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Check(c.credit, Presentation{Date: MustDate("2026-03-15"), Documents: c.docs}, th)
			if r.LastPresentationDay != c.want {
				t.Fatalf("last day = %v, want %v (%s)", r.LastPresentationDay, c.want, r.LastPresentationWhy)
			}
			if !strings.Contains(r.LastPresentationWhy, c.because) {
				t.Fatalf("reason = %q, want it to mention %q", r.LastPresentationWhy, c.because)
			}
		})
	}

	// With neither limit there is nothing to say.
	r := Check(Credit{}, Presentation{Date: MustDate("2026-03-15")}, th)
	if !r.LastPresentationDay.Zero() || r.LastPresentationWhy != "" {
		t.Errorf("with no expiry and no shipment there is no last day, got %v %q", r.LastPresentationDay, r.LastPresentationWhy)
	}
}

func TestAnswerExpectedByCountsBankingDays(t *testing.T) {
	credit := minimalCredit()
	credit.ExpiryDate = MustDate("2026-04-30")
	credit.BankClosedDays = []Date{MustDate("2026-03-25")}

	// Presented Monday 23 March 2026, with Wednesday 25 closed: the five
	// banking days following are 24, 26, 27, 30 and 31 March.
	r := Check(credit, Presentation{Date: MustDate("2026-03-23")}, StatutoryThresholds())
	if r.AnswerExpectedBy != MustDate("2026-03-31") {
		t.Fatalf("answer expected by %v, want 31 March 2026", r.AnswerExpectedBy)
	}

	// With no presentation date there is nothing to count from.
	r = Check(credit, Presentation{}, StatutoryThresholds())
	if !r.AnswerExpectedBy.Zero() {
		t.Fatalf("answer expected by %v with no presentation date", r.AnswerExpectedBy)
	}
}

func TestReportStatesItsBasisAndDefersToTheBank(t *testing.T) {
	credit, presentation := Sample()
	r := Check(credit, presentation, StatutoryThresholds())
	report := Report(credit, presentation, r)

	for _, want := range []string{
		"PRE-PRESENTATION SELF-CHECK",
		credit.Reference,
		"VERDICT:",
		"Date of shipment taken as 12 March 2026",
		"Last day to present",
		"Answer due from the bank",
		"FIGURES APPLIED",
		"UCP 600 art. 14(c)",
		"UCP 600 art. 28(f)(ii)",
		"Not UCP 600",
		"DISCREPANCY",
		"NOT A DISCREPANCY",
		"NOT CHECKED",
		"is not a bank's examination and it is not advice",
		"arts. 14(a), 15 and 16",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("the report never says %q", want)
		}
	}

	// Every finding has to appear in it, so nothing is found and then hidden.
	for _, f := range r.Findings {
		if !strings.Contains(report, "["+f.Rule+"] "+f.Title) {
			t.Errorf("the report leaves out %s: %q", f.Rule, f.Title)
		}
	}

	// And an empty case must still produce a readable report.
	empty := Check(Credit{}, Presentation{}, Thresholds{})
	if got := Report(Credit{}, Presentation{}, empty); !strings.Contains(got, "nothing could be checked") {
		t.Errorf("the report on an empty case does not say so:\n%s", got)
	}
}

func TestReportRenderingHelpers(t *testing.T) {
	if got := orDash(""); got != "—" {
		t.Errorf("orDash(\"\") = %q", got)
	}
	if got := orDash("  "); got != "—" {
		t.Errorf("orDash(whitespace) = %q", got)
	}
	if got := orDash("India"); got != "India" {
		t.Errorf("orDash(%q) = %q", "India", got)
	}
	if got := pad("abc", 6); got != "abc   " {
		t.Errorf("pad(\"abc\", 6) = %q", got)
	}
	if got := pad("abcdef", 3); got != "abcdef " {
		t.Errorf("pad of an over-long string = %q, want a single trailing space", got)
	}
	if got := isAre(1); got != "is" {
		t.Errorf("isAre(1) = %q", got)
	}
	if got := isAre(2); got != "are" {
		t.Errorf("isAre(2) = %q", got)
	}
	if got := itThem(1); got != "it" {
		t.Errorf("itThem(1) = %q", got)
	}
	if got := itThem(3); got != "them" {
		t.Errorf("itThem(3) = %q", got)
	}
}

func TestEditedThresholdsAreCarriedIntoTheResult(t *testing.T) {
	credit, presentation := Sample()
	own := StatutoryThresholds()
	own.PresentationPeriodDays = 10

	r := Check(credit, presentation, own)
	if !r.Thresholds.Edited {
		t.Fatalf("a changed figure must be reported as edited in the result")
	}
	if r.Thresholds.PresentationPeriodDays != 10 {
		t.Fatalf("the changed figure was not applied: %d", r.Thresholds.PresentationPeriodDays)
	}
	report := Report(credit, presentation, r)
	if !strings.Contains(report, "10 calendar days") {
		t.Fatalf("the report does not state the figure actually applied")
	}
}
