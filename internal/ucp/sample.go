package ucp

// Sample returns a worked example: a credit and a draft presentation carrying
// several of the discrepancies that are actually cited in practice, so that
// somebody opening the tool for the first time can see what it does before
// typing anything of their own. The figures are invented and the parties do not
// exist.
//
// What is wrong with it, deliberately: the bill of lading is dated after the
// latest shipment date; the invoice describes the goods with a different model
// number; the invoice is made out to a buying agent rather than to the
// applicant; the sum insured is short of 110%; and only two of three originals
// of the bill of lading are in the set. What is deliberately right but looks
// wrong: the packing list describes the goods in general terms; the bill of
// lading shows a transhipment on a containerised shipment under a credit that
// prohibits transhipment; the credit's condition about the vessel's age names no
// document and will be disregarded; and the presentation is made after the
// stated expiry date but inside the extension article 29(a) gives, because the
// bank is closed on the day the credit expires.
func Sample() (Credit, Presentation) {
	credit := Credit{
		Reference:     "ILC/2026/004471",
		IssuingBank:   "Banco de Manila, Makati",
		AvailableWith: "Any bank in India by negotiation",
		Availability:  AvailabilityNegotiation,

		Currency:        "USD",
		Amount:          48600,
		AmountQualifier: QualifierExact,

		Applicant: Party{
			Name:    "Sampaguita Textile Mills Inc",
			Address: "118 Chino Roces Avenue, Makati City 1203",
			Country: "Philippines",
		},
		Beneficiary: Party{
			Name:    "Coimbatore Spindle Works Private Limited",
			Address: "SF 44/2 Trichy Road, Coimbatore 641018",
			Country: "India",
		},

		IssueDate: MustDate("2026-01-12"),
		// The expiry falls on a day the bank is closed, so article 29(a) moves
		// the last day for presentation to the following banking day — and
		// article 29(c) leaves the latest shipment date exactly where it was.
		ExpiryDate:  MustDate("2026-03-26"),
		ExpiryPlace: "India",

		LatestShipmentDate:     MustDate("2026-03-10"),
		PresentationPeriodDays: 0, // silent, so the 21 days of art. 14(c) apply

		GoodsDescription: "Ring frame spindle assemblies model RSA-4200, 1800 sets, CIF Manila",

		Quantity:               1800,
		QuantityUnit:           "sets",
		QuantityInPackingUnits: true,
		UnitPrice:              27,
		TradeTerm:              "CIF MANILA",

		PortOfLoading:   "Tuticorin",
		PortOfDischarge: "Manila",

		PartialShipments: PermissionProhibited,
		Transhipment:     PermissionProhibited,

		InsurancePercent: 0, // silent, so 110% of the CIF value applies
		InsuranceRisks:   "Institute Cargo Clauses (A), all risks, warehouse to warehouse",

		RequiredDocuments: []Requirement{
			{Kind: KindInvoice, Wording: "Signed commercial invoice in triplicate", Originals: 1, Copies: 2},
			{Kind: KindTransport, Wording: "Full set 3/3 original clean on board ocean bills of lading", Originals: 3},
			{Kind: KindInsurance, Wording: "Insurance certificate for 110 pct CIF value, blank endorsed", Originals: 2},
			{Kind: KindPackingList, Wording: "Packing list in duplicate", Originals: 1, Copies: 1},
			{Kind: KindOrigin, Wording: "Certificate of origin issued by an independent chamber of commerce", Originals: 1},
		},

		NonDocumentaryConditions: []string{
			"Goods to be shipped on a vessel not more than 20 years old",
		},

		BankClosedDays: []Date{MustDate("2026-03-26")},
	}

	presentation := Presentation{
		Date:          MustDate("2026-03-27"),
		Place:         "Any bank in India by negotiation",
		DrawnCurrency: "USD",
		DrawnAmount:   48600,
		Documents: []Document{
			{
				Kind:                KindInvoice,
				Title:               "Commercial invoice CSW/EX/2026/118",
				Issuer:              "Coimbatore Spindle Works Pvt Ltd",
				IssuedByBeneficiary: true,
				Date:                MustDate("2026-03-12"),
				Originals:           1,
				Copies:              2,
				Signed:              true,
				GoodsDescription:    "Ring frame spindle assemblies model RSA-4400, 1800 sets, CIF Manila",
				InvoiceNumberShown:  "CSW/EX/2026/118",
				LCReferenceShown:    "ILC/2026/004471",
				Quantity:            1800,
				QuantityUnit:        "sets",
				Packages:            150,
				GrossWeightKg:       9450,
				ShippingMarks:       "STM/MANILA/1-150",
				Invoice: &Invoice{
					Currency:            "USD",
					Amount:              48600,
					MadeOutTo:           Party{Name: "Pacific Sourcing Agency Inc", Address: "7F Ayala Triangle, Makati City", Country: "Philippines"},
					Quantity:            1800,
					Unit:                "sets",
					UnitPrice:           27,
					TradeTerm:           "CIF MANILA",
					Signed:              true,
					FreightAndInsurance: 3200,
				},
			},
			{
				Kind:               KindTransport,
				Title:              "Ocean bill of lading TUT/MNL/30881",
				Issuer:             "Southern Lines (Agents) Pvt Ltd",
				Date:               MustDate("2026-03-11"),
				Originals:          2,
				Copies:             3,
				GoodsDescription:   "Spindle assemblies",
				LCReferenceShown:   "ILC/2026/004471",
				InvoiceNumberShown: "CSW/EX/2026/118",
				Packages:           150,
				GrossWeightKg:      9450,
				ShippingMarks:      "STM/MANILA/1-150",
				Transport: &Transport{
					Mode:              ModeBillOfLading,
					OnBoardDate:       MustDate("2026-03-12"),
					PreprintedOnBoard: false,
					Carrier:           "Southern Lines",
					SignedBy:          "Southern Lines (Agents) Pvt Ltd as agents for the carrier",
					Vessel:            "MV Kaveri Star",
					Voyage:            "118W",
					PortOfLoading:     "Tuticorin",
					PortOfDischarge:   "Manila",
					Shipper:           "Coimbatore Spindle Works Private Limited",
					Consignee:         "To the order of Banco de Manila",
					NotifyParty:       "Sampaguita Textile Mills Inc, 118 Chino Roces Avenue, Makati City 1203",
					OriginalsIssued:   3,
					Transhipment:      TranshipmentWill,
					Containerised:     true,
					FreightPrepaid:    true,
					Packages:          150,
					PackageUnit:       "cartons",
					GrossWeightKg:     9450,
					ShippingMarks:     "STM/MANILA/1-150",
				},
			},
			{
				Kind:             KindInsurance,
				Title:            "Marine insurance certificate MIC/2026/88120",
				Issuer:           "Bharat General Insurance Company Limited",
				Date:             MustDate("2026-03-12"),
				Originals:        2,
				GoodsDescription: "Spindle assemblies, 150 cartons",
				LCReferenceShown: "ILC/2026/004471",
				Insurance: &Insurance{
					Currency:        "USD",
					Amount:          51000,
					IsCoverNote:     false,
					IssuedBy:        "Bharat General Insurance Company Limited",
					OriginalsIssued: 2,
					Risks:           "Institute Cargo Clauses (A), warehouse to warehouse",
					CoverFrom:       "Coimbatore",
					CoverTo:         "Manila",
					Endorsed:        false,
				},
			},
			{
				Kind:               KindPackingList,
				Title:              "Packing list CSW/PL/2026/118",
				Issuer:             "Coimbatore Spindle Works Private Limited",
				Date:               MustDate("2026-03-11"),
				Originals:          1,
				Copies:             1,
				GoodsDescription:   "Spindle assemblies",
				InvoiceNumberShown: "CSW/EX/2026/118",
				Quantity:           1800,
				QuantityUnit:       "sets",
				Packages:           150,
				GrossWeightKg:      9450,
				ShippingMarks:      "STM/MANILA/1-150",
			},
			{
				Kind:             KindOrigin,
				Title:            "Certificate of origin COI/2026/4471",
				Issuer:           "Southern India Chamber of Commerce and Industry",
				Date:             MustDate("2026-03-13"),
				Originals:        1,
				GoodsDescription: "Spindle assemblies of Indian origin",
				LCReferenceShown: "ILC/2026/004471",
				Packages:         150,
			},
		},
	}

	return credit, presentation
}
