package ucp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Date is a calendar day with no time and no zone. Every date in a documentary
// credit is a calendar day: a bill of lading dated 10 March is dated 10 March
// wherever it is read, and comparing it against an expiry date must not depend
// on anybody's clock.
type Date struct {
	Y int
	M int
	D int
}

// NewDate builds a date and reports whether it is a day that exists. 31 April
// and 29 February 2023 do not.
func NewDate(y, m, d int) (Date, bool) {
	if m < 1 || m > 12 || d < 1 || y < 1 {
		return Date{}, false
	}
	if d > DaysInMonth(y, m) {
		return Date{}, false
	}
	return Date{y, m, d}, true
}

// Zero reports whether the date is unset. An unset date is not an error on its
// own: most fields of a credit are optional, and a rule that needs a date it
// has not been given says so rather than guessing.
func (d Date) Zero() bool { return d.Y == 0 && d.M == 0 && d.D == 0 }

// Valid reports whether the date is a real calendar day.
func (d Date) Valid() bool {
	_, ok := NewDate(d.Y, d.M, d.D)
	return ok
}

// String renders the date as yyyy-mm-dd, the form the interface and the data
// file use. An unset date renders as the empty string.
func (d Date) String() string {
	if d.Zero() {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.Y, d.M, d.D)
}

// Human renders the date the way a document would show it: 10 March 2026.
func (d Date) Human() string {
	if d.Zero() || !d.Valid() {
		return "—"
	}
	return fmt.Sprintf("%d %s %d", d.D, time.Month(d.M).String(), d.Y)
}

// MarshalJSON writes the date as a yyyy-mm-dd string, or null when unset, so
// the data file reads as plainly as the interface shows it.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.Zero() {
		return []byte(`""`), nil
	}
	return json.Marshal(d.String())
}

// UnmarshalJSON accepts a yyyy-mm-dd string, an empty string or null.
func (d *Date) UnmarshalJSON(raw []byte) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("a date must be written as yyyy-mm-dd in quotes: %w", err)
	}
	if strings.TrimSpace(s) == "" {
		*d = Date{}
		return nil
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// ParseDate reads the date forms an exporter is likely to type: the ISO form
// 2026-03-10, and the day-first forms 10/03/2026 and 10-03-2026 that both
// Indian and Philippine paperwork use. It refuses anything ambiguous rather
// than guessing: 03/10/2026 day-first is 3 October, and this tool will not
// decide for the user which of two readings a slashed date carries, so a
// day-first reading is applied consistently and stated in the documentation.
func ParseDate(s string) (Date, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return Date{}, nil
	}

	sep := ""
	switch {
	case strings.Count(t, "-") == 2:
		sep = "-"
	case strings.Count(t, "/") == 2:
		sep = "/"
	case strings.Count(t, ".") == 2:
		sep = "."
	default:
		return Date{}, fmt.Errorf("%q is not a date — write it as yyyy-mm-dd, for example 2026-03-10", s)
	}

	parts := strings.Split(t, sep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return Date{}, fmt.Errorf("%q is not a date — write it as yyyy-mm-dd, for example 2026-03-10", s)
		}
	}

	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return Date{}, fmt.Errorf("%q is not a date — write it as yyyy-mm-dd, for example 2026-03-10", s)
		}
		nums[i] = n
	}

	var y, m, d int
	if len(parts[0]) == 4 {
		y, m, d = nums[0], nums[1], nums[2] // yyyy-mm-dd
	} else if len(parts[2]) == 4 {
		d, m, y = nums[0], nums[1], nums[2] // dd/mm/yyyy, read day first
	} else {
		return Date{}, fmt.Errorf("%q needs a four-digit year — write it as yyyy-mm-dd, for example 2026-03-10", s)
	}

	out, ok := NewDate(y, m, d)
	if !ok {
		return Date{}, fmt.Errorf("%s is not a day that exists", s)
	}
	return out, nil
}

// MustDate is for tests and for the worked example. It panics on a date that
// does not exist, which in those two callers is a programming mistake.
func MustDate(s string) Date {
	d, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

// DaysInMonth returns the length of a month, leap years included.
func DaysInMonth(y, m int) int {
	switch m {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if LeapYear(y) {
			return 29
		}
		return 28
	}
	return 0
}

// LeapYear applies the Gregorian rule: every fourth year, except centuries,
// except every fourth century. 1900 is not a leap year; 2000 is.
func LeapYear(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

// time converts to a UTC midnight instant for arithmetic only. It never leaves
// the package.
func (d Date) time() time.Time {
	return time.Date(d.Y, time.Month(d.M), d.D, 0, 0, 0, 0, time.UTC)
}

func fromTime(t time.Time) Date {
	return Date{t.Year(), int(t.Month()), t.Day()}
}

// AddDays moves a date by a number of calendar days, which is what every period
// in UCP 600 is counted in unless it says banking days.
func (d Date) AddDays(n int) Date {
	if d.Zero() {
		return d
	}
	return fromTime(d.time().AddDate(0, 0, n))
}

// Before, After and Equal compare two days.
func (d Date) Before(o Date) bool { return d.compare(o) < 0 }
func (d Date) After(o Date) bool  { return d.compare(o) > 0 }
func (d Date) Equal(o Date) bool  { return d.compare(o) == 0 }

// OnOrBefore and OnOrAfter read the way the rules do: the words "to", "until",
// "till", "from" and "between", when used to determine a period of shipment,
// include the dates mentioned (UCP 600 article 3), so a deadline day is itself
// still within time.
func (d Date) OnOrBefore(o Date) bool { return d.compare(o) <= 0 }
func (d Date) OnOrAfter(o Date) bool  { return d.compare(o) >= 0 }

func (d Date) compare(o Date) int {
	switch {
	case d.Y != o.Y:
		return sign(d.Y - o.Y)
	case d.M != o.M:
		return sign(d.M - o.M)
	case d.D != o.D:
		return sign(d.D - o.D)
	}
	return 0
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// DaysBetween counts calendar days from d to o, positive when o is later. The
// day d itself is not counted, so 10 March to 31 March is 21 days — which is
// how article 14(c)'s "not later than 21 calendar days after the date of
// shipment" is counted.
func DaysBetween(d, o Date) int {
	if d.Zero() || o.Zero() {
		return 0
	}
	return int(o.time().Sub(d.time()).Hours() / 24)
}

// Weekday returns the day of the week.
func (d Date) Weekday() time.Weekday { return d.time().Weekday() }

// BankingDay reports whether a bank is open on a date, given the weekend the
// thresholds name and the closed days the credit lists. UCP 600 does not define
// a banking day as a list of dates — article 2 defines it as a day on which a
// bank is regularly open at the place at which an act subject to the rules is
// to be performed — so the days come from the user, who knows their own bank's
// holidays.
func (t Thresholds) BankingDay(d Date, closed []Date) bool {
	if d.Zero() {
		return false
	}
	if t.weekend(d.Weekday()) {
		return false
	}
	for _, c := range closed {
		if c.Equal(d) {
			return false
		}
	}
	return true
}

// NextBankingDay returns the first day from d onward on which the bank is open.
// It gives up after a year rather than looping forever on a closed-days list
// that covers every weekday.
func (t Thresholds) NextBankingDay(d Date, closed []Date) (Date, bool) {
	if d.Zero() {
		return d, false
	}
	for i := 0; i < 366; i++ {
		c := d.AddDays(i)
		if t.BankingDay(c, closed) {
			return c, true
		}
	}
	return d, false
}

// LastDayForPresentation applies article 29(a): if the expiry date of a credit
// or the last day for presentation falls on a day when the bank to which
// presentation is to be made is closed for reasons other than force majeure
// (article 36), it is extended to the first following banking day. It returns
// the day presentation may still be made on, and whether the extension moved
// it.
//
// Article 29(c) is the other half of this rule and is deliberately not applied
// here: the latest date for shipment is never extended by article 29(a). See
// the shipment rule, which calls this function for nothing.
func (t Thresholds) LastDayForPresentation(expiry Date, closed []Date) (Date, bool) {
	if expiry.Zero() {
		return expiry, false
	}
	if t.BankingDay(expiry, closed) {
		return expiry, false
	}
	next, ok := t.NextBankingDay(expiry, closed)
	if !ok {
		return expiry, false
	}
	return next, true
}

// AddBankingDays counts n banking days forward from the day after d, which is
// how article 14(b) counts: a maximum of five banking days "following the day
// of presentation". The day of presentation itself is not one of the five, and
// a day the bank is closed is not counted at all.
func (t Thresholds) AddBankingDays(d Date, n int, closed []Date) (Date, bool) {
	if d.Zero() || n < 0 {
		return Date{}, false
	}
	cur := d
	counted := 0
	for i := 0; i < 366*2 && counted < n; i++ {
		cur = cur.AddDays(1)
		if t.BankingDay(cur, closed) {
			counted++
		}
	}
	if counted < n {
		return Date{}, false
	}
	return cur, true
}

// Window is a closed interval of calendar days, both ends included.
type Window struct {
	From Date `json:"from"`
	To   Date `json:"to"`
}

// Contains reports whether a day falls inside the window, both ends included.
func (w Window) Contains(d Date) bool {
	if w.From.Zero() || w.To.Zero() || d.Zero() {
		return false
	}
	return d.OnOrAfter(w.From) && d.OnOrBefore(w.To)
}

func (w Window) String() string {
	if w.From.Zero() || w.To.Zero() {
		return ""
	}
	return w.From.Human() + " to " + w.To.Human()
}

// OnOrAboutWindow applies UCP 600 article 3 to an "on or about" date: five
// calendar days before until five calendar days after, both start and end dates
// included.
func (t Thresholds) OnOrAboutWindow(d Date) Window {
	if d.Zero() {
		return Window{}
	}
	n := t.Normalise().OnOrAboutDays
	return Window{From: d.AddDays(-n), To: d.AddDays(n)}
}

// MonthPart names the parts of a month that article 3 defines, so that a credit
// reading "shipment during the first half of March 2026" can be turned into the
// window the rules give it.
type MonthPart string

const (
	MonthWhole      MonthPart = "whole"
	MonthFirstHalf  MonthPart = "first-half"
	MonthSecondHalf MonthPart = "second-half"
	MonthBeginning  MonthPart = "beginning"
	MonthMiddle     MonthPart = "middle"
	MonthEnd        MonthPart = "end"
)

// MonthPartWindow turns a month and one of article 3's expressions into the
// window of days it covers, all dates inclusive. It is the one place the day
// numbers 10, 11, 15, 16, 20 and 21 appear, and they come from the named
// constants above.
func MonthPartWindow(y, m int, part MonthPart) (Window, error) {
	if m < 1 || m > 12 || y < 1 {
		return Window{}, fmt.Errorf("%d-%02d is not a month", y, m)
	}
	last := DaysInMonth(y, m)
	first, end := 1, last
	switch part {
	case MonthWhole, "":
		// the whole month
	case MonthFirstHalf:
		end = ArtFirstHalfLastDay
	case MonthSecondHalf:
		first = ArtSecondHalfFirstDay
	case MonthBeginning:
		end = ArtBeginningLastDay
	case MonthMiddle:
		first, end = ArtMiddleFirstDay, ArtMiddleLastDay
	case MonthEnd:
		first = ArtEndOfMonthFirstDay
	default:
		return Window{}, fmt.Errorf("%q is not one of the expressions article 3 defines", part)
	}
	from, ok := NewDate(y, m, first)
	if !ok {
		return Window{}, fmt.Errorf("%d-%02d-%02d is not a day that exists", y, m, first)
	}
	to, ok := NewDate(y, m, end)
	if !ok {
		return Window{}, fmt.Errorf("%d-%02d-%02d is not a day that exists", y, m, end)
	}
	return Window{From: from, To: to}, nil
}

// Today is the only place this package reads the clock, and only the local
// calendar day, so that a report says which day it was produced on.
func Today() Date { return fromTime(time.Now()) }
