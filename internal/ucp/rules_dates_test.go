package ucp

import (
	"strings"
	"testing"
)

func TestRuleExpiry(t *testing.T) {
	th := StatutoryThresholds()

	cases := []struct {
		name     string
		expiry   string
		closed   []string
		present  string
		want     Severity
		mentions []string
		skip     bool
	}{
		{name: "no expiry date, nothing to test", present: "2026-03-12", skip: true},
		{name: "no presentation date, nothing to test", expiry: "2026-03-31", skip: true},
		{
			name: "well inside the expiry date", expiry: "2026-03-31", present: "2026-03-12",
			want: SeverityCleared, mentions: []string{"19 days to spare"},
		},
		{
			// Presenting on the last day is within time, but with nothing in
			// hand: the tool says so rather than calling it clear.
			name: "on the expiry date itself is within time but flagged", expiry: "2026-03-31", present: "2026-03-31",
			want: SeverityCaution, mentions: []string{"0 days to spare", "over the counter"},
		},
		{
			name: "two days before expiry is a caution", expiry: "2026-03-31", present: "2026-03-30",
			want: SeverityCaution, mentions: []string{"over the counter"},
		},
		{
			name: "a day after expiry", expiry: "2026-03-31", present: "2026-04-01",
			want: SeverityDiscrepancy, mentions: []string{"1 day", "art. 6(e)", "amended"},
		},
		{
			name: "a week after expiry", expiry: "2026-03-31", present: "2026-04-07",
			want: SeverityDiscrepancy, mentions: []string{"7 days"},
		},
		{
			// 28 March 2026 is a Saturday, so the last day moves to Monday 30.
			name: "expiry on a Saturday, presented on the Monday", expiry: "2026-03-28", present: "2026-03-30",
			want: SeverityCleared, mentions: []string{"art. 29(a)", "covering schedule"},
		},
		{
			name:   "expiry on a listed holiday, presented the next day",
			expiry: "2026-03-26", closed: []string{"2026-03-26"}, present: "2026-03-27",
			want: SeverityCleared, mentions: []string{"art. 29(a)"},
		},
		{
			name: "expiry on a Saturday, presented on the Tuesday after", expiry: "2026-03-28", present: "2026-03-31",
			want: SeverityDiscrepancy, mentions: []string{"30 March 2026"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := minimalCredit()
			if c.expiry != "" {
				credit.ExpiryDate = MustDate(c.expiry)
			}
			for _, d := range c.closed {
				credit.BankClosedDays = append(credit.BankClosedDays, MustDate(d))
			}
			p := Presentation{}
			if c.present != "" {
				p.Date = MustDate(c.present)
			}

			r := Check(credit, p, th)
			if c.skip {
				wantSkipped(t, r, "D1-EXPIRY")
				return
			}
			wantFinding(t, r, "D1-EXPIRY", c.want, c.mentions...)
		})
	}
}

func TestRuleLateShipmentAndArticle29c(t *testing.T) {
	th := StatutoryThresholds()

	cases := []struct {
		name     string
		latest   string
		bl       Document
		want     Severity
		mentions []string
		skip     bool
	}{
		{name: "no latest shipment date", bl: billOfLading("2026-03-08", ""), skip: true},
		{name: "no transport document", latest: "2026-03-10", skip: true},
		{
			name: "shipped well inside", latest: "2026-03-10", bl: billOfLading("2026-03-05", ""),
			want: SeverityCleared, mentions: []string{"5 days before"},
		},
		{
			name:   "shipped on the last day allowed — art. 3 includes the date mentioned",
			latest: "2026-03-10", bl: billOfLading("2026-03-10", ""),
			want: SeverityCleared, mentions: []string{"art. 3", "include the date mentioned"},
		},
		{
			name:   "the on board notation is the date of shipment, not the issue date",
			latest: "2026-03-10", bl: billOfLading("2026-03-08", "2026-03-12"),
			want: SeverityDiscrepancy, mentions: []string{"12 March 2026", "2 days", "art. 20(b)"},
		},
		{
			name:   "shipped a month late",
			latest: "2026-03-10", bl: billOfLading("2026-04-10", ""),
			want: SeverityDiscrepancy, mentions: []string{"31 days", "waiver"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := minimalCredit()
			if c.latest != "" {
				credit.LatestShipmentDate = MustDate(c.latest)
			}
			p := Presentation{Date: MustDate("2026-03-15")}
			if c.bl.Kind != "" {
				p.Documents = []Document{c.bl}
			}

			r := Check(credit, p, th)
			if c.skip {
				wantSkipped(t, r, "D2-LATE-SHIPMENT")
				return
			}
			wantFinding(t, r, "D2-LATE-SHIPMENT", c.want, c.mentions...)
		})
	}

	// Article 29(c): the extension that moves the last day for presentation
	// never moves the latest shipment date. This is the half of article 29 that
	// gets forgotten, and the finding has to say so.
	t.Run("art. 29(c) is stated when the expiry was extended", func(t *testing.T) {
		credit := minimalCredit()
		credit.ExpiryDate = MustDate("2026-03-28") // a Saturday
		credit.LatestShipmentDate = MustDate("2026-03-10")
		r := Check(credit, Presentation{
			Date:      MustDate("2026-03-30"),
			Documents: []Document{billOfLading("2026-03-12", "")},
		}, th)
		f := wantFinding(t, r, "D2-LATE-SHIPMENT", SeverityDiscrepancy, "article 29(c)")
		if !strings.Contains(f.Detail, "not extended") {
			t.Fatalf("the finding must say the shipment date is not extended: %q", f.Detail)
		}
		// And the presentation itself was in time on the extended day.
		wantFinding(t, r, "D1-EXPIRY", SeverityCleared, "art. 29(a)")
	})
}

func TestRuleStaleFollowsArticle14c(t *testing.T) {
	th := StatutoryThresholds()

	seaBL := func(shipped string) Document { return billOfLading(shipped, "") }
	forwarder := func(shipped string) Document {
		return Document{Kind: KindTransport, Title: "forwarder's certificate of receipt",
			Date: MustDate(shipped), Originals: 1, Transport: &Transport{Mode: ModeOther}}
	}

	cases := []struct {
		name     string
		period   int
		doc      Document
		present  string
		want     Severity
		mentions []string
		skip     bool
	}{
		{name: "no presentation date", doc: seaBL("2026-03-10"), skip: true},
		{name: "no transport document", present: "2026-03-20", skip: true},
		{
			name: "presented on the 21st day is in time, with nothing to spare", doc: seaBL("2026-03-10"), present: "2026-03-31",
			want: SeverityCaution, mentions: []string{"21 calendar days", "31 March 2026"},
		},
		{
			name: "presented on the 22nd day is stale", doc: seaBL("2026-03-10"), present: "2026-04-01",
			want: SeverityDiscrepancy, mentions: []string{"22 calendar days", "31 March 2026"},
		},
		{
			name: "two days left is a caution", doc: seaBL("2026-03-10"), present: "2026-03-29",
			want: SeverityCaution, mentions: []string{"calendar days", "weekend"},
		},
		{
			name: "well inside the period is clear", doc: seaBL("2026-03-10"), present: "2026-03-18",
			want: SeverityCleared, mentions: []string{"8 calendar days", "31 March 2026"},
		},
		{
			name:   "a period the credit states displaces the 21 days",
			period: 10, doc: seaBL("2026-03-10"), present: "2026-03-25",
			want: SeverityDiscrepancy, mentions: []string{"10 days", "the period the credit states"},
		},
		{
			name: "a document outside arts. 19 to 25 is not caught by the rule at all",
			doc:  forwarder("2026-03-10"), present: "2026-04-30",
			want: SeverityCleared, mentions: []string{"articles 19 to 25", "art. 14(c)"},
		},
		{
			name:   "but a period the credit states reaches even that document",
			period: 10, doc: forwarder("2026-03-10"), present: "2026-03-25",
			want: SeverityDiscrepancy, mentions: []string{"the period the credit states"},
		},
		{
			name: "21 days from a leap day", doc: seaBL("2024-02-29"), present: "2024-03-21",
			want: SeverityCaution, mentions: []string{"21 March 2024"},
		},
		{
			name: "22 days from a leap day", doc: seaBL("2024-02-29"), present: "2024-03-22",
			want: SeverityDiscrepancy, mentions: []string{"21 March 2024"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := minimalCredit()
			credit.PresentationPeriodDays = c.period
			p := Presentation{}
			if c.present != "" {
				p.Date = MustDate(c.present)
			}
			if c.doc.Kind != "" {
				p.Documents = []Document{c.doc}
			}

			r := Check(credit, p, th)
			if c.skip {
				wantSkipped(t, r, "D3-STALE")
				return
			}
			wantFinding(t, r, "D3-STALE", c.want, c.mentions...)
		})
	}
}

func TestRuleOnOrAbout(t *testing.T) {
	th := StatutoryThresholds()
	cases := []struct {
		name      string
		onOrAbout string
		shipped   string
		want      Severity
		mentions  []string
		skip      bool
	}{
		{name: "the credit does not use the expression", shipped: "2026-03-10", skip: true},
		{name: "no transport document", onOrAbout: "2026-03-10", skip: true},
		{name: "five days early is inside", onOrAbout: "2026-03-10", shipped: "2026-03-05", want: SeverityCleared, mentions: []string{"5 March 2026 to 15 March 2026"}},
		{name: "five days late is inside", onOrAbout: "2026-03-10", shipped: "2026-03-15", want: SeverityCleared},
		{name: "six days early is outside", onOrAbout: "2026-03-10", shipped: "2026-03-04", want: SeverityDiscrepancy, mentions: []string{"art. 3", "amendment"}},
		{name: "six days late is outside", onOrAbout: "2026-03-10", shipped: "2026-03-16", want: SeverityDiscrepancy},
		{name: "across a month boundary", onOrAbout: "2026-03-02", shipped: "2026-02-25", want: SeverityCleared, mentions: []string{"25 February 2026"}},
		{name: "across a leap-year February", onOrAbout: "2024-03-02", shipped: "2024-02-26", want: SeverityCleared, mentions: []string{"26 February 2024"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := minimalCredit()
			if c.onOrAbout != "" {
				credit.ShipmentOnOrAbout = MustDate(c.onOrAbout)
			}
			p := Presentation{Date: MustDate("2026-03-20")}
			if c.shipped != "" {
				p.Documents = []Document{billOfLading(c.shipped, "")}
			}
			r := Check(credit, p, th)
			if c.skip {
				wantSkipped(t, r, "D4-ON-OR-ABOUT")
				return
			}
			wantFinding(t, r, "D4-ON-OR-ABOUT", c.want, c.mentions...)
		})
	}
}

func TestRuleDocumentDatesFollowsArticle14i(t *testing.T) {
	th := StatutoryThresholds()
	credit := minimalCredit()
	credit.IssueDate = MustDate("2026-03-01")

	t.Run("no presentation date", func(t *testing.T) {
		r := Check(credit, Presentation{Documents: []Document{billOfLading("2026-03-05", "")}}, th)
		wantSkipped(t, r, "D5-DOC-DATES")
	})

	t.Run("no dated document", func(t *testing.T) {
		r := Check(credit, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{
			{Kind: KindPackingList, Title: "undated packing list", Originals: 1},
		}}, th)
		wantSkipped(t, r, "D5-DOC-DATES")
	})

	t.Run("a document dated after presentation is a discrepancy", func(t *testing.T) {
		late := Document{Kind: KindOrigin, Title: "certificate of origin", Date: MustDate("2026-03-20"), Originals: 1}
		r := Check(credit, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{late}}, th)
		wantFinding(t, r, "D5-DOC-DATES", SeverityDiscrepancy, "20 March 2026", "art. 14(i)", "Re-date")
	})

	t.Run("a document dated before the credit was issued is expressly allowed", func(t *testing.T) {
		early := Document{Kind: KindOrigin, Title: "certificate of origin", Date: MustDate("2026-02-10"), Originals: 1}
		r := Check(credit, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{early}}, th)
		wantFinding(t, r, "D5-DOC-DATES", SeverityCleared, "prior to the issuance date", "Do not have the document reissued")
		wantNoFinding(t, r, "D5-DOC-DATES", SeverityDiscrepancy)
	})

	t.Run("a document dated on the presentation day is in time", func(t *testing.T) {
		same := Document{Kind: KindOrigin, Title: "certificate of origin", Date: MustDate("2026-03-12"), Originals: 1}
		r := Check(credit, Presentation{Date: MustDate("2026-03-12"), Documents: []Document{same}}, th)
		wantFinding(t, r, "D5-DOC-DATES", SeverityCleared)
		wantNoFinding(t, r, "D5-DOC-DATES", SeverityDiscrepancy)
	})
}

func TestRuleInsuranceDateFollowsArticle28e(t *testing.T) {
	th := StatutoryThresholds()
	credit := minimalCredit()

	insurance := func(dated, effective string) Document {
		d := Document{Kind: KindInsurance, Title: "insurance certificate", Originals: 1,
			Issuer: "Bharat General Insurance", Insurance: &Insurance{Currency: "USD", Amount: 11000,
				IssuedBy: "Bharat General Insurance"}}
		if dated != "" {
			d.Date = MustDate(dated)
		}
		if effective != "" {
			d.Insurance.CoverEffectiveFrom = MustDate(effective)
		}
		return d
	}

	cases := []struct {
		name     string
		docs     []Document
		want     Severity
		mentions []string
		skip     bool
	}{
		{name: "no insurance document", docs: []Document{billOfLading("2026-03-10", "")}, skip: true},
		{name: "no transport document", docs: []Document{insurance("2026-03-10", "")}, skip: true},
		{
			name: "no date on the insurance at all",
			docs: []Document{billOfLading("2026-03-10", ""), insurance("", "")},
			skip: true,
		},
		{
			name: "dated before shipment",
			docs: []Document{billOfLading("2026-03-10", ""), insurance("2026-03-08", "")},
			want: SeverityCleared, mentions: []string{"8 March 2026"},
		},
		{
			name: "dated on the day of shipment",
			docs: []Document{billOfLading("2026-03-10", ""), insurance("2026-03-10", "")},
			want: SeverityCleared,
		},
		{
			name: "dated after shipment",
			docs: []Document{billOfLading("2026-03-10", ""), insurance("2026-03-12", "")},
			want: SeverityDiscrepancy, mentions: []string{"art. 28(e)", "uninsured", "reissue"},
		},
		{
			name: "dated after shipment but cover runs from before it",
			docs: []Document{billOfLading("2026-03-10", ""), insurance("2026-03-12", "2026-03-09")},
			want: SeverityCleared, mentions: []string{"art. 28(e)", "9 March 2026"},
		},
		{
			name: "dated after shipment and cover runs from after it too",
			docs: []Document{billOfLading("2026-03-10", ""), insurance("2026-03-12", "2026-03-11")},
			want: SeverityDiscrepancy,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: c.docs}, th)
			if c.skip {
				wantSkipped(t, r, "D6-INSURANCE-DATE")
				return
			}
			wantFinding(t, r, "D6-INSURANCE-DATE", c.want, c.mentions...)
		})
	}
}

func TestRuleMaturityCountsAsArticle3Does(t *testing.T) {
	th := StatutoryThresholds()

	cases := []struct {
		name     string
		credit   func(Credit) Credit
		docs     []Document
		mentions []string
		skip     bool
	}{
		{
			name:   "a sight credit has no maturity",
			credit: func(c Credit) Credit { c.Availability = AvailabilitySight; return c },
			skip:   true,
		},
		{
			name:   "an acceptance credit with no tenor",
			credit: func(c Credit) Credit { c.Availability = AvailabilityAcceptance; return c },
			skip:   true,
		},
		{
			name: "a tenor with no stated basis",
			credit: func(c Credit) Credit {
				c.Availability = AvailabilityAcceptance
				c.DraftTenorDays = 60
				return c
			},
			skip: true,
		},
		{
			name: "60 days from the date of shipment",
			credit: func(c Credit) Credit {
				c.Availability = AvailabilityAcceptance
				c.DraftTenorDays = 60
				c.DraftTenorBasis = TenorFromShipment
				return c
			},
			docs:     []Document{billOfLading("2026-03-10", "")},
			mentions: []string{"9 May 2026", "art. 3", "exclude the date mentioned"},
		},
		{
			name: "90 days from the invoice date, across a year end",
			credit: func(c Credit) Credit {
				c.Availability = AvailabilityDeferred
				c.DraftTenorDays = 90
				c.DraftTenorBasis = TenorFromInvoice
				return c
			},
			docs: []Document{{Kind: KindInvoice, Title: "invoice", Date: MustDate("2026-11-10"),
				Invoice: &Invoice{Currency: "USD", Amount: 100}}},
			mentions: []string{"8 February 2027"},
		},
		{
			name: "a sight tenor is counted from presentation, with a warning that it is only a guide",
			credit: func(c Credit) Credit {
				c.Availability = AvailabilityAcceptance
				c.DraftTenorDays = 30
				c.DraftTenorBasis = TenorFromSight
				return c
			},
			mentions: []string{"runs from the bank's acceptance"},
		},
		{
			name: "a tenor from shipment with no transport document",
			credit: func(c Credit) Credit {
				c.Availability = AvailabilityAcceptance
				c.DraftTenorDays = 60
				c.DraftTenorBasis = TenorFromShipment
				return c
			},
			skip: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			credit := c.credit(minimalCredit())
			r := Check(credit, Presentation{Date: MustDate("2026-03-20"), Documents: c.docs}, th)
			if c.skip {
				wantSkipped(t, r, "D7-MATURITY")
				return
			}
			wantFinding(t, r, "D7-MATURITY", SeverityVerify, c.mentions...)
		})
	}
}
