package ucp

import (
	"sort"
	"strings"
	"unicode"
)

// Match is how closely two pieces of text agree. The distinction between a
// typing error and a conflict of data is the single most consequential judgment
// this tool makes, because it decides whether the exporter is told to reprint a
// document or told to stop worrying.
//
// The rules behind it: data in a document need not be identical to, but must
// not conflict with, data in any other stipulated document or the credit (UCP
// 600 article 14(d)); the description of the goods in a commercial invoice must
// correspond with that in the credit (article 18(c)); in documents other than
// the invoice the description may be in general terms not conflicting with the
// credit (article 14(e)). The ICC's standard banking practice adds that a
// misspelling or typing error which does not affect the meaning of a word does
// not make a document discrepant, while a difference in a number or a model
// code does — see the comment on ConventionTypoEditDistance for how that
// practice is sourced here.
type Match int

const (
	// MatchExact — the two readings are the same text.
	MatchExact Match = iota
	// MatchEquivalent — the same once case, punctuation, spacing and the
	// ordinary abbreviations of a company name are set aside.
	MatchEquivalent
	// MatchGeneral — one is the other in general terms: every significant word
	// of the shorter appears in the longer, and no figure or code conflicts.
	// Acceptable in a document other than the invoice (article 14(e)).
	MatchGeneral
	// MatchTypo — within the tool's typing-error allowance, with no figure or
	// code differing. Reported for the exporter to decide on, never as
	// compliant.
	MatchTypo
	// MatchConflict — the data conflicts.
	MatchConflict
)

func (m Match) String() string {
	switch m {
	case MatchExact:
		return "identical"
	case MatchEquivalent:
		return "equivalent"
	case MatchGeneral:
		return "in general terms"
	case MatchTypo:
		return "probably a typing error"
	default:
		return "in conflict"
	}
}

// abbreviations are the expansions this tool treats as the same word. They are
// the forms that appear on Indian and Philippine company paperwork and in trade
// descriptions, where "Pvt Ltd" and "Private Limited" are the same company and
// no bank would refuse over the difference. The list is deliberately short:
// anything not on it is compared as written.
var abbreviations = map[string]string{
	"pvt":       "private",
	"pte":       "private",
	"ltd":       "limited",
	"ltda":      "limited",
	"plc":       "publiclimitedcompany",
	"llp":       "limitedliabilitypartnership",
	"co":        "company",
	"corp":      "corporation",
	"corpn":     "corporation",
	"inc":       "incorporated",
	"intl":      "international",
	"int":       "international",
	"mfg":       "manufacturing",
	"mfrs":      "manufacturers",
	"exp":       "exports",
	"ind":       "industries",
	"inds":      "industries",
	"ent":       "enterprises",
	"and":       "and",
	"&":         "and",
	"bros":      "brothers",
	"st":        "street",
	"rd":        "road",
	"opc":       "onepersoncompany",
	"sdn":       "sendirian",
	"bhd":       "berhad",
	"kgs":       "kg",
	"kilos":     "kg",
	"kilogram":  "kg",
	"kilograms": "kg",
	"mts":       "mt",
	"pcs":       "pieces",
	"pc":        "pieces",
	"nos":       "pieces",
	"ctns":      "cartons",
	"ctn":       "cartons",
	"pkgs":      "packages",
	"pkg":       "packages",
}

// noiseWords carry no meaning in a name or a description and are dropped before
// comparing. "The" and "M/s" are the usual offenders.
var noiseWords = map[string]bool{
	"the": true,
	"a":   true,
	"an":  true,
	"of":  true,
	"ms":  true, // M/s, the Indian honorific prefix on a firm's name
}

// Tokens breaks text into comparable words: lower case, punctuation removed,
// abbreviations expanded, noise dropped. A word carrying a digit is left as it
// is found, because a figure or a model code must be compared character for
// character.
func Tokens(s string) []string {
	raw := splitWords(s)
	out := make([]string, 0, len(raw))
	for _, w := range raw {
		if hasDigit(w) {
			out = append(out, w)
			continue
		}
		if full, ok := abbreviations[w]; ok {
			w = full
		}
		if noiseWords[w] {
			continue
		}
		out = append(out, w)
	}
	return out
}

// splitWords lower-cases and splits on anything that is not a letter or a
// digit. An ampersand is kept as its own word first, so that "A & B" and "A and
// B" come out the same, and a leading "M/s" — the honorific an Indian firm's
// name is written with — is dropped, which splitting alone would turn into two
// stray letters. Only a leading one is dropped, so that "m/s" inside a
// description of a motor survives.
func splitWords(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "&", " and ")
	for _, prefix := range []string{"m/s.", "m/s", "messrs."} {
		if strings.HasPrefix(s, prefix) {
			s = s[len(prefix):]
			break
		}
	}
	var out []string
	var cur strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
			continue
		}
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func hasDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// Normalised joins the comparable words with single spaces. Two strings with
// the same Normalised form are the same data however they were typed.
func Normalised(s string) string { return strings.Join(Tokens(s), " ") }

// codeTokens are the words that must agree character for character: anything
// containing a digit. A quantity, an invoice number, a model code, a container
// number and an HS code all land here, and a difference in any of them is a
// conflict and not a typing error — "model 123" against "model 321" is the
// ICC's own example of a difference that is not a typing error.
func codeTokens(s string) []string {
	var out []string
	for _, w := range Tokens(s) {
		if hasDigit(w) {
			out = append(out, w)
		}
	}
	sort.Strings(out)
	return out
}

// Compare decides how two readings of the same data agree. The first argument
// is the credit's wording and the second the document's, which matters only for
// MatchGeneral: a document may describe the goods in general terms, the credit
// may not be cut down.
func Compare(credit, document string, t Thresholds) Match {
	t = t.Normalise()
	c, d := strings.TrimSpace(credit), strings.TrimSpace(document)

	if c == d {
		return MatchExact
	}
	if c == "" || d == "" {
		return MatchConflict
	}

	nc, nd := Normalised(c), Normalised(d)
	if nc == nd {
		return MatchEquivalent
	}

	// A figure or a code that differs is a conflict, whatever else agrees. The
	// comparison is on the sets of coded words, so a document that repeats one
	// of them or lists them in another order still matches.
	cc, dc := codeTokens(c), codeTokens(d)
	if !sameStrings(cc, dc) {
		if subsetStrings(dc, cc) && len(dc) < len(cc) {
			// The document carries fewer figures than the credit but no
			// different ones — the general-terms case, settled below.
		} else {
			return MatchConflict
		}
	}

	// General terms: one reading is the other with words added or left off, and
	// nothing in either contradicts the other. "Tuticorin" against "Tuticorin,
	// India" is the same port; "Coimbatore Spindle Works" against "Coimbatore
	// Spindle Works Private Limited" is the same firm named at two lengths.
	ct, dt := Tokens(nc), Tokens(nd)
	if len(dt) > 0 && len(ct) > 0 && (subsetStrings(dt, ct) || subsetStrings(ct, dt)) {
		return MatchGeneral
	}

	// Last, a typing error: close enough overall, with the figures agreeing.
	if sameStrings(cc, dc) && editDistance(nc, nd) <= t.TypoEditDistance {
		return MatchTypo
	}
	return MatchConflict
}

// Clash is a contradiction between two free-text descriptions of the same
// goods. It exists separately from Match because the test the rules apply to a
// description is not the test they apply to a name or a port.
//
// A name and a port are single fields: two readings of one field either agree
// or they do not. A description of goods is prose, and article 14(e) lets a
// document other than the invoice describe the goods "in general terms not
// conflicting with their description in the credit" — so a description that
// says less, or that adds the number of cartons or the country of origin, does
// not conflict at all. Only a contradiction does.
type Clash string

const (
	// ClashNone — nothing in the two readings contradicts the other.
	ClashNone Clash = ""
	// ClashCode — the same field carries a different figure or code in each,
	// which is the difference standard practice never treats as a slip.
	ClashCode Clash = "code"
	// ClashSubject — the two describe different goods: they have no
	// significant word in common.
	ClashSubject Clash = "subject"
)

// Conflicts reports whether a document's description of the goods contradicts
// the credit's, and says how. It looks for two things only, because they are
// the two that can be told apart reliably from text:
//
//   - the same field carrying a different code — "model 4400" where the credit
//     says "model 4200", matched by the word the code sits beside and by the
//     code being the same length, so that "1800 sets" and "150 cartons" are
//     not mistaken for two readings of one figure;
//   - no significant word in common at all, which means the documents are
//     about different goods.
//
// Anything else it leaves alone, deliberately. An exporter told to reprint a
// certificate that was never discrepant loses a day for nothing, and this is
// the test where that mistake is easiest to make.
func Conflicts(credit, document string, _ Thresholds) (Clash, string) {
	ct, dt := Tokens(credit), Tokens(document)
	if len(ct) == 0 || len(dt) == 0 {
		return ClashNone, ""
	}

	creditCodes, documentCodes := codesByContext(ct), codesByContext(dt)
	for context, theirs := range documentCodes {
		ours, ok := creditCodes[context]
		if !ok {
			continue
		}
		for _, theirCode := range theirs {
			for _, ourCode := range ours {
				if len([]rune(theirCode)) != len([]rune(ourCode)) {
					continue // two different fields, not two readings of one
				}
				if theirCode == ourCode {
					break
				}
				return ClashCode, "the credit reads \"" + context + " " + ourCode +
					"\" and the document reads \"" + context + " " + theirCode + "\""
			}
		}
	}

	shared := false
	for _, w := range dt {
		if !hasDigit(w) && containsString(ct, w) {
			shared = true
			break
		}
	}
	if !shared {
		return ClashSubject, "the two have no word in common"
	}
	return ClashNone, ""
}

// codesByContext indexes every coded word by the plain word it sits beside —
// the one immediately before it, and the one immediately after. A code between
// two other codes gets no context and is left out, because there is nothing to
// line it up against on the other side.
func codesByContext(tokens []string) map[string][]string {
	out := map[string][]string{}
	add := func(context, code string) {
		if context == "" {
			return
		}
		for _, had := range out[context] {
			if had == code {
				return
			}
		}
		out[context] = append(out[context], code)
	}
	for i, w := range tokens {
		if !hasDigit(w) {
			continue
		}
		if i > 0 && !hasDigit(tokens[i-1]) {
			add(tokens[i-1], w)
		}
		if i+1 < len(tokens) && !hasDigit(tokens[i+1]) {
			add(tokens[i+1], w)
		}
	}
	return out
}

// DescriptionMatch is how a document's description of the goods stands against
// the credit's.
type DescriptionMatch int

const (
	// DescriptionCorresponds — the document says everything the credit says,
	// and perhaps more. This is what article 18(c) asks of the invoice.
	DescriptionCorresponds DescriptionMatch = iota
	// DescriptionGeneral — the document says less than the credit, but nothing
	// that conflicts. Article 14(e) allows this outside the invoice.
	DescriptionGeneral
	// DescriptionTypo — it would correspond but for what looks like a typing
	// error.
	DescriptionTypo
	// DescriptionConflict — it contradicts the credit.
	DescriptionConflict
)

// CompareDescription compares a document's description of the goods with the
// credit's and says what the difference amounts to, with a phrase naming it for
// a report line.
func CompareDescription(credit, document string, t Thresholds) (DescriptionMatch, string) {
	t = t.Normalise()
	ct, dt := Tokens(credit), Tokens(document)
	switch {
	case len(ct) == 0 && len(dt) == 0:
		return DescriptionCorresponds, ""
	case len(ct) == 0 || len(dt) == 0:
		return DescriptionConflict, "one of the two says nothing about the goods"
	}

	if clash, why := Conflicts(credit, document, t); clash != ClashNone {
		return DescriptionConflict, why
	}

	// Which of the credit's words the document does not repeat.
	var missing []string
	for _, w := range ct {
		if !containsString(dt, w) {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return DescriptionCorresponds, ""
	}

	// A word the document nearly repeats is a typing error; a figure it nearly
	// repeats never is.
	budget := t.TypoEditDistance
	typos := make([]string, 0, len(missing))
	for _, w := range missing {
		nearest, distance := nearestWord(w, dt)
		switch {
		case nearest == "" || distance > budget:
			return DescriptionGeneral, "the document does not carry: " + strings.Join(missing, ", ")
		case hasDigit(w) || hasDigit(nearest):
			return DescriptionConflict, "the credit reads \"" + w + "\" and the document reads \"" + nearest + "\""
		default:
			budget -= distance
			typos = append(typos, "\""+nearest+"\" for \""+w+"\"")
		}
	}
	return DescriptionTypo, strings.Join(typos, ", ")
}

// nearestWord returns the closest of the candidates to w, and the distance.
func nearestWord(w string, candidates []string) (string, int) {
	best, bestDistance := "", -1
	for _, c := range candidates {
		d := editDistance(w, c)
		if bestDistance < 0 || d < bestDistance {
			best, bestDistance = c, d
		}
	}
	if bestDistance < 0 {
		return "", 0
	}
	return best, bestDistance
}

// containsString reports whether a word appears in a slice.
func containsString(all []string, w string) bool {
	for _, c := range all {
		if c == w {
			return true
		}
	}
	return false
}

// sameStrings compares two sorted slices for equality.
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// subsetStrings reports whether every word of a appears in b, counting
// repeats once.
func subsetStrings(a, b []string) bool {
	have := make(map[string]bool, len(b))
	for _, w := range b {
		have[w] = true
	}
	for _, w := range a {
		if !have[w] {
			return false
		}
	}
	return true
}

// editDistance counts the fewest single-character insertions, deletions,
// substitutions or transpositions of two neighbouring characters that turn one
// string into the other — the optimal string alignment distance.
//
// Transpositions are counted as one edit rather than two on purpose: swapping
// two letters is a single slip of the fingers, and "modle" for "model" is one
// of the ICC's own examples of a typing error that does not make a document
// discrepant. Plain Levenshtein would charge it two and call it a conflict.
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}

	// Three rows: the row before last is what a transposition reaches back to.
	prev2 := make([]int, len(rb)+1)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			best := min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				if swap := prev2[j-2] + 1; swap < best {
					best = swap
				}
			}
			cur[j] = best
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// countryAliases lets the two markets these tools are built for be written the
// several ways they are written on real documents. It is not a table of the
// world: a country not on it is compared as written, and a difference is
// reported as something to confirm rather than as a discrepancy, which is the
// safe direction to be wrong in.
var countryAliases = map[string]string{
	"in":                       "india",
	"ind":                      "india",
	"india":                    "india",
	"republicofindia":          "india",
	"bharat":                   "india",
	"ph":                       "philippines",
	"phl":                      "philippines",
	"philippines":              "philippines",
	"republicofthephilippines": "philippines",
	"pilipinas":                "philippines",
}

// SameCountry compares two country names for article 14(j), which requires the
// addresses of the beneficiary and the applicant in a document to be within the
// same country as those in the credit, not to be the same address. It returns
// whether they match and whether the answer is certain; an unrecognised pair of
// names that differ is not certain.
func SameCountry(a, b string) (same bool, certain bool) {
	ka := strings.Join(splitWords(a), "")
	kb := strings.Join(splitWords(b), "")
	if ka == "" || kb == "" {
		return false, false
	}

	na, knownA := countryAliases[ka]
	if !knownA {
		na = ka
	}
	nb, knownB := countryAliases[kb]
	if !knownB {
		nb = kb
	}

	if na == nb {
		return true, true
	}
	// Two names this tool recognises and that disagree are certainly different
	// countries. Anything else is a difference the exporter has to read.
	return false, knownA && knownB
}

// ContainsAny reports whether text carries any of the given words, compared on
// normalised words so that punctuation and case do not matter. It is used to
// read the credit's own wording — "independent", "charter party", "blank
// endorsed" — rather than to compare two documents.
func ContainsAny(text string, words ...string) bool {
	have := map[string]bool{}
	for _, w := range Tokens(text) {
		have[w] = true
	}
	for _, phrase := range words {
		parts := Tokens(phrase)
		if len(parts) == 0 {
			continue
		}
		all := true
		for _, p := range parts {
			if !have[p] {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// Shorten trims a long piece of text for a report line, on a word boundary.
func Shorten(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) <= max || max <= 1 {
		return s
	}
	r := []rune(s)
	cut := max
	for cut > 0 && !unicode.IsSpace(r[cut]) {
		cut--
	}
	if cut == 0 {
		cut = max
	}
	return strings.TrimSpace(string(r[:cut])) + "…"
}
