package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreenivas-sadhu-prabhakara/lc-discrepancy-pre-checker-ucp/internal/ucp"
)

// sampleCase is the worked example wrapped as a saveable case.
func sampleCase(name string) Case {
	credit, presentation := ucp.Sample()
	return Case{Name: name, Credit: credit, Presentation: presentation}
}

func tempStore(t *testing.T) (*store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cases.json")
	s, err := openStore(path)
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	return s, path
}

func TestOpenStoreOnNothing(t *testing.T) {
	s, path := tempStore(t)

	if got := len(s.snapshot().Cases); got != 0 {
		t.Errorf("a new store holds %d cases, want none", got)
	}
	if got := s.thresholds().PresentationPeriodDays; got != ucp.ArtPresentationPeriodDays {
		t.Errorf("a new store applies %d days, want the %d of art. 14(c)", got, ucp.ArtPresentationPeriodDays)
	}
	// Nothing is written until something is saved.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the file was created before anything was saved")
	}
}

func TestSaveAndReopen(t *testing.T) {
	s, path := tempStore(t)

	saved, err := s.saveCase(sampleCase("Manila spindles"))
	if err != nil {
		t.Fatalf("saveCase: %v", err)
	}
	if saved.ID == "" {
		t.Fatalf("a saved case must be given an identifier")
	}
	if saved.SavedOn.Zero() {
		t.Fatalf("a saved case must be dated")
	}

	// The file must be readable only by its owner: it holds a customer's name,
	// prices and bank details.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("the data file is mode %o, want 600", mode)
	}

	// And it must be plain, indented JSON a person can read.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(raw), "\n  \"cases\": [") {
		t.Errorf("the file is not indented JSON:\n%s", raw)
	}

	reopened, err := openStore(path)
	if err != nil {
		t.Fatalf("reopening: %v", err)
	}
	back := reopened.snapshot()
	if len(back.Cases) != 1 || back.Cases[0].ID != saved.ID {
		t.Fatalf("the case did not survive a reopen: %+v", back.Cases)
	}
	if back.Cases[0].Credit.ExpiryDate != saved.Credit.ExpiryDate {
		t.Errorf("the expiry date did not survive: %v then %v",
			saved.Credit.ExpiryDate, back.Cases[0].Credit.ExpiryDate)
	}
	if got := len(back.Cases[0].Presentation.Documents); got != len(saved.Presentation.Documents) {
		t.Errorf("%d documents came back, want %d", got, len(saved.Presentation.Documents))
	}
}

func TestSaveReplacesByIdentifier(t *testing.T) {
	s, _ := tempStore(t)

	first, err := s.saveCase(sampleCase("Manila spindles"))
	if err != nil {
		t.Fatalf("saveCase: %v", err)
	}

	// Fixing the invoice and saving again must replace the case, not add one.
	fixed := first
	fixed.Credit.LatestShipmentDate = ucp.MustDate("2026-03-20")
	again, err := s.saveCase(fixed)
	if err != nil {
		t.Fatalf("saving again: %v", err)
	}
	if again.ID != first.ID {
		t.Errorf("the identifier changed on a replace: %q then %q", first.ID, again.ID)
	}
	if got := len(s.snapshot().Cases); got != 1 {
		t.Errorf("%d cases saved, want the one replaced", got)
	}
	if got, _ := s.getCase(first.ID); got.Credit.LatestShipmentDate != ucp.MustDate("2026-03-20") {
		t.Errorf("the change was not kept: %v", got.Credit.LatestShipmentDate)
	}
}

func TestIdentifiersAreReadableAndUnique(t *testing.T) {
	s, _ := tempStore(t)

	a, err := s.saveCase(sampleCase("first"))
	if err != nil {
		t.Fatalf("saveCase: %v", err)
	}
	if a.ID != "ilc-2026-004471" {
		t.Errorf("identifier = %q, want it taken from the credit number", a.ID)
	}

	b, err := s.saveCase(sampleCase("second"))
	if err != nil {
		t.Fatalf("saveCase: %v", err)
	}
	if b.ID == a.ID {
		t.Fatalf("two cases were given the same identifier %q", a.ID)
	}
	if b.ID != "ilc-2026-004471-2" {
		t.Errorf("the second identifier = %q, want it numbered", b.ID)
	}
}

func TestNamesAndIdentifiersFallBack(t *testing.T) {
	s, _ := tempStore(t)

	cases := []struct {
		name     string
		in       Case
		wantName string
		wantID   string
	}{
		{"no name takes the credit number",
			Case{Credit: ucp.Credit{Reference: "LC/99"}}, "LC/99", "lc-99"},
		{"no credit number takes the applicant",
			Case{Credit: ucp.Credit{Applicant: ucp.Party{Name: "Sampaguita Mills"}}},
			"Sampaguita Mills", "sampaguita-mills"},
		{"nothing at all still saves",
			Case{}, "Unnamed credit", "unnamed-credit"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := s.saveCase(c.in)
			if err != nil {
				t.Fatalf("saveCase: %v", err)
			}
			if got.Name != c.wantName {
				t.Errorf("name = %q, want %q", got.Name, c.wantName)
			}
			if got.ID != c.wantID {
				t.Errorf("id = %q, want %q", got.ID, c.wantID)
			}
		})
	}
}

func TestSaveRefusesWhatCannotBeChecked(t *testing.T) {
	s, _ := tempStore(t)

	cases := []struct {
		name    string
		in      Case
		wantErr string
	}{
		{"a name longer than the field", Case{Name: strings.Repeat("x", 200)}, "longer than 120"},
		{"too many documents", func() Case {
			c := Case{Name: "wide"}
			for i := 0; i < 41; i++ {
				c.Presentation.Documents = append(c.Presentation.Documents,
					ucp.Document{Kind: ucp.KindOther, Title: "a document"})
			}
			return c
		}(), "more than this tool handles"},
		{"a credit that expires before it is issued", Case{
			Name: "backwards",
			Credit: ucp.Credit{
				IssueDate:  ucp.MustDate("2026-03-10"),
				ExpiryDate: ucp.MustDate("2026-03-01"),
			},
		}, "before it was issued"},
		{"a currency that is not a code", Case{Name: "odd", Credit: ucp.Credit{Currency: "DOLLARS"}}, "three-letter"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := s.saveCase(c.in); err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("saveCase = %v, want an error about %q", err, c.wantErr)
			}
		})
	}
	if got := len(s.snapshot().Cases); got != 0 {
		t.Errorf("%d cases were saved despite being refused", got)
	}
}

func TestDeleteCase(t *testing.T) {
	s, _ := tempStore(t)

	saved, err := s.saveCase(sampleCase("Manila spindles"))
	if err != nil {
		t.Fatalf("saveCase: %v", err)
	}
	if err := s.deleteCase(saved.ID); err != nil {
		t.Fatalf("deleteCase: %v", err)
	}
	if got := len(s.snapshot().Cases); got != 0 {
		t.Errorf("%d cases left after a delete", got)
	}
	if err := s.deleteCase(saved.ID); err == nil {
		t.Errorf("deleting the same case twice must fail")
	}
	if err := s.deleteCase(""); err == nil {
		t.Errorf("deleting nothing must fail")
	}
}

func TestPutThresholds(t *testing.T) {
	s, path := tempStore(t)

	own := ucp.StatutoryThresholds()
	own.PresentationPeriodDays = 10
	saved, err := s.putThresholds(own)
	if err != nil {
		t.Fatalf("putThresholds: %v", err)
	}
	if saved.PresentationPeriodDays != 10 || !saved.Edited {
		t.Fatalf("the figure was not kept: %+v", saved)
	}

	reopened, err := openStore(path)
	if err != nil {
		t.Fatalf("reopening: %v", err)
	}
	if got := reopened.thresholds().PresentationPeriodDays; got != 10 {
		t.Errorf("the figure did not survive a reopen: %d", got)
	}

	// A blank set comes back as the figures UCP 600 fixes.
	back, err := s.putThresholds(ucp.Thresholds{})
	if err != nil {
		t.Fatalf("putThresholds: %v", err)
	}
	if back.PresentationPeriodDays != ucp.ArtPresentationPeriodDays || back.Edited {
		t.Errorf("a blank set did not return to the statute: %+v", back)
	}
}

func TestOpenStoreRefusesWhatItCannotRead(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"not JSON at all", "this is not a data file", "reading"},
		{"a newer format", `{"version": 99, "cases": []}`, "newer version"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(dir, idSafe(c.name)+".json")
			if err := os.WriteFile(path, []byte(c.body), 0o600); err != nil {
				t.Fatalf("writing the file: %v", err)
			}
			if _, err := openStore(path); err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("openStore = %v, want an error about %q", err, c.wantErr)
			}
		})
	}

	t.Run("an empty file is an empty store", func(t *testing.T) {
		path := filepath.Join(dir, "empty.json")
		if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
			t.Fatalf("writing: %v", err)
		}
		s, err := openStore(path)
		if err != nil {
			t.Fatalf("openStore: %v", err)
		}
		if got := len(s.snapshot().Cases); got != 0 {
			t.Errorf("%d cases read out of an empty file", got)
		}
	})

	t.Run("an older format is read and upgraded", func(t *testing.T) {
		path := filepath.Join(dir, "older.json")
		body := `{"version": 0, "cases": [{"id":"old","name":"an older case","credit":{"reference":"LC/1"}}]}`
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("writing: %v", err)
		}
		s, err := openStore(path)
		if err != nil {
			t.Fatalf("openStore: %v", err)
		}
		data := s.snapshot()
		if len(data.Cases) != 1 || data.Cases[0].Name != "an older case" {
			t.Fatalf("the older case was not read: %+v", data.Cases)
		}
		if data.Version != fileFormatVersion {
			t.Errorf("version = %d, want it upgraded to %d", data.Version, fileFormatVersion)
		}
	})
}

func TestSaveIsAtomicAndDoesNotLeaveRubbishBehind(t *testing.T) {
	s, path := tempStore(t)
	if _, err := s.saveCase(sampleCase("Manila spindles")); err != nil {
		t.Fatalf("saveCase: %v", err)
	}
	if _, err := s.saveCase(sampleCase("another")); err != nil {
		t.Fatalf("saveCase: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("reading the directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Errorf("a temporary file was left behind: %s", entry.Name())
		}
	}

	// What is on disk must parse, which is the whole point of the rename.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var back fileData
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("what was written does not parse: %v", err)
	}
	if len(back.Cases) != 2 {
		t.Errorf("%d cases on disk, want 2", len(back.Cases))
	}
}

func TestSortCasesAndFind(t *testing.T) {
	all := []Case{
		{ID: "b", Name: "beta", SavedOn: ucp.MustDate("2026-03-01")},
		{ID: "c", Name: "alpha", SavedOn: ucp.MustDate("2026-03-10")},
		{ID: "a", Name: "zeta", SavedOn: ucp.MustDate("2026-03-10")},
	}
	sortCases(all)
	if all[0].Name != "alpha" || all[1].Name != "zeta" || all[2].Name != "beta" {
		t.Fatalf("sorted wrongly: %q, %q, %q", all[0].Name, all[1].Name, all[2].Name)
	}
	if got := findCase(all, "b"); got != 2 {
		t.Errorf("findCase(b) = %d, want 2", got)
	}
	if got := findCase(all, "nothing"); got != -1 {
		t.Errorf("findCase of a missing case = %d, want -1", got)
	}
	if got := findCase(all, "  "); got != -1 {
		t.Errorf("findCase of nothing = %d, want -1", got)
	}
}

func TestIdSafe(t *testing.T) {
	cases := map[string]string{
		"ILC/2026/004471":       "ilc-2026-004471",
		"  spaces  here ":       "spaces-here",
		"Already-Safe":          "already-safe",
		"!!!":                   "",
		"":                      "",
		"a/b//c":                "a-b-c",
		strings.Repeat("x", 80): strings.Repeat("x", 80),
	}
	for in, want := range cases {
		if got := idSafe(in); got != want {
			t.Errorf("idSafe(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMaxCases(t *testing.T) {
	s, _ := tempStore(t)

	// Fill the store to the limit without writing the file five hundred times.
	for i := 0; i < maxCases; i++ {
		s.data.Cases = append(s.data.Cases, Case{ID: "case-" + itoa(i), Name: "case"}.normalise())
	}

	// A new case is refused, and says why.
	if _, err := s.saveCase(Case{Name: "one too many", Credit: ucp.Credit{Reference: "LC/over"}}); err == nil ||
		!strings.Contains(err.Error(), "which is the limit") {
		t.Fatalf("saveCase = %v, want the limit of %d enforced", err, maxCases)
	}

	// Replacing one that is already there still works, because it adds nothing.
	if _, err := s.saveCase(Case{ID: "case-1", Name: "replaced"}); err != nil {
		t.Fatalf("replacing an existing case at the limit: %v", err)
	}
	if got, _ := s.getCase("case-1"); got.Name != "replaced" {
		t.Errorf("the replacement did not take: %q", got.Name)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
