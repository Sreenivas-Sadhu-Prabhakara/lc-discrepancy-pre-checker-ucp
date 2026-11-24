package ucp

import (
	"fmt"
	"strings"
)

// Permission is what the credit says about something it may allow or forbid.
type Permission string

const (
	// PermissionUnstated — the credit says nothing. Where UCP 600 supplies an
	// answer for silence, the rule that needs it applies that answer and says
	// which article it came from.
	PermissionUnstated   Permission = ""
	PermissionAllowed    Permission = "allowed"
	PermissionProhibited Permission = "prohibited"
)

// NormalisePermission reads the wording off a credit. "Not allowed", "not
// permitted" and "prohibited" are the same instruction.
func NormalisePermission(p Permission) Permission {
	switch Normalised(string(p)) {
	case "allowed", "permitted", "allowed partial", "yes":
		return PermissionAllowed
	case "prohibited", "not allowed", "not permitted", "forbidden", "no":
		return PermissionProhibited
	default:
		return PermissionUnstated
	}
}

// Availability is how the credit is available, which article 6(b) requires it
// to state. It decides whether a draft is in the picture at all.
type Availability string

const (
	AvailabilityUnstated    Availability = ""
	AvailabilitySight       Availability = "sight-payment"
	AvailabilityDeferred    Availability = "deferred-payment"
	AvailabilityAcceptance  Availability = "acceptance"
	AvailabilityNegotiation Availability = "negotiation"
)

// NormaliseAvailability reads the wording off field 41 of a credit: "by
// acceptance", "acceptance", "BY DEF PAYMENT" and the rest.
func NormaliseAvailability(a Availability) Availability {
	t := Normalised(string(a))
	switch {
	case t == "":
		return AvailabilityUnstated
	case ContainsAny(t, "acceptance", "accept"):
		return AvailabilityAcceptance
	case ContainsAny(t, "deferred", "def payment", "deferred payment"):
		return AvailabilityDeferred
	case ContainsAny(t, "negotiation", "negotiate"):
		return AvailabilityNegotiation
	case ContainsAny(t, "sight", "payment"):
		return AvailabilitySight
	default:
		return AvailabilityUnstated
	}
}

// TenorBasis is what a credit counts a usance tenor from. Article 3 settles the
// counting once the basis is known: the words "from" and "after", when used to
// determine a maturity date, exclude the date mentioned.
type TenorBasis string

const (
	TenorFromUnstated     TenorBasis = ""
	TenorFromShipment     TenorBasis = "shipment"
	TenorFromInvoice      TenorBasis = "invoice"
	TenorFromSight        TenorBasis = "sight"
	TenorFromPresentation TenorBasis = "presentation"
)

// NormaliseTenorBasis reads the wordings a credit uses for the same thing: "60
// days after B/L date" and "60 days from date of shipment" are one basis.
func NormaliseTenorBasis(b TenorBasis) TenorBasis {
	t := Normalised(string(b))
	switch {
	case t == "":
		return TenorFromUnstated
	case ContainsAny(t, "shipment", "bl", "b l", "bill lading", "transport"):
		return TenorFromShipment
	case ContainsAny(t, "invoice"):
		return TenorFromInvoice
	case ContainsAny(t, "sight", "acceptance"):
		return TenorFromSight
	case ContainsAny(t, "presentation", "presented", "negotiation"):
		return TenorFromPresentation
	default:
		return TenorFromUnstated
	}
}

// Kind is the sort of document, which decides which articles reach it.
type Kind string

const (
	KindInvoice       Kind = "commercial-invoice"
	KindTransport     Kind = "transport"
	KindInsurance     Kind = "insurance"
	KindDraft         Kind = "draft"
	KindPackingList   Kind = "packing-list"
	KindOrigin        Kind = "certificate-of-origin"
	KindInspection    Kind = "inspection-certificate"
	KindWeight        Kind = "weight-certificate"
	KindBeneficiaryCt Kind = "beneficiary-certificate"
	KindOther         Kind = "other"
)

// Title gives a document kind the name a credit would use for it.
func (k Kind) Title() string {
	switch k {
	case KindInvoice:
		return "commercial invoice"
	case KindTransport:
		return "transport document"
	case KindInsurance:
		return "insurance document"
	case KindDraft:
		return "draft (bill of exchange)"
	case KindPackingList:
		return "packing list"
	case KindOrigin:
		return "certificate of origin"
	case KindInspection:
		return "inspection certificate"
	case KindWeight:
		return "weight certificate"
	case KindBeneficiaryCt:
		return "beneficiary's certificate"
	default:
		return "document"
	}
}

// Mode is the kind of transport document, because UCP 600 gives each its own
// article and they do not agree with one another. Which article applies decides
// what the date of shipment is, whether a full set is wanted, and whether a
// prohibition on transhipment has any effect at all.
type Mode string

const (
	ModeUnstated Mode = ""
	// ModeMultimodal — a transport document covering at least two different
	// modes of transport, article 19.
	ModeMultimodal Mode = "multimodal"
	// ModeBillOfLading — a bill of lading, article 20.
	ModeBillOfLading Mode = "bill-of-lading"
	// ModeSeaWaybill — a non-negotiable sea waybill, article 21.
	ModeSeaWaybill Mode = "sea-waybill"
	// ModeCharterParty — a charter party bill of lading, article 22.
	ModeCharterParty Mode = "charter-party-bill-of-lading"
	// ModeAir — an air transport document, article 23.
	ModeAir Mode = "air"
	// ModeRoadRail — a road, rail or inland waterway transport document,
	// article 24.
	ModeRoadRail Mode = "road-rail-inland-waterway"
	// ModeCourier — a courier receipt, post receipt or certificate of posting,
	// article 25.
	ModeCourier Mode = "courier-or-post"
	// ModeOther — a document for which none of articles 19 to 25 applies, such
	// as a forwarder's certificate of receipt. The stale-documents rule in
	// article 14(c) does not reach it.
	ModeOther Mode = "other"
)

// Article returns the article of UCP 600 that governs a mode, and whether that
// article is one of 19 to 25 — which is the test article 14(c) uses for the
// 21-day presentation period.
func (m Mode) Article() (string, bool) {
	switch m {
	case ModeMultimodal:
		return "art. 19", true
	case ModeBillOfLading:
		return "art. 20", true
	case ModeSeaWaybill:
		return "art. 21", true
	case ModeCharterParty:
		return "art. 22", true
	case ModeAir:
		return "art. 23", true
	case ModeRoadRail:
		return "art. 24", true
	case ModeCourier:
		return "art. 25", true
	default:
		return "", false
	}
}

// Title names the mode the way a credit would.
func (m Mode) Title() string {
	switch m {
	case ModeMultimodal:
		return "multimodal transport document"
	case ModeBillOfLading:
		return "bill of lading"
	case ModeSeaWaybill:
		return "non-negotiable sea waybill"
	case ModeCharterParty:
		return "charter party bill of lading"
	case ModeAir:
		return "air transport document"
	case ModeRoadRail:
		return "road, rail or inland waterway transport document"
	case ModeCourier:
		return "courier receipt, post receipt or certificate of posting"
	case ModeOther:
		return "transport document outside articles 19 to 25"
	default:
		return "transport document"
	}
}

// Transhipment is what a transport document says about transhipment.
type Transhipment string

const (
	// TranshipmentNone — the document says nothing about it.
	TranshipmentNone Transhipment = ""
	// TranshipmentMay — the document says transhipment may take place, or the
	// carrier reserves the right to tranship.
	TranshipmentMay Transhipment = "may"
	// TranshipmentWill — the document shows that transhipment will take place,
	// naming the place or the second conveyance.
	TranshipmentWill Transhipment = "will"
)

// Party is a name and an address as a document carries them.
type Party struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Country string `json:"country"`
}

func (p Party) normalise() Party {
	p.Name = strings.Join(strings.Fields(p.Name), " ")
	p.Address = strings.Join(strings.Fields(p.Address), " ")
	p.Country = strings.Join(strings.Fields(p.Country), " ")
	return p
}

// Empty reports whether nothing has been entered for the party.
func (p Party) Empty() bool {
	return strings.TrimSpace(p.Name) == "" && strings.TrimSpace(p.Address) == "" && strings.TrimSpace(p.Country) == ""
}

// Requirement is one line of the credit's list of documents required, as the
// credit words it.
type Requirement struct {
	Kind Kind `json:"kind"`
	// Wording is the credit's own text for the document, which the issuer
	// rules in article 3 and the originals rules in article 17 are read
	// against: "signed commercial invoice in triplicate", "certificate of
	// origin issued by an independent chamber of commerce".
	Wording string `json:"wording"`
	// Originals and Copies are how many of each the credit calls for. Where
	// the credit says "in triplicate" or "in three copies", article 17(e) is
	// satisfied by one original and two copies, so the right entry is
	// Originals 1, Copies 2 — and the tool says so when it sees three
	// originals demanded of a document that cannot be issued in three.
	Originals int `json:"originals"`
	Copies    int `json:"copies"`
	// MustBeSigned records that the credit asks for the document signed. A
	// commercial invoice need not be signed unless the credit says so (article
	// 18(a)).
	MustBeSigned bool `json:"mustBeSigned"`
	// RequireOnBoard records a credit calling for an on board bill of lading,
	// which article 20(a)(ii) then requires to be evidenced.
	RequireOnBoard bool `json:"requireOnBoard"`
	// RequireFullSet records a credit calling for a full set of originals.
	RequireFullSet bool `json:"requireFullSet"`
	// CountsInCopies records a credit that counts the document in copies —
	// "in duplicate", "in two fold", "in triplicate", "in three copies" —
	// rather than demanding originals. Article 17(e) applies only to the
	// first: a credit worded that way is satisfied by one original and the
	// remaining number in copies, while "full set 3/3 original bills of
	// lading" is not. It is read off the credit's own wording, because the
	// two readings are a different rule and not a different number.
	CountsInCopies bool `json:"countsInCopies"`
}

func (r Requirement) normalise() Requirement {
	r.Wording = strings.Join(strings.Fields(r.Wording), " ")
	if r.Kind == "" {
		r.Kind = KindOther
	}
	if r.Originals < 0 {
		r.Originals = 0
	}
	if r.Copies < 0 {
		r.Copies = 0
	}
	if r.Originals == 0 && r.Copies == 0 {
		r.Originals = 1
	}
	if ContainsAny(r.Wording, "signed") {
		r.MustBeSigned = true
	}
	if ContainsAny(r.Wording, "on board", "shipped on board", "clean on board") {
		r.RequireOnBoard = true
	}
	if ContainsAny(r.Wording, "full set") {
		r.RequireFullSet = true
	}
	r.CountsInCopies = ContainsAny(r.Wording, "duplicate", "triplicate", "quadruplicate", "fold", "copies", "copy") &&
		!ContainsAny(r.Wording, "original", "originals", "full set")
	return r
}

// IssuerMustBeIndependent reads the credit's wording for the terms article 3
// deals with: "first class", "well known", "qualified", "independent",
// "official", "competent" and "local" used to describe the issuer of a document
// allow any issuer except the beneficiary to issue it.
func (r Requirement) IssuerMustBeIndependent() bool {
	return ContainsAny(r.Wording, "first class", "well known", "qualified", "independent", "official", "competent", "local")
}

// Credit is the credit's key terms, as advised. Only a handful of fields are
// needed for the tool to say anything useful; every rule that wants a field it
// has not been given reports that it could not run rather than assuming.
type Credit struct {
	Reference     string `json:"reference"`
	IssuingBank   string `json:"issuingBank"`
	AvailableWith string `json:"availableWith"`

	Availability    Availability `json:"availability"`
	DraftTenorDays  int          `json:"draftTenorDays"`
	DraftTenorBasis TenorBasis   `json:"draftTenorBasis"`

	Currency              string    `json:"currency"`
	Amount                float64   `json:"amount"`
	AmountQualifier       Qualifier `json:"amountQualifier"`
	TolerancePlusPercent  float64   `json:"tolerancePlusPercent"`
	ToleranceMinusPercent float64   `json:"toleranceMinusPercent"`

	Applicant   Party `json:"applicant"`
	Beneficiary Party `json:"beneficiary"`

	IssueDate   Date   `json:"issueDate"`
	ExpiryDate  Date   `json:"expiryDate"`
	ExpiryPlace string `json:"expiryPlace"`

	LatestShipmentDate Date `json:"latestShipmentDate"`
	// ShipmentOnOrAbout is set where the credit words the shipment date as "on
	// or about", which article 3 turns into a window of five days either side.
	ShipmentOnOrAbout Date `json:"shipmentOnOrAbout"`
	// PresentationPeriodDays is a period the credit states itself for
	// presentation after the date of shipment. Zero means the credit is silent
	// and the 21 days of article 14(c) apply.
	PresentationPeriodDays int `json:"presentationPeriodDays"`

	GoodsDescription string `json:"goodsDescription"`

	Quantity                      float64   `json:"quantity"`
	QuantityUnit                  string    `json:"quantityUnit"`
	QuantityQualifier             Qualifier `json:"quantityQualifier"`
	QuantityInPackingUnits        bool      `json:"quantityInPackingUnits"`
	QuantityTolerancePlusPercent  float64   `json:"quantityTolerancePlusPercent"`
	QuantityToleranceMinusPercent float64   `json:"quantityToleranceMinusPercent"`
	UnitPrice                     float64   `json:"unitPrice"`
	UnitPriceQualifier            Qualifier `json:"unitPriceQualifier"`

	// TradeTerm is the term of delivery the credit states: FOB, CFR, CIF, CIP
	// and the rest. It decides whether a CIF or CIP value can be worked out
	// from the documents for the insurance test in article 28(f)(ii).
	TradeTerm string `json:"tradeTerm"`

	PlaceOfReceipt   string `json:"placeOfReceipt"`
	PortOfLoading    string `json:"portOfLoading"`
	PortOfDischarge  string `json:"portOfDischarge"`
	FinalDestination string `json:"finalDestination"`

	PartialShipments Permission `json:"partialShipments"`
	Transhipment     Permission `json:"transhipment"`

	// InsurancePercent is the percentage of the value the credit requires to be
	// insured. Zero means the credit is silent and the 110% of article
	// 28(f)(ii) applies.
	InsurancePercent float64 `json:"insurancePercent"`
	InsuranceRisks   string  `json:"insuranceRisks"`

	RequiredDocuments []Requirement `json:"requiredDocuments"`

	// NonDocumentaryConditions are conditions the credit states without naming
	// a document to show they have been met. Article 14(h) deems such a
	// condition not stated and disregards it; the tool lists them so that the
	// exporter knows which of the credit's instructions a bank will ignore.
	NonDocumentaryConditions []string `json:"nonDocumentaryConditions"`

	// BankClosedDays are days the bank to which presentation is to be made is
	// closed, beyond the weekend. They drive the extension in article 29(a) and
	// the examination deadline in article 14(b). UCP 600 cannot know them; the
	// exporter's bank does.
	BankClosedDays []Date `json:"bankClosedDays"`
}

// Normalise tidies what the user typed without changing its meaning: whitespace
// collapsed, currency upper-cased, wordings read into the enumerations, and
// negative figures brought to zero.
func (c Credit) Normalise() Credit {
	c.Reference = strings.Join(strings.Fields(c.Reference), " ")
	c.IssuingBank = strings.Join(strings.Fields(c.IssuingBank), " ")
	c.AvailableWith = strings.Join(strings.Fields(c.AvailableWith), " ")
	c.Currency = strings.ToUpper(strings.Join(strings.Fields(c.Currency), ""))
	c.ExpiryPlace = strings.Join(strings.Fields(c.ExpiryPlace), " ")
	c.GoodsDescription = strings.Join(strings.Fields(c.GoodsDescription), " ")
	c.QuantityUnit = strings.Join(strings.Fields(c.QuantityUnit), " ")
	c.TradeTerm = strings.ToUpper(strings.Join(strings.Fields(c.TradeTerm), " "))
	c.PlaceOfReceipt = strings.Join(strings.Fields(c.PlaceOfReceipt), " ")
	c.PortOfLoading = strings.Join(strings.Fields(c.PortOfLoading), " ")
	c.PortOfDischarge = strings.Join(strings.Fields(c.PortOfDischarge), " ")
	c.FinalDestination = strings.Join(strings.Fields(c.FinalDestination), " ")
	c.InsuranceRisks = strings.Join(strings.Fields(c.InsuranceRisks), " ")

	c.Applicant = c.Applicant.normalise()
	c.Beneficiary = c.Beneficiary.normalise()

	c.Availability = NormaliseAvailability(c.Availability)
	c.DraftTenorBasis = NormaliseTenorBasis(c.DraftTenorBasis)
	c.AmountQualifier = NormaliseQualifier(c.AmountQualifier)
	c.QuantityQualifier = NormaliseQualifier(c.QuantityQualifier)
	c.UnitPriceQualifier = NormaliseQualifier(c.UnitPriceQualifier)
	c.PartialShipments = NormalisePermission(c.PartialShipments)
	c.Transhipment = NormalisePermission(c.Transhipment)

	for _, f := range []*float64{
		&c.Amount, &c.TolerancePlusPercent, &c.ToleranceMinusPercent,
		&c.Quantity, &c.QuantityTolerancePlusPercent, &c.QuantityToleranceMinusPercent,
		&c.UnitPrice, &c.InsurancePercent,
	} {
		if *f < 0 {
			*f = 0
		}
	}
	if c.DraftTenorDays < 0 {
		c.DraftTenorDays = 0
	}
	if c.PresentationPeriodDays < 0 {
		c.PresentationPeriodDays = 0
	}

	docs := make([]Requirement, 0, len(c.RequiredDocuments))
	for _, r := range c.RequiredDocuments {
		docs = append(docs, r.normalise())
	}
	c.RequiredDocuments = docs

	conditions := make([]string, 0, len(c.NonDocumentaryConditions))
	for _, s := range c.NonDocumentaryConditions {
		if s = strings.Join(strings.Fields(s), " "); s != "" {
			conditions = append(conditions, s)
		}
	}
	c.NonDocumentaryConditions = conditions

	closed := make([]Date, 0, len(c.BankClosedDays))
	for _, d := range c.BankClosedDays {
		if !d.Zero() && d.Valid() {
			closed = append(closed, d)
		}
	}
	c.BankClosedDays = closed
	return c
}

// Validate refuses a credit that cannot be examined at all. It is deliberately
// short: almost every field is optional, and a missing field produces an entry
// in the report's list of what could not be checked rather than an error.
func (c Credit) Validate() error {
	c = c.Normalise()
	if c.Currency != "" && len(c.Currency) != 3 {
		return fmt.Errorf("the credit currency %q is not a three-letter code such as USD", c.Currency)
	}
	for _, f := range []struct {
		name string
		date Date
	}{
		{"issue date", c.IssueDate},
		{"expiry date", c.ExpiryDate},
		{"latest shipment date", c.LatestShipmentDate},
		{"\"on or about\" shipment date", c.ShipmentOnOrAbout},
	} {
		if !f.date.Zero() && !f.date.Valid() {
			return fmt.Errorf("the credit's %s, %s, is not a day that exists", f.name, f.date.String())
		}
	}
	if !c.IssueDate.Zero() && !c.ExpiryDate.Zero() && c.ExpiryDate.Before(c.IssueDate) {
		return fmt.Errorf("the credit expires on %s, before it was issued on %s — check both dates",
			c.ExpiryDate.Human(), c.IssueDate.Human())
	}
	if c.TolerancePlusPercent > 100 || c.ToleranceMinusPercent > 100 ||
		c.QuantityTolerancePlusPercent > 100 || c.QuantityToleranceMinusPercent > 100 {
		return fmt.Errorf("a tolerance above 100%% is not a tolerance — check what the credit states")
	}
	if c.InsurancePercent > 1000 {
		return fmt.Errorf("an insurance percentage of %.0f%% is not a figure a credit would state", c.InsurancePercent)
	}
	return nil
}

// CIFDeterminable reports whether a CIF or CIP value can be worked out from the
// documents, which article 28(f)(ii) turns on: where the credit gives no
// insurance figure and the CIF or CIP value cannot be determined, the coverage
// is measured against the greater of the amount demanded and the gross value of
// the goods on the invoice instead.
func (c Credit) CIFDeterminable() bool {
	term := strings.ToUpper(strings.TrimSpace(c.TradeTerm))
	return strings.HasPrefix(term, "CIF") || strings.HasPrefix(term, "CIP")
}

// Invoice is what a commercial invoice carries beyond the common fields.
type Invoice struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	MadeOutTo Party   `json:"madeOutTo"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	UnitPrice float64 `json:"unitPrice"`
	TradeTerm string  `json:"tradeTerm"`
	Signed    bool    `json:"signed"`
	// FreightAndInsurance is what the invoice shows for freight and insurance
	// where the term is CIF or CIP, so that the CIF value the insurance is
	// measured against can be read off the invoice rather than guessed.
	FreightAndInsurance float64 `json:"freightAndInsurance"`
}

// Transport is what a transport document carries.
type Transport struct {
	Mode Mode `json:"mode"`
	// OnBoardDate is the date of an on board notation. Where there is one, it
	// is the date of shipment (article 20(b)); where there is none, the date of
	// issuance of the document is (same sub-article). For an air transport
	// document the equivalent is a specific notation of the actual date of
	// shipment (article 23(a)(iii)).
	OnBoardDate Date `json:"onBoardDate"`
	// PreprintedOnBoard records a document whose printed wording already says
	// the goods are shipped on board, which article 20(a)(ii) accepts without
	// a notation.
	PreprintedOnBoard bool `json:"preprintedOnBoard"`

	Carrier          string `json:"carrier"`
	SignedBy         string `json:"signedBy"`
	Vessel           string `json:"vessel"`
	Voyage           string `json:"voyage"`
	PlaceOfReceipt   string `json:"placeOfReceipt"`
	PortOfLoading    string `json:"portOfLoading"`
	PortOfDischarge  string `json:"portOfDischarge"`
	FinalDestination string `json:"finalDestination"`

	Shipper         string `json:"shipper"`
	Consignee       string `json:"consignee"`
	NotifyParty     string `json:"notifyParty"`
	OriginalsIssued int    `json:"originalsIssued"`

	Transhipment  Transhipment `json:"transhipment"`
	Containerised bool         `json:"containerised"`

	// AdverseClause is the text of any clause or notation on the document that
	// declares a defective condition of the goods or their packaging. Article
	// 27 accepts only a clean transport document, and that is what makes one
	// unclean.
	AdverseClause string `json:"adverseClause"`
	// OnDeck records a document indicating the goods are or will be loaded on
	// deck, which article 26(a) refuses. A clause saying they may be is
	// acceptable, and is recorded in MayBeOnDeck.
	OnDeck      bool `json:"onDeck"`
	MayBeOnDeck bool `json:"mayBeOnDeck"`
	// SubjectToCharterParty records a bill of lading indicating that it is
	// subject to a charter party, which article 20(a)(vi) refuses unless the
	// credit calls for a charter party bill of lading under article 22.
	SubjectToCharterParty bool `json:"subjectToCharterParty"`

	FreightPrepaid bool    `json:"freightPrepaid"`
	Packages       float64 `json:"packages"`
	PackageUnit    string  `json:"packageUnit"`
	GrossWeightKg  float64 `json:"grossWeightKg"`
	ShippingMarks  string  `json:"shippingMarks"`
}

// Insurance is what an insurance document carries.
type Insurance struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
	// CoverEffectiveFrom is a date the document itself says cover runs from,
	// which article 28(e) allows to rescue a document dated after shipment.
	CoverEffectiveFrom Date `json:"coverEffectiveFrom"`
	// IsCoverNote records a broker's cover note, which article 28(c) refuses
	// outright.
	IsCoverNote bool `json:"isCoverNote"`
	// IssuedBy is the insurer, underwriter, agent or proxy that issued it
	// (article 28(a)).
	IssuedBy        string `json:"issuedBy"`
	OriginalsIssued int    `json:"originalsIssued"`
	Risks           string `json:"risks"`
	CoverFrom       string `json:"coverFrom"`
	CoverTo         string `json:"coverTo"`
	Endorsed        bool   `json:"endorsed"`
}

// Draft is what a draft, or bill of exchange, carries.
type Draft struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	TenorDays int     `json:"tenorDays"`
	TenorText string  `json:"tenorText"`
	DrawnOn   string  `json:"drawnOn"`
}

// Document is one document of the presentation as the exporter describes it.
// The three optional parts carry what only that kind of document has.
type Document struct {
	Kind Kind `json:"kind"`
	// Title is the document's own heading. Nothing requires a document to be
	// titled as the credit names it, so a mismatch here is never a
	// discrepancy; it is kept so that the report names the right piece of
	// paper.
	Title string `json:"title"`

	Issuer              string `json:"issuer"`
	IssuedByBeneficiary bool   `json:"issuedByBeneficiary"`
	Date                Date   `json:"date"`
	Originals           int    `json:"originals"`
	Copies              int    `json:"copies"`
	Signed              bool   `json:"signed"`

	// GoodsDescription is the description as it appears on this document. The
	// invoice's must correspond with the credit (article 18(c)); any other
	// document's may be in general terms not conflicting with it (article
	// 14(e)). An empty description on a document other than the invoice is not
	// a discrepancy — article 14(e) applies only "if stated".
	GoodsDescription string `json:"goodsDescription"`

	// InvoiceNumberShown, LCReferenceShown and the quantity fields let the
	// cross-document test in article 14(d) compare the data documents repeat
	// from one another.
	InvoiceNumberShown string  `json:"invoiceNumberShown"`
	LCReferenceShown   string  `json:"lcReferenceShown"`
	Quantity           float64 `json:"quantity"`
	QuantityUnit       string  `json:"quantityUnit"`
	Packages           float64 `json:"packages"`
	GrossWeightKg      float64 `json:"grossWeightKg"`
	ShippingMarks      string  `json:"shippingMarks"`

	Invoice   *Invoice   `json:"invoice,omitempty"`
	Transport *Transport `json:"transport,omitempty"`
	Insurance *Insurance `json:"insurance,omitempty"`
	Draft     *Draft     `json:"draft,omitempty"`
}

// Label names the document for a report line.
func (d Document) Label() string {
	if t := strings.TrimSpace(d.Title); t != "" {
		return t
	}
	if d.Kind == KindTransport && d.Transport != nil {
		return d.Transport.Mode.Title()
	}
	return d.Kind.Title()
}

func (d Document) normalise() Document {
	if d.Kind == "" {
		d.Kind = KindOther
	}
	d.Title = strings.Join(strings.Fields(d.Title), " ")
	d.Issuer = strings.Join(strings.Fields(d.Issuer), " ")
	d.GoodsDescription = strings.Join(strings.Fields(d.GoodsDescription), " ")
	d.InvoiceNumberShown = strings.Join(strings.Fields(d.InvoiceNumberShown), " ")
	d.LCReferenceShown = strings.Join(strings.Fields(d.LCReferenceShown), " ")
	d.QuantityUnit = strings.Join(strings.Fields(d.QuantityUnit), " ")
	d.ShippingMarks = strings.Join(strings.Fields(d.ShippingMarks), " ")

	if d.Originals < 0 {
		d.Originals = 0
	}
	if d.Copies < 0 {
		d.Copies = 0
	}
	if d.Quantity < 0 {
		d.Quantity = 0
	}
	if d.Packages < 0 {
		d.Packages = 0
	}
	if d.GrossWeightKg < 0 {
		d.GrossWeightKg = 0
	}

	// A document kind carries its own part and no other, so that a rule can
	// never read an insurance figure off a packing list.
	switch d.Kind {
	case KindInvoice:
		if d.Invoice == nil {
			d.Invoice = &Invoice{}
		}
		d.Transport, d.Insurance, d.Draft = nil, nil, nil
		d.Invoice.Currency = strings.ToUpper(strings.Join(strings.Fields(d.Invoice.Currency), ""))
		d.Invoice.Unit = strings.Join(strings.Fields(d.Invoice.Unit), " ")
		d.Invoice.TradeTerm = strings.ToUpper(strings.Join(strings.Fields(d.Invoice.TradeTerm), " "))
		d.Invoice.MadeOutTo = d.Invoice.MadeOutTo.normalise()
		for _, f := range []*float64{&d.Invoice.Amount, &d.Invoice.Quantity, &d.Invoice.UnitPrice, &d.Invoice.FreightAndInsurance} {
			if *f < 0 {
				*f = 0
			}
		}
		// The quantity may be entered once, on the document or on its invoice
		// part, and the arithmetic rule needs it on the invoice while the
		// cross-document rule needs it on the document. Fill whichever is
		// missing from whichever is there.
		if d.Quantity == 0 {
			d.Quantity = d.Invoice.Quantity
		}
		if d.Invoice.Quantity == 0 {
			d.Invoice.Quantity = d.Quantity
		}
		if d.QuantityUnit == "" {
			d.QuantityUnit = d.Invoice.Unit
		}
		if d.Invoice.Unit == "" {
			d.Invoice.Unit = d.QuantityUnit
		}
		d.Signed = d.Signed || d.Invoice.Signed
	case KindTransport:
		if d.Transport == nil {
			d.Transport = &Transport{}
		}
		d.Invoice, d.Insurance, d.Draft = nil, nil, nil
		tr := d.Transport
		for _, s := range []*string{&tr.Carrier, &tr.SignedBy, &tr.Vessel, &tr.Voyage,
			&tr.PlaceOfReceipt, &tr.PortOfLoading, &tr.PortOfDischarge, &tr.FinalDestination,
			&tr.Shipper, &tr.Consignee, &tr.NotifyParty, &tr.AdverseClause, &tr.PackageUnit, &tr.ShippingMarks} {
			*s = strings.Join(strings.Fields(*s), " ")
		}
		if tr.OriginalsIssued < 0 {
			tr.OriginalsIssued = 0
		}
		if tr.Packages < 0 {
			tr.Packages = 0
		}
		if tr.GrossWeightKg < 0 {
			tr.GrossWeightKg = 0
		}
		if tr.Mode == "" {
			tr.Mode = ModeUnstated
		}
		if d.Packages == 0 {
			d.Packages = tr.Packages
		}
		if d.GrossWeightKg == 0 {
			d.GrossWeightKg = tr.GrossWeightKg
		}
		if d.ShippingMarks == "" {
			d.ShippingMarks = tr.ShippingMarks
		}
	case KindInsurance:
		if d.Insurance == nil {
			d.Insurance = &Insurance{}
		}
		d.Invoice, d.Transport, d.Draft = nil, nil, nil
		in := d.Insurance
		in.Currency = strings.ToUpper(strings.Join(strings.Fields(in.Currency), ""))
		for _, s := range []*string{&in.IssuedBy, &in.Risks, &in.CoverFrom, &in.CoverTo} {
			*s = strings.Join(strings.Fields(*s), " ")
		}
		if in.Amount < 0 {
			in.Amount = 0
		}
		if in.OriginalsIssued < 0 {
			in.OriginalsIssued = 0
		}
	case KindDraft:
		if d.Draft == nil {
			d.Draft = &Draft{}
		}
		d.Invoice, d.Transport, d.Insurance = nil, nil, nil
		dr := d.Draft
		dr.Currency = strings.ToUpper(strings.Join(strings.Fields(dr.Currency), ""))
		dr.TenorText = strings.Join(strings.Fields(dr.TenorText), " ")
		dr.DrawnOn = strings.Join(strings.Fields(dr.DrawnOn), " ")
		if dr.Amount < 0 {
			dr.Amount = 0
		}
		if dr.TenorDays < 0 {
			dr.TenorDays = 0
		}
	default:
		d.Invoice, d.Transport, d.Insurance, d.Draft = nil, nil, nil, nil
	}
	return d
}

// Presentation is the set of documents the exporter means to present, and when
// and where.
type Presentation struct {
	Date          Date       `json:"date"`
	Place         string     `json:"place"`
	DrawnCurrency string     `json:"drawnCurrency"`
	DrawnAmount   float64    `json:"drawnAmount"`
	Documents     []Document `json:"documents"`
}

// Normalise tidies the presentation and every document in it.
func (p Presentation) Normalise() Presentation {
	p.Place = strings.Join(strings.Fields(p.Place), " ")
	p.DrawnCurrency = strings.ToUpper(strings.Join(strings.Fields(p.DrawnCurrency), ""))
	if p.DrawnAmount < 0 {
		p.DrawnAmount = 0
	}
	docs := make([]Document, 0, len(p.Documents))
	for _, d := range p.Documents {
		docs = append(docs, d.normalise())
	}
	p.Documents = docs
	return p
}

// Validate refuses a presentation that cannot be examined.
func (p Presentation) Validate() error {
	p = p.Normalise()
	if !p.Date.Zero() && !p.Date.Valid() {
		return fmt.Errorf("the presentation date %s is not a day that exists", p.Date.String())
	}
	if p.DrawnCurrency != "" && len(p.DrawnCurrency) != 3 {
		return fmt.Errorf("the currency drawn, %q, is not a three-letter code such as USD", p.DrawnCurrency)
	}
	for i, d := range p.Documents {
		if !d.Date.Zero() && !d.Date.Valid() {
			return fmt.Errorf("document %d (%s) is dated %s, which is not a day that exists", i+1, d.Label(), d.Date.String())
		}
		if d.Kind == KindTransport && d.Transport != nil && !d.Transport.OnBoardDate.Zero() && !d.Transport.OnBoardDate.Valid() {
			return fmt.Errorf("the on board date on %s, %s, is not a day that exists", d.Label(), d.Transport.OnBoardDate.String())
		}
		if d.Kind == KindInsurance && d.Insurance != nil && !d.Insurance.CoverEffectiveFrom.Zero() && !d.Insurance.CoverEffectiveFrom.Valid() {
			return fmt.Errorf("the date cover runs from on %s, %s, is not a day that exists", d.Label(), d.Insurance.CoverEffectiveFrom.String())
		}
	}
	return nil
}

// First returns the first document of a kind, and whether there is one.
func (p Presentation) First(k Kind) (Document, bool) {
	for _, d := range p.Documents {
		if d.Kind == k {
			return d, true
		}
	}
	return Document{}, false
}

// OfKind returns every document of a kind.
func (p Presentation) OfKind(k Kind) []Document {
	var out []Document
	for _, d := range p.Documents {
		if d.Kind == k {
			out = append(out, d)
		}
	}
	return out
}

// TransportDocuments returns the transport documents in the presentation.
func (p Presentation) TransportDocuments() []Document { return p.OfKind(KindTransport) }

// HasTransportUnderArticles19To25 reports whether the presentation includes a
// transport document that one of articles 19 to 25 governs. That, and not the
// mere fact of a shipment, is what brings the 21-day period of article 14(c)
// into play.
func (p Presentation) HasTransportUnderArticles19To25() bool {
	for _, d := range p.TransportDocuments() {
		if d.Transport == nil {
			continue
		}
		if _, governed := d.Transport.Mode.Article(); governed {
			return true
		}
	}
	return false
}

// ShipmentDate works out the date of shipment the rules give the presentation,
// and says which rule gave it.
//
//   - For a bill of lading, a sea waybill or a charter party bill of lading,
//     the date of issuance is the date of shipment unless there is an on board
//     notation, in which case the date in the notation is (articles 20(b),
//     21(b) and 22(a)(ii) respectively).
//   - For an air transport document, the date of issuance is the date of
//     shipment unless there is a specific notation of the actual date of
//     shipment (article 23(a)(iii)). The flight date in the carrier's box is
//     not it.
//   - Where more than one set of transport documents is presented, the latest
//     date of shipment on any of them is the date of shipment (article 31(b)).
func (p Presentation) ShipmentDate() (Date, string, bool) {
	var latest Date
	reason := ""
	sets := 0

	for _, d := range p.TransportDocuments() {
		if d.Transport == nil {
			continue
		}
		var when Date
		var why string
		switch {
		case !d.Transport.OnBoardDate.Zero():
			when = d.Transport.OnBoardDate
			switch d.Transport.Mode {
			case ModeAir:
				why = "the notation of the actual date of shipment on the " + d.Transport.Mode.Title() + " — art. 23(a)(iii)"
			case ModeBillOfLading:
				why = "the on board notation on the bill of lading — art. 20(b)"
			case ModeSeaWaybill:
				why = "the on board notation on the non-negotiable sea waybill — art. 21(b)"
			case ModeCharterParty:
				why = "the on board notation on the charter party bill of lading — art. 22(a)(ii)"
			default:
				why = "the shipment notation on the " + d.Transport.Mode.Title()
			}
		case !d.Date.Zero():
			when = d.Date
			switch d.Transport.Mode {
			case ModeAir:
				why = "the date of issuance of the " + d.Transport.Mode.Title() + ", there being no notation of an actual shipment date — art. 23(a)(iii)"
			case ModeBillOfLading:
				why = "the date of issuance of the bill of lading, there being no on board notation — art. 20(b)"
			case ModeSeaWaybill:
				why = "the date of issuance of the non-negotiable sea waybill, there being no on board notation — art. 21(b)"
			case ModeCharterParty:
				why = "the date of issuance of the charter party bill of lading, there being no on board notation — art. 22(a)(ii)"
			default:
				why = "the date of issuance of the " + d.Transport.Mode.Title()
			}
		default:
			continue
		}
		sets++
		if latest.Zero() || when.After(latest) {
			latest, reason = when, why
		}
	}

	if latest.Zero() {
		return Date{}, "", false
	}
	if sets > 1 {
		reason = "the latest shipment date shown on the " + plural(sets, "set", "sets") + " of transport documents presented — art. 31(b); taken from " + reason
	}
	return latest, reason, true
}
