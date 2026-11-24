// Package ucp examines a draft set of export documents against the key terms of
// a documentary credit, using the rules of UCP 600 — the Uniform Customs and
// Practice for Documentary Credits, 2007 revision, ICC publication 600.
//
// Everything in this package is a pure function of its inputs. Nothing here
// reads a file, opens a socket or looks at the clock except through the Date
// helpers, so every rule can be tested on its own.
//
// What this package is not: it is not the bank. A bank examines the documents
// themselves, on their face, and decides whether the presentation complies
// (articles 14(a) and 15). This package examines a description of those
// documents that the exporter typed in. It catches the mechanical mismatches
// that cause most refusals; it cannot see a signature, a stamp, or whether a
// document is an original.
package ucp

import "time"

// Every figure UCP 600 fixes is defined here, once, with the sub-article it
// comes from. Nothing else in the package hard-codes a number taken from the
// rules. Each one is also a field of Thresholds, so a credit that varies it —
// several of them are expressly variable — or an amendment to the rules can be
// accommodated without touching the code.
const (
	// ArtPresentationPeriodDays: a presentation including one or more original
	// transport documents subject to articles 19 through 25 must be made no
	// later than 21 calendar days after the date of shipment, and in any event
	// not later than the expiry date of the credit. UCP 600 article 14(c).
	//
	// A credit may state its own period, which displaces this one.
	ArtPresentationPeriodDays = 21

	// ArtExaminationBankingDays: each of the nominated bank, the confirming
	// bank and the issuing bank has a maximum of five banking days following
	// the day of presentation to determine whether a presentation complies
	// (article 14(b)); notice of refusal must be given no later than the close
	// of the fifth banking day following the day of presentation (article
	// 16(d)).
	ArtExaminationBankingDays = 5

	// ArtOnOrAboutDays: "on or about" and similar expressions are read as a
	// stipulation that an event is to occur during a period of five calendar
	// days before until five calendar days after the specified date, both
	// start and end dates included. UCP 600 article 3.
	ArtOnOrAboutDays = 5

	// ArtAboutTolerancePercent: "about" or "approximately" used in connection
	// with the amount of the credit, the quantity, or the unit price allows a
	// tolerance not to exceed 10% more or 10% less than the figure it refers
	// to. UCP 600 article 30(a).
	ArtAboutTolerancePercent = 10.0

	// ArtQuantityTolerancePercent: a tolerance not to exceed 5% more or 5%
	// less than the quantity of the goods is allowed, provided the credit does
	// not state the quantity in terms of a stipulated number of packing units
	// or individual items and the total amount of the drawings does not exceed
	// the amount of the credit. UCP 600 article 30(b).
	ArtQuantityTolerancePercent = 5.0

	// ArtShortDrawTolerancePercent: even when partial shipments are not
	// allowed, a tolerance not to exceed 5% less than the amount of the credit
	// is allowed, provided the quantity of the goods is shipped in full, a
	// unit price stated in the credit is not reduced, and article 30(b) does
	// not apply. UCP 600 article 30(c).
	//
	// This tolerance runs one way only. There is no matching 5% more.
	ArtShortDrawTolerancePercent = 5.0

	// ArtMinInsurancePercent: where the credit gives no indication of the
	// insurance coverage required, the amount of insurance coverage must be at
	// least 110% of the CIF or CIP value of the goods. UCP 600 article
	// 28(f)(ii).
	ArtMinInsurancePercent = 110.0

	// ArtMinOriginalsPerDocument: at least one original of each document
	// stipulated in the credit must be presented. UCP 600 article 17(a).
	ArtMinOriginalsPerDocument = 1

	// ArtFirstHalfLastDay and the month-part days below come from UCP 600
	// article 3: "first half" and "second half" of a month mean the 1st to the
	// 15th and the 16th to the last day of the month; "beginning", "middle"
	// and "end" mean the 1st to the 10th, the 11th to the 20th and the 21st to
	// the last day of the month. All dates inclusive.
	ArtFirstHalfLastDay   = 15
	ArtBeginningLastDay   = 10
	ArtMiddleFirstDay     = 11
	ArtMiddleLastDay      = 20
	ArtSecondHalfFirstDay = 16
	ArtEndOfMonthFirstDay = 21
)

// These two defaults are conventions of this tool, not figures from UCP 600.
// They are named here so that they are as visible as the cited ones and are
// never mistaken for rules.
const (
	// ConventionTypoEditDistance is how many single-character edits this tool
	// will treat as a probable typing error rather than a conflict of data,
	// when comparing a name or a goods description.
	//
	// This is a heuristic, not a rule. It stands in for the ICC's standard
	// banking practice that a misspelling or typing error which does not
	// affect the meaning of a word does not make a document discrepant, while
	// a difference in a number or a model code does. That practice is stated
	// in ISBP 745 (International Standard Banking Practice for the Examination
	// of Documents under UCP 600, ICC publication 745), in its general
	// principles on misspellings and typing errors. No paragraph number is
	// cited here because this build has not verified one, and an invented
	// citation is worse than none. Whatever this tool reports as a probable
	// typing error is for the exporter to correct or to accept at their own
	// risk; it is never reported as compliant.
	ConventionTypoEditDistance = 2

	// ConventionAmountDecimals is how many decimal places amounts are compared
	// to. Two suits the currencies these tools are used with. Credits in a
	// currency with no minor unit — JPY, KRW and VND among them — should set
	// it to 0 so that a rounding difference is not reported.
	ConventionAmountDecimals = 2
)

// Thresholds carries every figure above in a form the user can change, because
// a credit may vary some of them and the rules themselves are revised from time
// to time. A zero field means "use the figure UCP 600 fixes", so an empty
// Thresholds behaves exactly like the constants.
type Thresholds struct {
	PresentationPeriodDays    int     `json:"presentationPeriodDays"`
	ExaminationBankingDays    int     `json:"examinationBankingDays"`
	OnOrAboutDays             int     `json:"onOrAboutDays"`
	AboutTolerancePercent     float64 `json:"aboutTolerancePercent"`
	QuantityTolerancePercent  float64 `json:"quantityTolerancePercent"`
	ShortDrawTolerancePercent float64 `json:"shortDrawTolerancePercent"`
	MinInsurancePercent       float64 `json:"minInsurancePercent"`
	MinOriginalsPerDocument   int     `json:"minOriginalsPerDocument"`

	// TypoEditDistance and AmountDecimals are the two conventions above.
	TypoEditDistance int `json:"typoEditDistance"`
	AmountDecimals   int `json:"amountDecimals"`
	// AmountDecimalsSet says that AmountDecimals is a deliberate choice. It has
	// to exist because zero is both a real answer — a credit in a currency with
	// no minor unit — and what an absent field reads as, and the two must not
	// be confused.
	AmountDecimalsSet bool `json:"amountDecimalsSet"`

	// WeekendDays are the days of the week the bank of presentation is closed,
	// as weekday numbers with Sunday = 0. They are used only to work out the
	// extension in article 29(a) and the examination deadline in article
	// 14(b); UCP 600 fixes no weekend, and in several markets it is not
	// Saturday and Sunday. Nil means Saturday and Sunday.
	WeekendDays []int `json:"weekendDays"`

	// Edited records that the user has changed something, so a report can say
	// whether it was produced on the figures UCP 600 fixes or on altered ones.
	Edited bool `json:"edited"`
}

// StatutoryThresholds returns the figures exactly as UCP 600 fixes them.
func StatutoryThresholds() Thresholds {
	return Thresholds{
		PresentationPeriodDays:    ArtPresentationPeriodDays,
		ExaminationBankingDays:    ArtExaminationBankingDays,
		OnOrAboutDays:             ArtOnOrAboutDays,
		AboutTolerancePercent:     ArtAboutTolerancePercent,
		QuantityTolerancePercent:  ArtQuantityTolerancePercent,
		ShortDrawTolerancePercent: ArtShortDrawTolerancePercent,
		MinInsurancePercent:       ArtMinInsurancePercent,
		MinOriginalsPerDocument:   ArtMinOriginalsPerDocument,
		TypoEditDistance:          ConventionTypoEditDistance,
		AmountDecimals:            ConventionAmountDecimals,
		AmountDecimalsSet:         true,
		WeekendDays:               []int{int(time.Saturday), int(time.Sunday)},
	}
}

// Normalise fills in anything the user left blank with the figure UCP 600 fixes
// and brings anything impossible back into range. It never silently keeps a
// negative percentage or a zero presentation period.
func (t Thresholds) Normalise() Thresholds {
	d := StatutoryThresholds()

	if t.PresentationPeriodDays <= 0 {
		t.PresentationPeriodDays = d.PresentationPeriodDays
	}
	if t.ExaminationBankingDays <= 0 {
		t.ExaminationBankingDays = d.ExaminationBankingDays
	}
	if t.OnOrAboutDays < 0 {
		t.OnOrAboutDays = d.OnOrAboutDays
	} else if t.OnOrAboutDays == 0 {
		t.OnOrAboutDays = d.OnOrAboutDays
	}
	if t.AboutTolerancePercent <= 0 {
		t.AboutTolerancePercent = d.AboutTolerancePercent
	}
	if t.QuantityTolerancePercent < 0 {
		t.QuantityTolerancePercent = d.QuantityTolerancePercent
	} else if t.QuantityTolerancePercent == 0 {
		t.QuantityTolerancePercent = d.QuantityTolerancePercent
	}
	if t.ShortDrawTolerancePercent < 0 {
		t.ShortDrawTolerancePercent = d.ShortDrawTolerancePercent
	} else if t.ShortDrawTolerancePercent == 0 {
		t.ShortDrawTolerancePercent = d.ShortDrawTolerancePercent
	}
	if t.MinInsurancePercent <= 0 {
		t.MinInsurancePercent = d.MinInsurancePercent
	}
	if t.MinOriginalsPerDocument <= 0 {
		t.MinOriginalsPerDocument = d.MinOriginalsPerDocument
	}
	if t.TypoEditDistance < 0 {
		t.TypoEditDistance = d.TypoEditDistance
	} else if t.TypoEditDistance == 0 {
		t.TypoEditDistance = d.TypoEditDistance
	}
	if !t.AmountDecimalsSet || t.AmountDecimals < 0 || t.AmountDecimals > 6 {
		t.AmountDecimals = d.AmountDecimals
	}
	t.AmountDecimalsSet = true
	t.WeekendDays = normaliseWeekend(t.WeekendDays)
	t.Edited = t.differsFromStatute()
	return t
}

// normaliseWeekend keeps the weekday numbers that are real, drops duplicates,
// and falls back to Saturday and Sunday when nothing usable is left. A credit
// presented in a market with no weekend closure can send an explicit empty list
// as [7] — out of range — which this rejects; such a market should instead list
// its holidays on the credit and leave the weekend alone.
func normaliseWeekend(in []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(in))
	for _, d := range in {
		if d < 0 || d > 6 || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	if len(out) == 0 {
		return []int{int(time.Saturday), int(time.Sunday)}
	}
	return out
}

// differsFromStatute reports whether any figure has been moved off the one UCP
// 600 fixes, so that a report can say so plainly.
func (t Thresholds) differsFromStatute() bool {
	d := StatutoryThresholds()
	if t.PresentationPeriodDays != d.PresentationPeriodDays ||
		t.ExaminationBankingDays != d.ExaminationBankingDays ||
		t.OnOrAboutDays != d.OnOrAboutDays ||
		t.AboutTolerancePercent != d.AboutTolerancePercent ||
		t.QuantityTolerancePercent != d.QuantityTolerancePercent ||
		t.ShortDrawTolerancePercent != d.ShortDrawTolerancePercent ||
		t.MinInsurancePercent != d.MinInsurancePercent ||
		t.MinOriginalsPerDocument != d.MinOriginalsPerDocument ||
		t.TypoEditDistance != d.TypoEditDistance ||
		t.AmountDecimals != d.AmountDecimals {
		return true
	}
	if len(t.WeekendDays) != len(d.WeekendDays) {
		return true
	}
	for i := range t.WeekendDays {
		if t.WeekendDays[i] != d.WeekendDays[i] {
			return true
		}
	}
	return false
}

// weekend reports whether a weekday is one the bank is closed on.
func (t Thresholds) weekend(w time.Weekday) bool {
	for _, d := range t.WeekendDays {
		if int(w) == d {
			return true
		}
	}
	return false
}

// Citation names where a figure comes from, for the report and the interface.
type Citation struct {
	Key     string `json:"key"`
	Figure  string `json:"figure"`
	Source  string `json:"source"`
	Varies  bool   `json:"variesByCredit"`
	Comment string `json:"comment"`
}

// Citations lists every figure the tool applies, with its source, in the order
// a reader would want them. The interface and the written report both use this
// rather than restating a figure anywhere else.
func (t Thresholds) Citations() []Citation {
	t = t.Normalise()
	return []Citation{
		{"presentationPeriodDays", plural(t.PresentationPeriodDays, "calendar day", "calendar days"), "UCP 600 art. 14(c)", true,
			"Only where the presentation includes an original transport document subject to arts 19-25, and never beyond the expiry date. A credit may state its own period."},
		{"examinationBankingDays", plural(t.ExaminationBankingDays, "banking day", "banking days"), "UCP 600 arts. 14(b), 16(d)", false,
			"Counted from the day following the day of presentation. Used to say when you should expect an answer."},
		{"onOrAboutDays", plural(t.OnOrAboutDays, "calendar day", "calendar days") + " either side, both days included", "UCP 600 art. 3", false,
			"Applies to \"on or about\" and similar expressions."},
		{"aboutTolerancePercent", percentText(t.AboutTolerancePercent) + " more or less", "UCP 600 art. 30(a)", false,
			"Where the credit says \"about\" or \"approximately\" of the amount, the quantity or the unit price."},
		{"quantityTolerancePercent", percentText(t.QuantityTolerancePercent) + " more or less", "UCP 600 art. 30(b)", true,
			"Quantity only, and not where the credit states the quantity as a number of packing units or individual items, nor where the drawings would exceed the credit amount."},
		{"shortDrawTolerancePercent", percentText(t.ShortDrawTolerancePercent) + " less only", "UCP 600 art. 30(c)", true,
			"A short drawing where partial shipments are not allowed, on condition that the full quantity is shipped and the unit price is not reduced. There is no matching tolerance upward."},
		{"minInsurancePercent", percentText(t.MinInsurancePercent) + " of the CIF or CIP value", "UCP 600 art. 28(f)(ii)", true,
			"The minimum where the credit gives no figure. A percentage stated in the credit is a minimum, not a target."},
		{"minOriginalsPerDocument", plural(t.MinOriginalsPerDocument, "original", "originals"), "UCP 600 art. 17(a)", false,
			"At least this many of each document the credit stipulates."},
		{"typoEditDistance", plural(t.TypoEditDistance, "character edit", "character edits"), "Not UCP 600 — a convention of this tool", true,
			"Stands in for the ICC practice that a misspelling which does not affect meaning is not a discrepancy. A difference in a digit or a model code is never treated as a typing error."},
		{"amountDecimals", plural(t.AmountDecimals, "decimal place", "decimal places"), "Not UCP 600 — a rounding convention", true,
			"Set it to 0 for a credit in a currency with no minor unit, such as JPY, KRW or VND."},
		{"weekendDays", weekdayNames(t.WeekendDays), "Not UCP 600 — a local convention", true,
			"UCP 600 fixes no weekend. Used only for the extension in art. 29(a) and the examination deadline in art. 14(b)."},
	}
}

func weekdayNames(days []int) string {
	out := ""
	for i, d := range days {
		if i > 0 {
			out += ", "
		}
		out += time.Weekday(d).String()
	}
	if out == "" {
		return "none"
	}
	return out
}
