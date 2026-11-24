package ucp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalisePermission(t *testing.T) {
	cases := map[string]Permission{
		"":              PermissionUnstated,
		"allowed":       PermissionAllowed,
		"ALLOWED":       PermissionAllowed,
		"permitted":     PermissionAllowed,
		"prohibited":    PermissionProhibited,
		"NOT ALLOWED":   PermissionProhibited,
		"not permitted": PermissionProhibited,
		"forbidden":     PermissionProhibited,
		"maybe":         PermissionUnstated,
	}
	for in, want := range cases {
		if got := NormalisePermission(Permission(in)); got != want {
			t.Errorf("NormalisePermission(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormaliseAvailability(t *testing.T) {
	cases := map[string]Availability{
		"":                    AvailabilityUnstated,
		"by acceptance":       AvailabilityAcceptance,
		"ACCEPTANCE":          AvailabilityAcceptance,
		"by deferred payment": AvailabilityDeferred,
		"by negotiation":      AvailabilityNegotiation,
		"by sight payment":    AvailabilitySight,
		"sight":               AvailabilitySight,
		"something else":      AvailabilityUnstated,
	}
	for in, want := range cases {
		if got := NormaliseAvailability(Availability(in)); got != want {
			t.Errorf("NormaliseAvailability(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormaliseTenorBasis(t *testing.T) {
	cases := map[string]TenorBasis{
		"":                          TenorFromUnstated,
		"shipment":                  TenorFromShipment,
		"60 days after B/L date":    TenorFromShipment,
		"from date of shipment":     TenorFromShipment,
		"after invoice date":        TenorFromInvoice,
		"after sight":               TenorFromSight,
		"from date of presentation": TenorFromPresentation,
		"whenever":                  TenorFromUnstated,
	}
	for in, want := range cases {
		if got := NormaliseTenorBasis(TenorBasis(in)); got != want {
			t.Errorf("NormaliseTenorBasis(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestModeArticleAndTitle(t *testing.T) {
	cases := []struct {
		mode     Mode
		article  string
		governed bool
	}{
		{ModeMultimodal, "art. 19", true},
		{ModeBillOfLading, "art. 20", true},
		{ModeSeaWaybill, "art. 21", true},
		{ModeCharterParty, "art. 22", true},
		{ModeAir, "art. 23", true},
		{ModeRoadRail, "art. 24", true},
		{ModeCourier, "art. 25", true},
		{ModeOther, "", false},
		{ModeUnstated, "", false},
	}
	for _, c := range cases {
		article, governed := c.mode.Article()
		if article != c.article || governed != c.governed {
			t.Errorf("%q.Article() = (%q, %v), want (%q, %v)", c.mode, article, governed, c.article, c.governed)
		}
		if c.mode.Title() == "" {
			t.Errorf("%q has no title", c.mode)
		}
	}
	for _, k := range []Kind{KindInvoice, KindTransport, KindInsurance, KindDraft, KindPackingList,
		KindOrigin, KindInspection, KindWeight, KindBeneficiaryCt, KindOther} {
		if k.Title() == "" {
			t.Errorf("%q has no title", k)
		}
	}
}

func TestCreditNormaliseTidiesWithoutChangingMeaning(t *testing.T) {
	in := Credit{
		Reference:        "  ILC/2026/004471  ",
		Currency:         " usd ",
		TradeTerm:        " cif  manila ",
		GoodsDescription: "Spindle    assemblies\nmodel RSA-4200",
		Availability:     "BY NEGOTIATION",
		AmountQualifier:  "APPROX",
		PartialShipments: "NOT ALLOWED",
		Transhipment:     "Prohibited",
		Amount:           -100,
		Quantity:         -5,
		UnitPrice:        -1,
		DraftTenorDays:   -30,
		InsurancePercent: -110,
		Applicant:        Party{Name: "  Sampaguita   Mills  Inc ", Country: " Philippines "},
		RequiredDocuments: []Requirement{
			{Kind: "", Wording: "  Signed commercial  invoice in triplicate "},
			{Kind: KindTransport, Wording: "Full set 3/3 clean on board ocean bills of lading", Originals: -3},
		},
		NonDocumentaryConditions: []string{"  ", "Vessel not over 20 years old"},
		BankClosedDays:           []Date{{2026, 3, 26}, {2023, 2, 29}, {}},
	}
	got := in.Normalise()

	if got.Reference != "ILC/2026/004471" {
		t.Errorf("reference = %q", got.Reference)
	}
	if got.Currency != "USD" {
		t.Errorf("currency = %q, want USD", got.Currency)
	}
	if got.TradeTerm != "CIF MANILA" {
		t.Errorf("trade term = %q", got.TradeTerm)
	}
	if got.GoodsDescription != "Spindle assemblies model RSA-4200" {
		t.Errorf("goods = %q", got.GoodsDescription)
	}
	if got.Availability != AvailabilityNegotiation {
		t.Errorf("availability = %q", got.Availability)
	}
	if got.AmountQualifier != QualifierAbout {
		t.Errorf("amount qualifier = %q", got.AmountQualifier)
	}
	if got.PartialShipments != PermissionProhibited || got.Transhipment != PermissionProhibited {
		t.Errorf("permissions = %q / %q", got.PartialShipments, got.Transhipment)
	}
	for name, v := range map[string]float64{
		"amount": got.Amount, "quantity": got.Quantity, "unit price": got.UnitPrice, "insurance": got.InsurancePercent,
	} {
		if v != 0 {
			t.Errorf("a negative %s should come back as zero, got %v", name, v)
		}
	}
	if got.DraftTenorDays != 0 {
		t.Errorf("a negative tenor should come back as zero, got %d", got.DraftTenorDays)
	}
	if got.Applicant.Name != "Sampaguita Mills Inc" || got.Applicant.Country != "Philippines" {
		t.Errorf("applicant = %+v", got.Applicant)
	}
	if got.RequiredDocuments[0].Kind != KindOther {
		t.Errorf("a requirement with no kind should become \"other\", got %q", got.RequiredDocuments[0].Kind)
	}
	if !got.RequiredDocuments[0].MustBeSigned {
		t.Errorf("\"signed commercial invoice\" should be read as requiring a signature")
	}
	if !got.RequiredDocuments[1].RequireOnBoard || !got.RequiredDocuments[1].RequireFullSet {
		t.Errorf("\"full set ... clean on board\" should be read as requiring both, got %+v", got.RequiredDocuments[1])
	}
	if got.RequiredDocuments[1].Originals != 1 {
		t.Errorf("a negative number of originals should become the floor of one, got %d", got.RequiredDocuments[1].Originals)
	}
	if len(got.NonDocumentaryConditions) != 1 {
		t.Errorf("blank conditions should be dropped, got %v", got.NonDocumentaryConditions)
	}
	if len(got.BankClosedDays) != 1 {
		t.Errorf("unset and impossible closed days should be dropped, got %v", got.BankClosedDays)
	}
}

func TestRequirementIssuerMustBeIndependent(t *testing.T) {
	cases := []struct {
		wording string
		want    bool
	}{
		{"Certificate of origin issued by an independent chamber of commerce", true},
		{"Inspection certificate issued by a first class surveyor", true},
		{"Certificate issued by a well known laboratory", true},
		{"Certificate from an official authority", true},
		{"Certificate from a competent authority", true},
		{"Certificate issued by a qualified engineer", true},
		{"Certificate issued by the local chamber of commerce", true},
		{"Beneficiary's certificate confirming dispatch of documents", false},
		{"Certificate of origin", false},
	}
	for _, c := range cases {
		r := Requirement{Wording: c.wording}.normalise()
		if got := r.IssuerMustBeIndependent(); got != c.want {
			t.Errorf("%q: independent = %v, want %v", c.wording, got, c.want)
		}
	}
}

func TestCreditValidate(t *testing.T) {
	cases := []struct {
		name    string
		credit  Credit
		wantErr string
	}{
		{"an empty credit is valid — every field is optional", Credit{}, ""},
		{"a two-letter currency", Credit{Currency: "US"}, "three-letter"},
		{"an impossible expiry", Credit{ExpiryDate: Date{2023, 2, 29}}, "not a day that exists"},
		{"an impossible shipment date", Credit{LatestShipmentDate: Date{2026, 4, 31}}, "not a day that exists"},
		{"expiry before issue", Credit{IssueDate: Date{2026, 3, 10}, ExpiryDate: Date{2026, 3, 1}}, "before it was issued"},
		{"a tolerance over 100%", Credit{TolerancePlusPercent: 150}, "not a tolerance"},
		{"a quantity tolerance over 100%", Credit{QuantityToleranceMinusPercent: 150}, "not a tolerance"},
		{"an absurd insurance percentage", Credit{InsurancePercent: 5000}, "not a figure"},
		{"a sound credit", Credit{Currency: "USD", IssueDate: Date{2026, 1, 12}, ExpiryDate: Date{2026, 3, 31}}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.credit.Validate()
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want no error", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want an error about %q", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("Validate() = %q, want it to mention %q", err, c.wantErr)
			}
		})
	}
}

func TestPresentationValidate(t *testing.T) {
	cases := []struct {
		name    string
		in      Presentation
		wantErr string
	}{
		{"an empty presentation is valid", Presentation{}, ""},
		{"an impossible presentation date", Presentation{Date: Date{2023, 2, 29}}, "not a day that exists"},
		{"a two-letter currency", Presentation{DrawnCurrency: "US"}, "three-letter"},
		{"an impossible document date", Presentation{Documents: []Document{{Kind: KindInvoice, Date: Date{2026, 4, 31}}}}, "not a day that exists"},
		{"an impossible on board date", Presentation{Documents: []Document{
			{Kind: KindTransport, Transport: &Transport{OnBoardDate: Date{2023, 2, 29}}},
		}}, "on board date"},
		{"an impossible cover date", Presentation{Documents: []Document{
			{Kind: KindInsurance, Insurance: &Insurance{CoverEffectiveFrom: Date{2023, 2, 29}}},
		}}, "cover runs from"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.in.Validate()
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want no error", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("Validate() = %v, want an error about %q", err, c.wantErr)
			}
		})
	}
}

func TestDocumentNormaliseKeepsOnlyItsOwnPart(t *testing.T) {
	d := Document{
		Kind:      KindInvoice,
		Invoice:   &Invoice{Currency: "usd", Amount: 100, Quantity: 10, Unit: " sets ", UnitPrice: 10, Signed: true},
		Transport: &Transport{Mode: ModeBillOfLading},
		Insurance: &Insurance{Amount: 50},
		Draft:     &Draft{Amount: 100},
	}.normalise()

	if d.Transport != nil || d.Insurance != nil || d.Draft != nil {
		t.Fatalf("an invoice must carry only its own part")
	}
	if d.Invoice.Currency != "USD" || d.Invoice.Unit != "sets" {
		t.Fatalf("invoice fields were not tidied: %+v", d.Invoice)
	}
	if d.Quantity != 10 || d.QuantityUnit != "sets" {
		t.Fatalf("the invoice quantity should fill the document's own quantity, got %v %q", d.Quantity, d.QuantityUnit)
	}
	if !d.Signed {
		t.Fatalf("a signed invoice should mark the document signed")
	}

	// A transport document gets a part even when none was sent, so that no
	// rule has to guard against a nil.
	tr := Document{Kind: KindTransport}.normalise()
	if tr.Transport == nil {
		t.Fatalf("a transport document must always carry a transport part")
	}
	// And a kind with no part of its own carries none.
	pl := Document{Kind: KindPackingList, Invoice: &Invoice{Amount: 1}}.normalise()
	if pl.Invoice != nil {
		t.Fatalf("a packing list must not carry an invoice part")
	}
	// An unnamed kind becomes "other".
	if got := (Document{}).normalise().Kind; got != KindOther {
		t.Fatalf("an unnamed kind = %q, want %q", got, KindOther)
	}
	// Negative counts come back to zero.
	neg := Document{Kind: KindOther, Originals: -2, Copies: -1, Quantity: -5, Packages: -3, GrossWeightKg: -9}.normalise()
	if neg.Originals != 0 || neg.Copies != 0 || neg.Quantity != 0 || neg.Packages != 0 || neg.GrossWeightKg != 0 {
		t.Fatalf("negative counts were kept: %+v", neg)
	}
}

func TestDocumentLabel(t *testing.T) {
	cases := []struct {
		name string
		in   Document
		want string
	}{
		{"its own title wins", Document{Kind: KindInvoice, Title: "Invoice CSW/118"}, "Invoice CSW/118"},
		{"a transport document falls back to its mode", Document{Kind: KindTransport, Transport: &Transport{Mode: ModeAir}}, "air transport document"},
		{"anything else falls back to its kind", Document{Kind: KindPackingList}, "packing list"},
	}
	for _, c := range cases {
		if got := c.in.Label(); got != c.want {
			t.Errorf("%s: Label() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestShipmentDateFollowsTheTransportArticles(t *testing.T) {
	cases := []struct {
		name   string
		docs   []Document
		want   Date
		reason string
		found  bool
	}{
		{
			name: "a bill of lading with an on board notation uses the notation — art. 20(b)",
			docs: []Document{{Kind: KindTransport, Date: MustDate("2026-03-08"), Transport: &Transport{
				Mode: ModeBillOfLading, OnBoardDate: MustDate("2026-03-12"),
			}}},
			want: MustDate("2026-03-12"), reason: "on board notation", found: true,
		},
		{
			name: "a bill of lading without one uses the date of issuance — art. 20(b)",
			docs: []Document{{Kind: KindTransport, Date: MustDate("2026-03-08"), Transport: &Transport{
				Mode: ModeBillOfLading, PreprintedOnBoard: true,
			}}},
			want: MustDate("2026-03-08"), reason: "date of issuance", found: true,
		},
		{
			name: "an air waybill uses its date of issuance unless it notes the actual shipment date — art. 23(a)(iii)",
			docs: []Document{{Kind: KindTransport, Date: MustDate("2026-03-08"), Transport: &Transport{Mode: ModeAir}}},
			want: MustDate("2026-03-08"), reason: "art. 23(a)(iii)", found: true,
		},
		{
			name: "an air waybill with a notation of the actual shipment date uses it",
			docs: []Document{{Kind: KindTransport, Date: MustDate("2026-03-08"), Transport: &Transport{
				Mode: ModeAir, OnBoardDate: MustDate("2026-03-10"),
			}}},
			want: MustDate("2026-03-10"), reason: "art. 23(a)(iii)", found: true,
		},
		{
			name: "a sea waybill cites its own sub-article",
			docs: []Document{{Kind: KindTransport, Date: MustDate("2026-03-08"), Transport: &Transport{Mode: ModeSeaWaybill}}},
			want: MustDate("2026-03-08"), reason: "art. 21(b)", found: true,
		},
		{
			name: "a charter party bill cites its own sub-article",
			docs: []Document{{Kind: KindTransport, Date: MustDate("2026-03-08"), Transport: &Transport{Mode: ModeCharterParty}}},
			want: MustDate("2026-03-08"), reason: "art. 22(a)(ii)", found: true,
		},
		{
			name: "two sets take the latest shipment date — art. 31(b)",
			docs: []Document{
				{Kind: KindTransport, Title: "B/L one", Date: MustDate("2026-03-08"), Transport: &Transport{Mode: ModeBillOfLading}},
				{Kind: KindTransport, Title: "B/L two", Date: MustDate("2026-03-11"), Transport: &Transport{Mode: ModeBillOfLading}},
			},
			want: MustDate("2026-03-11"), reason: "art. 31(b)", found: true,
		},
		{
			name: "no transport document, no date of shipment",
			docs: []Document{{Kind: KindInvoice, Date: MustDate("2026-03-08"), Invoice: &Invoice{Amount: 1}}},
			want: Date{}, found: false,
		},
		{
			name: "an undated transport document gives nothing",
			docs: []Document{{Kind: KindTransport, Transport: &Transport{Mode: ModeBillOfLading}}},
			want: Date{}, found: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Presentation{Documents: c.docs}.Normalise()
			got, reason, found := p.ShipmentDate()
			if found != c.found {
				t.Fatalf("found = %v, want %v", found, c.found)
			}
			if got != c.want {
				t.Fatalf("date = %v, want %v", got, c.want)
			}
			if c.found && !strings.Contains(reason, c.reason) {
				t.Fatalf("reason = %q, want it to mention %q", reason, c.reason)
			}
		})
	}
}

func TestHasTransportUnderArticles19To25(t *testing.T) {
	cases := []struct {
		name string
		mode Mode
		want bool
	}{
		{"a bill of lading is governed", ModeBillOfLading, true},
		{"a courier receipt is governed", ModeCourier, true},
		{"a forwarder's receipt is not", ModeOther, false},
		{"an unstated mode is not", ModeUnstated, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Presentation{Documents: []Document{
				{Kind: KindTransport, Date: MustDate("2026-03-10"), Transport: &Transport{Mode: c.mode}},
			}}.Normalise()
			if got := p.HasTransportUnderArticles19To25(); got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
	if (Presentation{}).HasTransportUnderArticles19To25() {
		t.Errorf("an empty presentation has no transport document")
	}
}

func TestPresentationLookups(t *testing.T) {
	p := Presentation{Documents: []Document{
		{Kind: KindInvoice, Title: "first invoice", Invoice: &Invoice{Amount: 1}},
		{Kind: KindInvoice, Title: "second invoice", Invoice: &Invoice{Amount: 2}},
		{Kind: KindTransport, Transport: &Transport{Mode: ModeAir}},
	}}.Normalise()

	first, ok := p.First(KindInvoice)
	if !ok || first.Title != "first invoice" {
		t.Errorf("First(invoice) = %q, %v", first.Title, ok)
	}
	if got := len(p.OfKind(KindInvoice)); got != 2 {
		t.Errorf("OfKind(invoice) returned %d, want 2", got)
	}
	if got := len(p.TransportDocuments()); got != 1 {
		t.Errorf("TransportDocuments() returned %d, want 1", got)
	}
	if _, ok := p.First(KindDraft); ok {
		t.Errorf("First(draft) found one that is not there")
	}
}

func TestCreditJSONRoundTrip(t *testing.T) {
	credit, presentation := Sample()

	raw, err := json.Marshal(struct {
		Credit       Credit       `json:"credit"`
		Presentation Presentation `json:"presentation"`
	}{credit, presentation})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var back struct {
		Credit       Credit       `json:"credit"`
		Presentation Presentation `json:"presentation"`
	}
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if back.Credit.Reference != credit.Reference || back.Credit.ExpiryDate != credit.ExpiryDate {
		t.Errorf("the credit did not survive the round trip")
	}
	if len(back.Presentation.Documents) != len(presentation.Documents) {
		t.Errorf("documents = %d, want %d", len(back.Presentation.Documents), len(presentation.Documents))
	}
	bl, ok := back.Presentation.First(KindTransport)
	if !ok || bl.Transport == nil || bl.Transport.OnBoardDate != MustDate("2026-03-12") {
		t.Errorf("the transport part did not survive the round trip: %+v", bl.Transport)
	}

	// Checking the round-tripped case must give the same answer.
	before := Check(credit, presentation, StatutoryThresholds())
	after := Check(back.Credit, back.Presentation, StatutoryThresholds())
	if before.Summary != after.Summary {
		t.Errorf("the summary changed across the round trip: %+v then %+v", before.Summary, after.Summary)
	}
}

func TestPartyEmpty(t *testing.T) {
	if !(Party{}).Empty() {
		t.Errorf("an unset party is empty")
	}
	if !(Party{Name: "   "}).Empty() {
		t.Errorf("a party of whitespace is empty")
	}
	if (Party{Country: "India"}).Empty() {
		t.Errorf("a party with a country is not empty")
	}
}

func TestCIFDeterminable(t *testing.T) {
	cases := map[string]bool{
		"CIF MANILA": true,
		"cif":        true,
		"CIP Manila": true,
		"FOB":        false,
		"CFR Manila": false,
		"":           false,
		"EXW":        false,
	}
	for term, want := range cases {
		c := Credit{TradeTerm: term}.Normalise()
		if got := c.CIFDeterminable(); got != want {
			t.Errorf("CIFDeterminable(%q) = %v, want %v", term, got, want)
		}
	}
}
