package ucp

import (
	"reflect"
	"testing"
	"time"
)

func TestStatutoryThresholdsMatchTheCitedConstants(t *testing.T) {
	th := StatutoryThresholds()
	cases := []struct {
		name      string
		got, want float64
	}{
		{"presentation period, art. 14(c)", float64(th.PresentationPeriodDays), ArtPresentationPeriodDays},
		{"examination days, arts. 14(b)/16(d)", float64(th.ExaminationBankingDays), ArtExaminationBankingDays},
		{"\"on or about\", art. 3", float64(th.OnOrAboutDays), ArtOnOrAboutDays},
		{"\"about\" tolerance, art. 30(a)", th.AboutTolerancePercent, ArtAboutTolerancePercent},
		{"quantity tolerance, art. 30(b)", th.QuantityTolerancePercent, ArtQuantityTolerancePercent},
		{"short-draw tolerance, art. 30(c)", th.ShortDrawTolerancePercent, ArtShortDrawTolerancePercent},
		{"minimum insurance, art. 28(f)(ii)", th.MinInsurancePercent, ArtMinInsurancePercent},
		{"originals per document, art. 17(a)", float64(th.MinOriginalsPerDocument), ArtMinOriginalsPerDocument},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: thresholds say %v, the constant says %v", c.name, c.got, c.want)
		}
	}
	if th.Edited {
		t.Errorf("the statutory thresholds must not report themselves as edited")
	}
	// The figures themselves, asserted against the article text rather than
	// against the constants, so that changing a constant without meaning to
	// fails here.
	if ArtPresentationPeriodDays != 21 || ArtExaminationBankingDays != 5 || ArtOnOrAboutDays != 5 {
		t.Errorf("art. 14(c) is 21 calendar days, arts. 14(b)/16(d) five banking days, art. 3 five days either side")
	}
	if ArtAboutTolerancePercent != 10 || ArtQuantityTolerancePercent != 5 || ArtShortDrawTolerancePercent != 5 || ArtMinInsurancePercent != 110 {
		t.Errorf("art. 30(a) is 10%%, arts. 30(b) and 30(c) are 5%%, art. 28(f)(ii) is 110%%")
	}
	if ArtFirstHalfLastDay != 15 || ArtSecondHalfFirstDay != 16 || ArtBeginningLastDay != 10 ||
		ArtMiddleFirstDay != 11 || ArtMiddleLastDay != 20 || ArtEndOfMonthFirstDay != 21 {
		t.Errorf("the month-part days of art. 3 are 1-15, 16-end, 1-10, 11-20 and 21-end")
	}
}

func TestThresholdsNormaliseFillsBlanksAndRejectsNonsense(t *testing.T) {
	statute := StatutoryThresholds()

	cases := []struct {
		name  string
		in    Thresholds
		check func(t *testing.T, got Thresholds)
	}{
		{"an empty set becomes the statutory one", Thresholds{}, func(t *testing.T, got Thresholds) {
			if got.PresentationPeriodDays != statute.PresentationPeriodDays ||
				got.MinInsurancePercent != statute.MinInsurancePercent ||
				got.AboutTolerancePercent != statute.AboutTolerancePercent {
				t.Fatalf("blanks were not filled: %+v", got)
			}
			if got.Edited {
				t.Fatalf("an empty set is not an edited set")
			}
		}},
		{"a credit's own presentation period is kept", Thresholds{PresentationPeriodDays: 15}, func(t *testing.T, got Thresholds) {
			if got.PresentationPeriodDays != 15 {
				t.Fatalf("got %d days, want the 15 entered", got.PresentationPeriodDays)
			}
			if !got.Edited {
				t.Fatalf("a changed figure must be reported as edited")
			}
		}},
		{"negatives fall back", Thresholds{
			PresentationPeriodDays: -5, ExaminationBankingDays: -1, OnOrAboutDays: -1,
			AboutTolerancePercent: -10, QuantityTolerancePercent: -5, ShortDrawTolerancePercent: -5,
			MinInsurancePercent: -110, MinOriginalsPerDocument: -1, TypoEditDistance: -3, AmountDecimals: -2,
		}, func(t *testing.T, got Thresholds) {
			if !reflect.DeepEqual(got, statute) {
				t.Fatalf("negatives did not fall back to the statute:\n got  %+v\n want %+v", got, statute)
			}
		}},
		{"an absurd precision falls back", Thresholds{AmountDecimals: 9}, func(t *testing.T, got Thresholds) {
			if got.AmountDecimals != ConventionAmountDecimals {
				t.Fatalf("got %d decimals, want the convention %d", got.AmountDecimals, ConventionAmountDecimals)
			}
		}},
		{"an absent decimals field reads as the convention, not as zero", Thresholds{PresentationPeriodDays: 21}, func(t *testing.T, got Thresholds) {
			if got.AmountDecimals != ConventionAmountDecimals {
				t.Fatalf("got %d decimals, want the convention %d", got.AmountDecimals, ConventionAmountDecimals)
			}
		}},
		{"zero decimals chosen deliberately is kept", Thresholds{AmountDecimals: 0, AmountDecimalsSet: true}, func(t *testing.T, got Thresholds) {
			if got.AmountDecimals != 0 {
				t.Fatalf("got %d decimals, want the 0 chosen for a currency with no minor unit", got.AmountDecimals)
			}
			if !got.Edited {
				t.Fatalf("choosing zero decimals is a change from the convention")
			}
		}},
		{"a weekend of two days is kept in order", Thresholds{WeekendDays: []int{5, 6}}, func(t *testing.T, got Thresholds) {
			if len(got.WeekendDays) != 2 || got.WeekendDays[0] != 5 || got.WeekendDays[1] != 6 {
				t.Fatalf("weekend = %v, want Friday and Saturday", got.WeekendDays)
			}
			if !got.Edited {
				t.Fatalf("a changed weekend must be reported as edited")
			}
		}},
		{"impossible weekdays are dropped", Thresholds{WeekendDays: []int{9, -2, 0, 0, 6}}, func(t *testing.T, got Thresholds) {
			if len(got.WeekendDays) != 2 || got.WeekendDays[0] != 0 || got.WeekendDays[1] != 6 {
				t.Fatalf("weekend = %v, want Sunday and Saturday with the rubbish dropped", got.WeekendDays)
			}
		}},
		{"an empty weekend falls back to Saturday and Sunday", Thresholds{WeekendDays: []int{}}, func(t *testing.T, got Thresholds) {
			if len(got.WeekendDays) != 2 {
				t.Fatalf("weekend = %v, want two days", got.WeekendDays)
			}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.check(t, c.in.Normalise())
		})
	}
}

func TestThresholdsWeekendLookup(t *testing.T) {
	th := StatutoryThresholds()
	if !th.weekend(time.Saturday) || !th.weekend(time.Sunday) {
		t.Errorf("the default weekend must be Saturday and Sunday")
	}
	if th.weekend(time.Wednesday) {
		t.Errorf("Wednesday is not the weekend")
	}

	gulf := Thresholds{WeekendDays: []int{int(time.Friday), int(time.Saturday)}}.Normalise()
	if !gulf.weekend(time.Friday) || gulf.weekend(time.Sunday) {
		t.Errorf("a Friday-Saturday weekend was not applied")
	}
}

func TestCitationsCoverEveryFigureAndSayWhereItComesFrom(t *testing.T) {
	cites := StatutoryThresholds().Citations()
	if len(cites) < 11 {
		t.Fatalf("only %d figures are cited; every figure the tool applies must be", len(cites))
	}

	wantKeys := []string{
		"presentationPeriodDays", "examinationBankingDays", "onOrAboutDays",
		"aboutTolerancePercent", "quantityTolerancePercent", "shortDrawTolerancePercent",
		"minInsurancePercent", "minOriginalsPerDocument", "typoEditDistance",
		"amountDecimals", "weekendDays",
	}
	have := map[string]Citation{}
	for _, c := range cites {
		have[c.Key] = c
	}
	for _, k := range wantKeys {
		c, ok := have[k]
		if !ok {
			t.Errorf("%s is applied but not cited", k)
			continue
		}
		if c.Figure == "" || c.Source == "" {
			t.Errorf("%s is cited without a figure or a source: %+v", k, c)
		}
	}

	// The three conventions must say plainly that they are not UCP 600.
	for _, k := range []string{"typoEditDistance", "amountDecimals", "weekendDays"} {
		if got := have[k].Source; got == "" || got[:7] != "Not UCP" {
			t.Errorf("%s is a convention of this tool and must say so, got %q", k, got)
		}
	}
	// And the statutory ones must carry an article number.
	for _, k := range []string{"presentationPeriodDays", "minInsurancePercent", "quantityTolerancePercent"} {
		if got := have[k].Source; len(got) < 8 || got[:8] != "UCP 600 " {
			t.Errorf("%s must cite a UCP 600 article, got %q", k, got)
		}
	}
}

func TestEditedFlagTracksEveryField(t *testing.T) {
	cases := []struct {
		name string
		in   Thresholds
	}{
		{"presentation period", Thresholds{PresentationPeriodDays: 15}},
		{"examination days", Thresholds{ExaminationBankingDays: 3}},
		{"on or about", Thresholds{OnOrAboutDays: 3}},
		{"about tolerance", Thresholds{AboutTolerancePercent: 7}},
		{"quantity tolerance", Thresholds{QuantityTolerancePercent: 3}},
		{"short draw tolerance", Thresholds{ShortDrawTolerancePercent: 3}},
		{"minimum insurance", Thresholds{MinInsurancePercent: 120}},
		{"originals", Thresholds{MinOriginalsPerDocument: 2}},
		{"typo allowance", Thresholds{TypoEditDistance: 1}},
		{"decimals", Thresholds{AmountDecimals: 3, AmountDecimalsSet: true}},
		{"weekend", Thresholds{WeekendDays: []int{5}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !c.in.Normalise().Edited {
				t.Fatalf("changing the %s must set the edited flag", c.name)
			}
		})
	}
	if StatutoryThresholds().Normalise().Edited {
		t.Fatalf("normalising the statutory set must not mark it edited")
	}
}

func TestWeekdayNames(t *testing.T) {
	cases := []struct {
		in   []int
		want string
	}{
		{[]int{6, 0}, "Saturday, Sunday"},
		{[]int{5}, "Friday"},
		{nil, "none"},
	}
	for _, c := range cases {
		if got := weekdayNames(c.in); got != c.want {
			t.Errorf("weekdayNames(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
