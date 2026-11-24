package ucp

import (
	"reflect"
	"testing"
)

func TestTokens(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"plain words", "ring frame spindles", []string{"ring", "frame", "spindles"}},
		{"case and punctuation go", "Ring-Frame, SPINDLES.", []string{"ring", "frame", "spindles"}},
		{"legal forms expand", "Coimbatore Spindle Works Pvt Ltd", []string{"coimbatore", "spindle", "works", "private", "limited"}},
		{"ampersand becomes and", "Patel & Sons", []string{"patel", "and", "sons"}},
		{"the honorific prefix goes", "M/s Patel Exports", []string{"patel", "exports"}},
		{"noise words go", "the parts of a machine", []string{"parts", "machine"}},
		{"a coded word is kept as written", "model RSA-4200", []string{"model", "rsa", "4200"}},
		{"units normalise", "1800 KGS", []string{"1800", "kg"}},
		{"messrs prefix goes too", "Messrs. Patel Exports", []string{"patel", "exports"}},
		{"m/s inside a description survives", "fan rated 12 m/s", []string{"fan", "rated", "12", "m", "s"}},
		{"empty", "   ", []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Tokens(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Tokens(%q) = %#v, want %#v", c.in, got, c.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	th := StatutoryThresholds()
	cases := []struct {
		name             string
		credit, document string
		want             Match
	}{
		{"identical", "Spindle assemblies", "Spindle assemblies", MatchExact},
		{"case only", "SPINDLE ASSEMBLIES", "Spindle assemblies", MatchEquivalent},
		{"punctuation and spacing only", "Spindle  assemblies,", "spindle assemblies", MatchEquivalent},
		{"legal form abbreviated", "Coimbatore Spindle Works Private Limited", "Coimbatore Spindle Works Pvt Ltd", MatchEquivalent},
		{"ampersand against and", "Patel & Sons Exports", "Patel and Sons Exports", MatchEquivalent},

		// The heart of it: a misspelling is not a conflict, a different number
		// is. This is the ICC's own example.
		{"a misspelling", "machine parts", "mashine parts", MatchTypo},
		{"two misspellings", "fountain pen model", "fountan pen modle", MatchTypo},
		{"a different model number", "model 123", "model 321", MatchConflict},
		{"a different model code", "spindle model RSA-4200", "spindle model RSA-4400", MatchConflict},
		{"a different quantity in the description", "1800 sets of spindles", "1600 sets of spindles", MatchConflict},

		{"general terms outside the invoice", "Ring frame spindle assemblies model RSA-4200 CIF Manila", "Spindle assemblies", MatchGeneral},
		{"general terms dropping the code", "Ring frame spindles model RSA-4200", "Ring frame spindles", MatchGeneral},
		{"more said than the credit says is not general", "Spindle assemblies", "Ring frame spindle assemblies model RSA-4200", MatchConflict},
		{"wholly different goods", "Spindle assemblies", "Cotton yarn 40s count", MatchConflict},
		{"an empty document description conflicts", "Spindle assemblies", "", MatchConflict},
		{"an empty credit description conflicts", "", "Spindle assemblies", MatchConflict},
		{"both empty are identical", "", "", MatchExact},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Compare(c.credit, c.document, th); got != c.want {
				t.Fatalf("Compare(%q, %q) = %v, want %v", c.credit, c.document, got, c.want)
			}
		})
	}
}

func TestCompareHonoursTheTypoAllowance(t *testing.T) {
	strict := Thresholds{TypoEditDistance: -1}.Normalise() // negative falls back to the default
	if strict.TypoEditDistance != ConventionTypoEditDistance {
		t.Fatalf("a negative allowance must fall back to the convention, got %d", strict.TypoEditDistance)
	}

	// With the allowance set to one edit, a two-character slip is a conflict.
	one := Thresholds{TypoEditDistance: 1}.Normalise()
	if got := Compare("fountain pen", "fountan pen", one); got != MatchTypo {
		t.Errorf("one edit with an allowance of one = %v, want a typing error", got)
	}
	if got := Compare("fountain pen model", "fountan pen modle", one); got != MatchConflict {
		t.Errorf("two edits with an allowance of one = %v, want a conflict", got)
	}

	// Widening the allowance never lets a different figure through.
	wide := Thresholds{TypoEditDistance: 20}.Normalise()
	if got := Compare("model 123", "model 321", wide); got != MatchConflict {
		t.Errorf("a different figure with a wide allowance = %v, want a conflict always", got)
	}
}

func TestMatchString(t *testing.T) {
	cases := map[Match]string{
		MatchExact:      "identical",
		MatchEquivalent: "equivalent",
		MatchGeneral:    "in general terms",
		MatchTypo:       "probably a typing error",
		MatchConflict:   "in conflict",
	}
	for m, want := range cases {
		if got := m.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", m, got, want)
		}
	}
}

func TestEditDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"machine", "mashine", 1},
		{"modle", "model", 1}, // a transposition is one slip, not two
		{"kitten", "sitting", 3},
		{"abcd", "badc", 2}, // two separate transpositions
		// The restricted form of the distance never edits the same stretch
		// twice, so "ca" to "abc" costs 3 rather than the 2 the unrestricted
		// form would charge. That is the conservative direction: it makes the
		// tool call something a conflict rather than wave it through as a slip.
		{"ca", "abc", 3},
	}
	for _, c := range cases {
		if got := editDistance(c.a, c.b); got != c.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestSameCountry(t *testing.T) {
	cases := []struct {
		name          string
		a, b          string
		same, certain bool
	}{
		{"identical", "India", "India", true, true},
		{"case and spacing", "  INDIA ", "india", true, true},
		{"code against name", "IN", "India", true, true},
		{"three-letter code", "IND", "Republic of India", true, true},
		{"the other market", "PH", "Philippines", true, true},
		{"two known countries that differ", "India", "Philippines", false, true},
		{"one unknown country matching by name", "Viet Nam", "Viet Nam", true, true},
		{"two unknown countries that differ", "Viet Nam", "Thailand", false, false},
		{"one side empty", "", "India", false, false},
		{"both empty", "", "", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			same, certain := SameCountry(c.a, c.b)
			if same != c.same || certain != c.certain {
				t.Fatalf("SameCountry(%q, %q) = (%v, %v), want (%v, %v)", c.a, c.b, same, certain, c.same, c.certain)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	wording := "Certificate of origin issued by an independent Chamber of Commerce"
	cases := []struct {
		name  string
		words []string
		want  bool
	}{
		{"a single word present", []string{"independent"}, true},
		{"a word absent", []string{"first class"}, false},
		{"a phrase present", []string{"chamber of commerce"}, true},
		{"a phrase with only one word present", []string{"independent surveyor"}, false},
		{"one of several", []string{"first class", "independent"}, true},
		{"nothing to look for", nil, false},
		{"an empty phrase is skipped", []string{""}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ContainsAny(wording, c.words...); got != c.want {
				t.Fatalf("ContainsAny(%v) = %v, want %v", c.words, got, c.want)
			}
		})
	}
}

func TestShorten(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"shorter than the limit", "spindle assemblies", 40, "spindle assemblies"},
		{"whitespace collapses", "spindle    assemblies", 40, "spindle assemblies"},
		{"cut on a word boundary", "ring frame spindle assemblies model RSA-4200", 20, "ring frame spindle…"},
		{"a single long word is cut where it must be", "aaaaaaaaaaaaaaaaaaaaaaaa", 10, "aaaaaaaaaa…"},
		{"a limit of one returns the whole string", "spindle assemblies", 1, "spindle assemblies"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Shorten(c.in, c.max); got != c.want {
				t.Fatalf("Shorten(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
			}
		})
	}
}

func TestNormalisedIsIdempotent(t *testing.T) {
	for _, s := range []string{
		"Coimbatore Spindle Works Pvt Ltd",
		"M/s Patel & Sons, the Exporters",
		"model RSA-4200, 1800 KGS",
	} {
		once := Normalised(s)
		if twice := Normalised(once); twice != once {
			t.Errorf("Normalised(%q) = %q, but normalising again gave %q", s, once, twice)
		}
	}
}
