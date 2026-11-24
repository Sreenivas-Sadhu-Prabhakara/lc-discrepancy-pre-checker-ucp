package ucp

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Qualifier is the wording a credit puts in front of a figure, which decides
// what tolerance the figure carries.
type Qualifier string

const (
	// QualifierExact — the figure as stated, with only the tolerances UCP 600
	// gives it by default.
	QualifierExact Qualifier = "exact"
	// QualifierAbout — "about" or "approximately", which allows 10% more or
	// less of the amount, the quantity or the unit price it refers to (UCP 600
	// article 30(a)).
	QualifierAbout Qualifier = "about"
	// QualifierMaximum — "maximum", "not exceeding" or "up to", which allows
	// no excess at all and no tolerance upward.
	QualifierMaximum Qualifier = "maximum"
)

// NormaliseQualifier reads the wording an exporter is likely to copy off the
// credit. Anything unrecognised is treated as the figure exactly as stated,
// which is the conservative reading.
func NormaliseQualifier(q Qualifier) Qualifier {
	switch Qualifier(Normalised(string(q))) {
	case "about", "approximately", "approx", "circa":
		return QualifierAbout
	case "maximum", "max", "not exceeding", "notexceeding", "up to", "upto":
		return QualifierMaximum
	default:
		return QualifierExact
	}
}

// Tolerance is a two-sided allowance around a figure, expressed in percent, and
// the reason it is what it is.
type Tolerance struct {
	PlusPercent  float64 `json:"plusPercent"`
	MinusPercent float64 `json:"minusPercent"`
	Source       string  `json:"source"`
}

// Band returns the lowest and highest figures the tolerance allows around a
// value, rounded to the currency's minor unit so that a comparison is not
// decided by a floating-point remainder.
func (tol Tolerance) Band(value float64, decimals int) (low, high float64) {
	low = RoundTo(value*(1-tol.MinusPercent/100), decimals)
	high = RoundTo(value*(1+tol.PlusPercent/100), decimals)
	return low, high
}

// AmountTolerance works out what the drawn amount is allowed to be, given the
// credit's wording. The order matters:
//
//   - a tolerance the credit states itself displaces the rules' defaults, and
//     article 30(c) says so expressly;
//   - "about" or "approximately" gives 10% either way (article 30(a));
//   - "maximum", "not exceeding" or "up to" gives nothing either way;
//   - otherwise the amount stands as stated, except that where partial
//     shipments are not allowed a drawing of up to 5% less is allowed on the
//     conditions in article 30(c), which the caller tests separately.
//
// There is no general tolerance upward on the credit amount. Article 18(b) lets
// a bank accept an invoice made out for more, but it may not honour more than
// the credit allows, so an invoice over the ceiling is still something the
// exporter needs to know about.
func AmountTolerance(c Credit, t Thresholds) Tolerance {
	t = t.Normalise()
	if c.TolerancePlusPercent > 0 || c.ToleranceMinusPercent > 0 {
		return Tolerance{
			PlusPercent:  c.TolerancePlusPercent,
			MinusPercent: c.ToleranceMinusPercent,
			Source:       "the tolerance stated in the credit",
		}
	}
	switch NormaliseQualifier(c.AmountQualifier) {
	case QualifierAbout:
		return Tolerance{
			PlusPercent:  t.AboutTolerancePercent,
			MinusPercent: t.AboutTolerancePercent,
			Source:       fmt.Sprintf("%s either way because the credit says \"about\" — UCP 600 art. 30(a)", percentText(t.AboutTolerancePercent)),
		}
	case QualifierMaximum:
		return Tolerance{Source: "none — the credit caps the amount (\"maximum\", \"not exceeding\" or \"up to\")"}
	default:
		return Tolerance{Source: "none — the credit states the amount exactly"}
	}
}

// QuantityTolerance works out what the shipped quantity is allowed to be.
// Article 30(b)'s 5% is narrower than it is usually remembered: it does not
// apply where the credit states the quantity as a number of packing units or
// individual items, and it does not apply if the drawings would exceed the
// credit amount. Where the credit says "about", article 30(a)'s 10% applies
// instead.
func QuantityTolerance(c Credit, drawingsExceedCredit bool, t Thresholds) Tolerance {
	t = t.Normalise()
	if c.QuantityTolerancePlusPercent > 0 || c.QuantityToleranceMinusPercent > 0 {
		return Tolerance{
			PlusPercent:  c.QuantityTolerancePlusPercent,
			MinusPercent: c.QuantityToleranceMinusPercent,
			Source:       "the quantity tolerance stated in the credit",
		}
	}
	if NormaliseQualifier(c.QuantityQualifier) == QualifierAbout {
		return Tolerance{
			PlusPercent:  t.AboutTolerancePercent,
			MinusPercent: t.AboutTolerancePercent,
			Source:       fmt.Sprintf("%s either way because the credit says \"about\" the quantity — UCP 600 art. 30(a)", percentText(t.AboutTolerancePercent)),
		}
	}
	if NormaliseQualifier(c.QuantityQualifier) == QualifierMaximum {
		return Tolerance{Source: "none — the credit caps the quantity"}
	}
	if c.QuantityInPackingUnits {
		return Tolerance{Source: "none — the credit states the quantity as a number of packing units or individual items, which art. 30(b) excludes"}
	}
	if drawingsExceedCredit {
		return Tolerance{Source: "none — art. 30(b) allows its tolerance only while the total of the drawings stays within the credit amount"}
	}
	return Tolerance{
		PlusPercent:  t.QuantityTolerancePercent,
		MinusPercent: t.QuantityTolerancePercent,
		Source:       fmt.Sprintf("%s either way — UCP 600 art. 30(b)", percentText(t.QuantityTolerancePercent)),
	}
}

// RoundTo rounds half away from zero to a number of decimal places, which is
// how an invoice is totalled and how a bank reads it.
func RoundTo(v float64, decimals int) float64 {
	if decimals < 0 {
		decimals = 0
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	f := math.Pow(10, float64(decimals))
	return math.Round(v*f) / f
}

// epsilon is half a minor unit: two amounts closer than this are the same
// amount, whatever the arithmetic that produced them.
func epsilon(decimals int) float64 {
	if decimals < 0 {
		decimals = 0
	}
	return 0.5 / math.Pow(10, float64(decimals))
}

// AmountsEqual compares two amounts at the currency's precision.
func AmountsEqual(a, b float64, decimals int) bool {
	return math.Abs(a-b) < epsilon(decimals)
}

// AmountExceeds reports whether a is above b by more than a rounding remainder.
func AmountExceeds(a, b float64, decimals int) bool {
	return a-b >= epsilon(decimals)
}

// AmountBelow reports whether a is under b by more than a rounding remainder.
func AmountBelow(a, b float64, decimals int) bool {
	return b-a >= epsilon(decimals)
}

// ParseAmount reads an amount the way it is written on paperwork: with a
// currency symbol, with thousands separators either Western (1,250,000.00) or
// Indian (12,50,000.00), and with spaces. It refuses anything that is not a
// number rather than silently reading part of it.
func ParseAmount(s string) (float64, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return 0, nil
	}
	var b strings.Builder
	for _, r := range t {
		switch {
		case r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		case r == ',', r == ' ', r == ' ', r == '\'':
			// a separator, in any of the groupings used in practice
		case r == '₹', r == '$', r == '€', r == '£', r == '¥', r == '₱':
			// a currency symbol in front of the figure
		default:
			if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				// a currency code such as USD written beside the figure
				continue
			}
			return 0, fmt.Errorf("%q is not an amount", s)
		}
	}
	cleaned := b.String()
	if cleaned == "" || cleaned == "-" || cleaned == "." {
		return 0, fmt.Errorf("%q is not an amount", s)
	}
	v, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not an amount", s)
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("%q is not an amount", s)
	}
	return v, nil
}

// Money renders an amount with its currency for a report line.
func Money(currency string, v float64, decimals int) string {
	code := strings.ToUpper(strings.TrimSpace(currency))
	text := strconv.FormatFloat(RoundTo(v, decimals), 'f', decimals, 64)
	if code == "" {
		return text
	}
	return code + " " + text
}

// percentText renders a percentage without a trailing ".0".
func percentText(p float64) string {
	if p == math.Trunc(p) {
		return strconv.FormatFloat(p, 'f', 0, 64) + "%"
	}
	return strconv.FormatFloat(p, 'f', -1, 64) + "%"
}

// Quantity renders a quantity with its unit.
func Quantity(v float64, unit string) string {
	text := strconv.FormatFloat(RoundTo(v, 3), 'f', -1, 64)
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return text
	}
	return text + " " + unit
}

// plural picks a word form for a count.
func plural(n int, one, many string) string {
	if n == 1 || n == -1 {
		return strconv.Itoa(n) + " " + one
	}
	return strconv.Itoa(n) + " " + many
}

// pick chooses a word form for a count without printing the count. It exists
// because plural() is wrong in a heading: "1 A document is missing" reads as a
// mistake, which is what a reader would take it for.
func pick(n int, one, many string) string {
	if n == 1 || n == -1 {
		return one
	}
	return many
}

// PercentOf returns part as a percentage of whole, or zero where there is
// nothing to divide by. It multiplies before dividing: the other order turns
// 110 of 100 into 110.00000000000001, which then prints in a report.
func PercentOf(part, whole float64) float64 {
	if whole == 0 {
		return 0
	}
	return part * 100 / whole
}
