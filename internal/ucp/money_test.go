package ucp

import "testing"

func TestParseAmount(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    float64
		wantErr bool
	}{
		{"plain", "48600", 48600, false},
		{"with decimals", "48600.50", 48600.50, false},
		{"western grouping", "1,250,000.00", 1250000, false},
		{"indian grouping", "12,50,000.00", 1250000, false},
		{"rupee symbol", "₹1,25,000", 125000, false},
		{"peso symbol", "₱48,600.00", 48600, false},
		{"dollar symbol", "$48,600.00", 48600, false},
		{"currency code beside the figure", "USD 48600.00", 48600, false},
		{"spaces as separators", "48 600.00", 48600, false},
		{"apostrophe grouping", "48'600.00", 48600, false},
		{"negative", "-100.25", -100.25, false},
		{"empty is zero", "", 0, false},
		{"whitespace is zero", "   ", 0, false},
		{"not a number", "forty thousand!", 0, true},
		{"a bare minus", "-", 0, true},
		{"a bare point", ".", 0, true},
		{"two points", "1.2.3", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseAmount(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("ParseAmount(%q) = %v, want an error", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAmount(%q): %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("ParseAmount(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestRoundToRoundsHalfAway(t *testing.T) {
	cases := []struct {
		in       float64
		decimals int
		want     float64
	}{
		// 0.125 is one of the few halves a float64 holds exactly, so it is the
		// case that really tests "half away from zero".
		{0.125, 2, 0.13},
		{-0.125, 2, -0.13},
		// 1.005 is not exactly representable: the nearest float64 is a hair
		// below it, so it rounds down. No rounding function can do otherwise
		// with a float, and it is asserted here so that nobody later "fixes"
		// it into an inconsistency.
		{1.005, 2, 1.0},
		{1.004, 2, 1.0},
		{2.675, 2, 2.68},
		{48600.555, 2, 48600.56},
		{48600.5, 0, 48601},
		{48600.4, 0, 48600},
		{1.23456, 4, 1.2346},
		{1.5, -1, 2}, // a negative precision is read as none
	}
	for _, c := range cases {
		if got := RoundTo(c.in, c.decimals); got != c.want {
			t.Errorf("RoundTo(%v, %d) = %v, want %v", c.in, c.decimals, got, c.want)
		}
	}
}

func TestAmountComparisons(t *testing.T) {
	cases := []struct {
		name     string
		a, b     float64
		decimals int
		equal    bool
		exceeds  bool
		below    bool
	}{
		{"identical", 100, 100, 2, true, false, false},
		{"a rounding remainder apart", 100, 100.001, 2, true, false, false},
		{"one paisa apart", 100, 100.01, 2, false, false, true},
		{"clearly above", 101, 100, 2, false, true, false},
		{"zero decimals, one unit apart", 100, 101, 0, false, false, true},
		{"zero decimals, a fraction apart", 100, 100.2, 0, true, false, false},
		{"both zero", 0, 0, 2, true, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := AmountsEqual(c.a, c.b, c.decimals); got != c.equal {
				t.Errorf("AmountsEqual(%v, %v, %d) = %v, want %v", c.a, c.b, c.decimals, got, c.equal)
			}
			if got := AmountExceeds(c.a, c.b, c.decimals); got != c.exceeds {
				t.Errorf("AmountExceeds(%v, %v, %d) = %v, want %v", c.a, c.b, c.decimals, got, c.exceeds)
			}
			if got := AmountBelow(c.a, c.b, c.decimals); got != c.below {
				t.Errorf("AmountBelow(%v, %v, %d) = %v, want %v", c.a, c.b, c.decimals, got, c.below)
			}
		})
	}
}

func TestNormaliseQualifier(t *testing.T) {
	cases := map[string]Qualifier{
		"":               QualifierExact,
		"exact":          QualifierExact,
		"about":          QualifierAbout,
		"ABOUT":          QualifierAbout,
		"approximately":  QualifierAbout,
		"approx":         QualifierAbout,
		"circa":          QualifierAbout,
		"maximum":        QualifierMaximum,
		"MAX":            QualifierMaximum,
		"not exceeding":  QualifierMaximum,
		"up to":          QualifierMaximum,
		"something else": QualifierExact,
	}
	for in, want := range cases {
		if got := NormaliseQualifier(Qualifier(in)); got != want {
			t.Errorf("NormaliseQualifier(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAmountToleranceFollowsArticle30a(t *testing.T) {
	th := StatutoryThresholds()
	cases := []struct {
		name        string
		credit      Credit
		plus, minus float64
	}{
		{"an exact amount has no tolerance", Credit{Amount: 100}, 0, 0},
		{"\"about\" gives ten per cent either way", Credit{Amount: 100, AmountQualifier: QualifierAbout}, 10, 10},
		{"\"maximum\" gives nothing", Credit{Amount: 100, AmountQualifier: QualifierMaximum}, 0, 0},
		{"a stated tolerance displaces the default", Credit{Amount: 100, TolerancePlusPercent: 5, ToleranceMinusPercent: 3}, 5, 3},
		{"a stated tolerance beats \"about\"", Credit{Amount: 100, AmountQualifier: QualifierAbout, TolerancePlusPercent: 2}, 2, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AmountTolerance(c.credit.Normalise(), th)
			if got.PlusPercent != c.plus || got.MinusPercent != c.minus {
				t.Fatalf("got +%v/-%v, want +%v/-%v", got.PlusPercent, got.MinusPercent, c.plus, c.minus)
			}
			if got.Source == "" {
				t.Fatalf("every tolerance must say where it comes from")
			}
		})
	}

	// The band is what the comparison actually uses.
	tol := AmountTolerance(Credit{Amount: 100, AmountQualifier: QualifierAbout}.Normalise(), th)
	low, high := tol.Band(100, 2)
	if low != 90 || high != 110 {
		t.Fatalf("band = %v to %v, want 90 to 110", low, high)
	}
}

func TestQuantityToleranceFollowsArticle30b(t *testing.T) {
	th := StatutoryThresholds()
	cases := []struct {
		name        string
		credit      Credit
		exceeds     bool
		plus, minus float64
	}{
		{"the default is five per cent either way",
			Credit{Quantity: 1000, QuantityUnit: "kg"}, false, 5, 5},
		{"goods counted in packing units get nothing",
			Credit{Quantity: 1000, QuantityUnit: "cartons", QuantityInPackingUnits: true}, false, 0, 0},
		{"drawings over the credit amount lose the tolerance",
			Credit{Quantity: 1000, QuantityUnit: "kg"}, true, 0, 0},
		{"\"about\" the quantity gives ten per cent",
			Credit{Quantity: 1000, QuantityUnit: "kg", QuantityQualifier: QualifierAbout}, false, 10, 10},
		{"\"about\" survives packing units, being a different sub-article",
			Credit{Quantity: 1000, QuantityUnit: "cartons", QuantityInPackingUnits: true, QuantityQualifier: QualifierAbout}, false, 10, 10},
		{"\"maximum\" the quantity gives nothing",
			Credit{Quantity: 1000, QuantityUnit: "kg", QuantityQualifier: QualifierMaximum}, false, 0, 0},
		{"a stated quantity tolerance displaces the default",
			Credit{Quantity: 1000, QuantityUnit: "kg", QuantityTolerancePlusPercent: 2, QuantityToleranceMinusPercent: 2}, false, 2, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := QuantityTolerance(c.credit.Normalise(), c.exceeds, th)
			if got.PlusPercent != c.plus || got.MinusPercent != c.minus {
				t.Fatalf("got +%v/-%v, want +%v/-%v (%s)", got.PlusPercent, got.MinusPercent, c.plus, c.minus, got.Source)
			}
			if got.Source == "" {
				t.Fatalf("every tolerance must say where it comes from")
			}
		})
	}
}

func TestMoneyAndQuantityRendering(t *testing.T) {
	cases := []struct {
		currency string
		value    float64
		decimals int
		want     string
	}{
		{"USD", 48600, 2, "USD 48600.00"},
		{"usd", 48600.555, 2, "USD 48600.56"},
		{"JPY", 48600, 0, "JPY 48600"},
		{"", 48600, 2, "48600.00"},
	}
	for _, c := range cases {
		if got := Money(c.currency, c.value, c.decimals); got != c.want {
			t.Errorf("Money(%q, %v, %d) = %q, want %q", c.currency, c.value, c.decimals, got, c.want)
		}
	}

	quantities := []struct {
		value float64
		unit  string
		want  string
	}{
		{1800, "sets", "1800 sets"},
		{1800.5, "kg", "1800.5 kg"},
		{1800.0004, "kg", "1800 kg"},
		{1800, "", "1800"},
	}
	for _, c := range quantities {
		if got := Quantity(c.value, c.unit); got != c.want {
			t.Errorf("Quantity(%v, %q) = %q, want %q", c.value, c.unit, got, c.want)
		}
	}
}

func TestPercentTextAndPlural(t *testing.T) {
	percents := map[float64]string{10: "10%", 110: "110%", 5: "5%", 2.5: "2.5%", 0: "0%"}
	for in, want := range percents {
		if got := percentText(in); got != want {
			t.Errorf("percentText(%v) = %q, want %q", in, got, want)
		}
	}

	plurals := []struct {
		n         int
		one, many string
		want      string
	}{
		{1, "day", "days", "1 day"},
		{2, "day", "days", "2 days"},
		{0, "day", "days", "0 days"},
		{-1, "day", "days", "-1 day"},
	}
	for _, c := range plurals {
		if got := plural(c.n, c.one, c.many); got != c.want {
			t.Errorf("plural(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestPercentOf(t *testing.T) {
	cases := []struct {
		part, whole, want float64
	}{
		{110, 100, 110},
		{5, 100, 5},
		{0, 100, 0},
		{50, 0, 0}, // nothing to divide by
	}
	for _, c := range cases {
		if got := PercentOf(c.part, c.whole); got != c.want {
			t.Errorf("PercentOf(%v, %v) = %v, want %v", c.part, c.whole, got, c.want)
		}
	}
}
