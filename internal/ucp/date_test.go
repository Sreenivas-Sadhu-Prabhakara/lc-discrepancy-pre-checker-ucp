package ucp

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    Date
		wantErr bool
	}{
		{"iso", "2026-03-10", Date{2026, 3, 10}, false},
		{"iso with spaces", "  2026-03-10 ", Date{2026, 3, 10}, false},
		{"iso single digit parts", "2026-3-9", Date{2026, 3, 9}, false},
		{"day first slashes", "10/03/2026", Date{2026, 3, 10}, false},
		{"day first dashes", "10-03-2026", Date{2026, 3, 10}, false},
		{"day first dots", "10.03.2026", Date{2026, 3, 10}, false},
		{"day first beyond 12 is unambiguous", "29/02/2024", Date{2024, 2, 29}, false},
		{"empty is the unset date", "", Date{}, false},
		{"whitespace is the unset date", "   ", Date{}, false},
		{"leap day in a non-leap year", "2023-02-29", Date{}, true},
		{"31 April", "2026-04-31", Date{}, true},
		{"month zero", "2026-00-10", Date{}, true},
		{"month thirteen", "2026-13-10", Date{}, true},
		{"day zero", "2026-03-00", Date{}, true},
		{"two digit year", "10/03/26", Date{}, true},
		{"not a date at all", "next Tuesday", Date{}, true},
		{"letters in the parts", "2026-mar-10", Date{}, true},
		{"too few parts", "2026-03", Date{}, true},
		{"too many parts", "2026-03-10-01", Date{}, true},
		{"empty part", "2026--10", Date{}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseDate(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("ParseDate(%q) = %v, want an error", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDate(%q): %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("ParseDate(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestDateStringAndHuman(t *testing.T) {
	cases := []struct {
		in         Date
		wantString string
		wantHuman  string
	}{
		{Date{2026, 3, 10}, "2026-03-10", "10 March 2026"},
		{Date{2026, 12, 1}, "2026-12-01", "1 December 2026"},
		{Date{}, "", "—"},
		{Date{2023, 2, 29}, "2023-02-29", "—"}, // an impossible day renders as unknown
	}
	for _, c := range cases {
		if got := c.in.String(); got != c.wantString {
			t.Errorf("%v.String() = %q, want %q", c.in, got, c.wantString)
		}
		if got := c.in.Human(); got != c.wantHuman {
			t.Errorf("%v.Human() = %q, want %q", c.in, got, c.wantHuman)
		}
	}
}

func TestDateJSONRoundTrip(t *testing.T) {
	type holder struct {
		When Date `json:"when"`
	}
	cases := []struct {
		name string
		raw  string
		want Date
		err  bool
	}{
		{"iso string", `{"when":"2026-03-10"}`, Date{2026, 3, 10}, false},
		{"empty string", `{"when":""}`, Date{}, false},
		{"absent", `{}`, Date{}, false},
		{"day first", `{"when":"10/03/2026"}`, Date{2026, 3, 10}, false},
		{"impossible day", `{"when":"2023-02-29"}`, Date{}, true},
		{"a number is not a date", `{"when":20260310}`, Date{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var h holder
			err := json.Unmarshal([]byte(c.raw), &h)
			if c.err {
				if err == nil {
					t.Fatalf("Unmarshal(%s) = %v, want an error", c.raw, h.When)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal(%s): %v", c.raw, err)
			}
			if h.When != c.want {
				t.Fatalf("Unmarshal(%s) = %v, want %v", c.raw, h.When, c.want)
			}
			out, err := json.Marshal(h)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var back holder
			if err := json.Unmarshal(out, &back); err != nil {
				t.Fatalf("re-reading %s: %v", out, err)
			}
			if back.When != c.want {
				t.Fatalf("round trip gave %v, want %v", back.When, c.want)
			}
		})
	}
}

func TestLeapYearAndDaysInMonth(t *testing.T) {
	leap := []struct {
		year int
		want bool
	}{
		{2024, true}, {2025, false}, {2000, true}, {1900, false}, {2100, false}, {2400, true},
	}
	for _, c := range leap {
		if got := LeapYear(c.year); got != c.want {
			t.Errorf("LeapYear(%d) = %v, want %v", c.year, got, c.want)
		}
	}

	months := []struct {
		y, m, want int
	}{
		{2026, 1, 31}, {2026, 2, 28}, {2024, 2, 29}, {2000, 2, 29}, {1900, 2, 28},
		{2026, 4, 30}, {2026, 12, 31}, {2026, 0, 0}, {2026, 13, 0},
	}
	for _, c := range months {
		if got := DaysInMonth(c.y, c.m); got != c.want {
			t.Errorf("DaysInMonth(%d, %d) = %d, want %d", c.y, c.m, got, c.want)
		}
	}
}

func TestAddDaysAcrossMonthsAndLeapYears(t *testing.T) {
	cases := []struct {
		name string
		from Date
		add  int
		want Date
	}{
		{"within a month", Date{2026, 3, 10}, 5, Date{2026, 3, 15}},
		{"over a month end", Date{2026, 3, 25}, 21, Date{2026, 4, 15}},
		{"over a year end", Date{2026, 12, 20}, 21, Date{2027, 1, 10}},
		{"into a leap day", Date{2024, 2, 27}, 2, Date{2024, 2, 29}},
		{"over a leap day", Date{2024, 2, 27}, 3, Date{2024, 3, 1}},
		{"over a non-leap February", Date{2026, 2, 27}, 2, Date{2026, 3, 1}},
		{"backwards", Date{2026, 3, 1}, -1, Date{2026, 2, 28}},
		{"backwards over a leap day", Date{2024, 3, 1}, -1, Date{2024, 2, 29}},
		{"zero days", Date{2026, 3, 10}, 0, Date{2026, 3, 10}},
		{"21 days from the end of February in a leap year", Date{2024, 2, 29}, 21, Date{2024, 3, 21}},
		{"the unset date stays unset", Date{}, 10, Date{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.from.AddDays(c.add); got != c.want {
				t.Fatalf("%v.AddDays(%d) = %v, want %v", c.from, c.add, got, c.want)
			}
		})
	}
}

func TestDaysBetween(t *testing.T) {
	cases := []struct {
		name string
		a, b Date
		want int
	}{
		{"same day", Date{2026, 3, 10}, Date{2026, 3, 10}, 0},
		{"21 days, the art. 14(c) count", Date{2026, 3, 10}, Date{2026, 3, 31}, 21},
		{"across February in a leap year", Date{2024, 2, 20}, Date{2024, 3, 1}, 10},
		{"across February in a common year", Date{2026, 2, 20}, Date{2026, 3, 1}, 9},
		{"backwards is negative", Date{2026, 3, 31}, Date{2026, 3, 10}, -21},
		{"a year", Date{2026, 1, 1}, Date{2027, 1, 1}, 365},
		{"a leap year", Date{2024, 1, 1}, Date{2025, 1, 1}, 366},
		{"an unset date gives zero", Date{}, Date{2026, 3, 10}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DaysBetween(c.a, c.b); got != c.want {
				t.Fatalf("DaysBetween(%v, %v) = %d, want %d", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestComparisonsIncludeTheDateMentioned(t *testing.T) {
	a, b := Date{2026, 3, 10}, Date{2026, 3, 10}
	if !a.Equal(b) || a.Before(b) || a.After(b) {
		t.Fatalf("equal dates compared wrongly")
	}
	// Article 3: "to", "until", "till" and "between" include the date
	// mentioned, so the deadline day itself is within time.
	if !a.OnOrBefore(b) || !a.OnOrAfter(b) {
		t.Fatalf("the date mentioned must be included")
	}
	later := Date{2026, 3, 11}
	if !a.Before(later) || !later.After(a) || later.OnOrBefore(a) {
		t.Fatalf("ordering is wrong")
	}
	// Year, month and day each have to be compared in turn.
	if !(Date{2025, 12, 31}).Before(Date{2026, 1, 1}) {
		t.Fatalf("year did not dominate")
	}
	if !(Date{2026, 1, 31}).Before(Date{2026, 2, 1}) {
		t.Fatalf("month did not dominate the day")
	}
}

func TestOnOrAboutWindowIsFiveDaysEitherSideInclusive(t *testing.T) {
	th := StatutoryThresholds()
	w := th.OnOrAboutWindow(Date{2026, 3, 10})
	if w.From != (Date{2026, 3, 5}) || w.To != (Date{2026, 3, 15}) {
		t.Fatalf("window = %v, want 5 to 15 March", w)
	}
	cases := []struct {
		day  Date
		want bool
	}{
		{Date{2026, 3, 5}, true},  // first day included
		{Date{2026, 3, 15}, true}, // last day included
		{Date{2026, 3, 4}, false},
		{Date{2026, 3, 16}, false},
		{Date{2026, 3, 10}, true},
	}
	for _, c := range cases {
		if got := w.Contains(c.day); got != c.want {
			t.Errorf("Contains(%v) = %v, want %v", c.day, got, c.want)
		}
	}

	// Across a month boundary, and across a leap day.
	if w := th.OnOrAboutWindow(Date{2026, 3, 2}); w.From != (Date{2026, 2, 25}) {
		t.Errorf("window from 2 March = %v, want from 25 February", w)
	}
	if w := th.OnOrAboutWindow(Date{2024, 3, 2}); w.From != (Date{2024, 2, 26}) {
		t.Errorf("window from 2 March 2024 = %v, want from 26 February (29 days in that February)", w)
	}
	if w := th.OnOrAboutWindow(Date{}); w.From != (Date{}) || w.To != (Date{}) {
		t.Errorf("an unset date must give an empty window, got %v", w)
	}
	if got := (Window{}).String(); got != "" {
		t.Errorf("an empty window renders as %q, want empty", got)
	}
}

func TestMonthPartWindow(t *testing.T) {
	cases := []struct {
		name     string
		y, m     int
		part     MonthPart
		from, to Date
		wantErr  bool
	}{
		{"first half of March", 2026, 3, MonthFirstHalf, Date{2026, 3, 1}, Date{2026, 3, 15}, false},
		{"second half of March", 2026, 3, MonthSecondHalf, Date{2026, 3, 16}, Date{2026, 3, 31}, false},
		{"second half of February, common year", 2026, 2, MonthSecondHalf, Date{2026, 2, 16}, Date{2026, 2, 28}, false},
		{"second half of February, leap year", 2024, 2, MonthSecondHalf, Date{2024, 2, 16}, Date{2024, 2, 29}, false},
		{"beginning of a month", 2026, 3, MonthBeginning, Date{2026, 3, 1}, Date{2026, 3, 10}, false},
		{"middle of a month", 2026, 3, MonthMiddle, Date{2026, 3, 11}, Date{2026, 3, 20}, false},
		{"end of a month", 2026, 4, MonthEnd, Date{2026, 4, 21}, Date{2026, 4, 30}, false},
		{"the whole month", 2026, 4, MonthWhole, Date{2026, 4, 1}, Date{2026, 4, 30}, false},
		{"an empty part is the whole month", 2026, 4, "", Date{2026, 4, 1}, Date{2026, 4, 30}, false},
		{"an unknown expression", 2026, 4, "last week", Date{}, Date{}, true},
		{"a month that does not exist", 2026, 13, MonthWhole, Date{}, Date{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := MonthPartWindow(c.y, c.m, c.part)
			if c.wantErr {
				if err == nil {
					t.Fatalf("MonthPartWindow(%d, %d, %q) = %v, want an error", c.y, c.m, c.part, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("MonthPartWindow: %v", err)
			}
			if got.From != c.from || got.To != c.to {
				t.Fatalf("got %v to %v, want %v to %v", got.From, got.To, c.from, c.to)
			}
		})
	}
}

func TestBankingDays(t *testing.T) {
	th := StatutoryThresholds()
	// 2026-03-28 is a Saturday and 2026-03-29 a Sunday.
	if got := (Date{2026, 3, 28}).Weekday(); got != time.Saturday {
		t.Fatalf("28 March 2026 is a %v, the test's premise is wrong", got)
	}

	closed := []Date{{2026, 3, 26}}
	cases := []struct {
		day  Date
		want bool
	}{
		{Date{2026, 3, 25}, true},  // Wednesday
		{Date{2026, 3, 26}, false}, // a listed holiday
		{Date{2026, 3, 27}, true},  // Friday
		{Date{2026, 3, 28}, false}, // Saturday
		{Date{2026, 3, 29}, false}, // Sunday
		{Date{2026, 3, 30}, true},  // Monday
		{Date{}, false},            // the unset date is not a banking day
	}
	for _, c := range cases {
		if got := th.BankingDay(c.day, closed); got != c.want {
			t.Errorf("BankingDay(%v) = %v, want %v", c.day, got, c.want)
		}
	}

	// A bank with no open day at all must not loop forever.
	everyDay := Thresholds{WeekendDays: []int{0, 1, 2, 3, 4, 5, 6}}.Normalise()
	if _, ok := everyDay.NextBankingDay(Date{2026, 3, 25}, nil); ok {
		t.Errorf("a bank closed every day cannot have a next banking day")
	}
	if _, ok := everyDay.AddBankingDays(Date{2026, 3, 25}, 5, nil); ok {
		t.Errorf("counting banking days at a bank that never opens must fail rather than hang")
	}
}

func TestLastDayForPresentationAppliesArticle29a(t *testing.T) {
	th := StatutoryThresholds()
	cases := []struct {
		name     string
		expiry   Date
		closed   []Date
		want     Date
		extended bool
	}{
		{"a banking day is not moved", Date{2026, 3, 31}, nil, Date{2026, 3, 31}, false},
		{"a Saturday moves to Monday", Date{2026, 3, 28}, nil, Date{2026, 3, 30}, true},
		{"a Sunday moves to Monday", Date{2026, 3, 29}, nil, Date{2026, 3, 30}, true},
		{"a holiday moves to the next open day", Date{2026, 3, 26}, []Date{{2026, 3, 26}}, Date{2026, 3, 27}, true},
		{"a holiday before a weekend moves to Monday", Date{2026, 3, 27}, []Date{{2026, 3, 27}}, Date{2026, 3, 30}, true},
		{"an unset expiry stays unset", Date{}, nil, Date{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, extended := th.LastDayForPresentation(c.expiry, c.closed)
			if got != c.want || extended != c.extended {
				t.Fatalf("got %v (extended %v), want %v (extended %v)", got, extended, c.want, c.extended)
			}
		})
	}
}

func TestAddBankingDaysCountsFromTheDayAfter(t *testing.T) {
	th := StatutoryThresholds()
	cases := []struct {
		name   string
		from   Date
		n      int
		closed []Date
		want   Date
	}{
		// Presented Monday 23 March 2026: the five banking days following are
		// Tue 24, Wed 25, Thu 26, Fri 27 and Mon 30 — the weekend is not counted.
		{"five banking days over a weekend", Date{2026, 3, 23}, 5, nil, Date{2026, 3, 30}},
		{"five banking days with a holiday in them", Date{2026, 3, 23}, 5, []Date{{2026, 3, 25}}, Date{2026, 3, 31}},
		{"one banking day", Date{2026, 3, 23}, 1, nil, Date{2026, 3, 24}},
		{"one banking day from a Friday", Date{2026, 3, 27}, 1, nil, Date{2026, 3, 30}},
		{"zero banking days is the day itself", Date{2026, 3, 23}, 0, nil, Date{2026, 3, 23}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := th.AddBankingDays(c.from, c.n, c.closed)
			if !ok {
				t.Fatalf("AddBankingDays(%v, %d) failed", c.from, c.n)
			}
			if got != c.want {
				t.Fatalf("AddBankingDays(%v, %d) = %v, want %v", c.from, c.n, got, c.want)
			}
		})
	}
	if _, ok := th.AddBankingDays(Date{}, 5, nil); ok {
		t.Errorf("counting from an unset date must fail")
	}
	if _, ok := th.AddBankingDays(Date{2026, 3, 23}, -1, nil); ok {
		t.Errorf("a negative count must fail")
	}
}

func TestMustDatePanicsOnNonsense(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("MustDate must panic on a day that does not exist")
		}
	}()
	MustDate("2023-02-29")
}

func TestTodayIsAValidDay(t *testing.T) {
	if got := Today(); !got.Valid() {
		t.Fatalf("Today() = %v, which is not a valid day", got)
	}
}
