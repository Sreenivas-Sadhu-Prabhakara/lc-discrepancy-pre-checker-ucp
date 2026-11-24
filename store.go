package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/sreenivas-sadhu-prabhakara/lc-discrepancy-pre-checker-ucp/internal/ucp"
)

// fileFormatVersion is written into the data file so a later version of the
// tool can tell what it is reading.
const fileFormatVersion = 1

// maxCases caps how many cases the file will hold, so that a runaway script
// cannot fill a disk. A busy exporter has a few dozen live credits.
const maxCases = 500

// Case is one credit and the presentation being prepared against it, saved so
// that it can be checked again after the documents have been corrected.
type Case struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Credit       ucp.Credit       `json:"credit"`
	Presentation ucp.Presentation `json:"presentation"`
	SavedOn      ucp.Date         `json:"savedOn"`
}

// normalise tidies a case and gives it a name if it has none.
func (c Case) normalise() Case {
	c.ID = strings.TrimSpace(c.ID)
	c.Name = strings.Join(strings.Fields(c.Name), " ")
	c.Credit = c.Credit.Normalise()
	c.Presentation = c.Presentation.Normalise()
	if c.Name == "" {
		switch {
		case c.Credit.Reference != "":
			c.Name = c.Credit.Reference
		case c.Credit.Applicant.Name != "":
			c.Name = c.Credit.Applicant.Name
		default:
			c.Name = "Unnamed credit"
		}
	}
	if c.SavedOn.Zero() {
		c.SavedOn = ucp.Today()
	}
	return c
}

// validate refuses a case that cannot be saved.
func (c Case) validate() error {
	if len([]rune(c.Name)) > 120 {
		return fmt.Errorf("the name of a case is longer than 120 characters")
	}
	if len(c.Presentation.Documents) > 40 {
		return fmt.Errorf("a presentation of %d documents is more than this tool handles; 40 is the limit",
			len(c.Presentation.Documents))
	}
	if err := c.Credit.Validate(); err != nil {
		return err
	}
	return c.Presentation.Validate()
}

// fileData is the whole contents of the data file.
type fileData struct {
	Version    int            `json:"version"`
	Thresholds ucp.Thresholds `json:"thresholds"`
	Cases      []Case         `json:"cases"`
}

// store keeps the cases in memory and writes the whole file out after every
// change. A few hundred cases is a few hundred kilobytes, so rewriting the file
// is both simpler and safer than updating it in place.
type store struct {
	mu   sync.RWMutex
	path string
	data fileData
}

// openStore reads the data file, or starts empty if there is no file yet. It
// does not create the file until something is saved.
func openStore(path string) (*store, error) {
	s := &store{
		path: path,
		data: fileData{Version: fileFormatVersion, Thresholds: ucp.StatutoryThresholds()},
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return s, nil
	}

	var loaded fileData
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return nil, fmt.Errorf("reading %s: %w (move the file aside to start again)", path, err)
	}
	if loaded.Version > fileFormatVersion {
		return nil, fmt.Errorf("%s was written by a newer version of this tool (format %d, this build reads %d)",
			path, loaded.Version, fileFormatVersion)
	}
	loaded.Version = fileFormatVersion
	loaded.Thresholds = loaded.Thresholds.Normalise()
	for i, c := range loaded.Cases {
		loaded.Cases[i] = c.normalise()
	}
	s.data = loaded
	return s, nil
}

// snapshot is a copy of everything saved, which a reader can use without
// holding the lock.
func (s *store) snapshot() fileData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.data
	out.Cases = append([]Case(nil), s.data.Cases...)
	return out
}

// thresholds returns the figures in force.
func (s *store) thresholds() ucp.Thresholds {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Thresholds
}

// change applies a change and saves the file. If saving fails the change is
// rolled back, so what is in memory always matches what is on disk.
func (s *store) change(apply func(d *fileData) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := append([]Case(nil), s.data.Cases...)
	beforeThresholds := s.data.Thresholds

	if err := apply(&s.data); err != nil {
		s.data.Cases, s.data.Thresholds = before, beforeThresholds
		return err
	}
	sortCases(s.data.Cases)
	if err := s.save(); err != nil {
		s.data.Cases, s.data.Thresholds = before, beforeThresholds
		return err
	}
	return nil
}

// save writes everything to the data file atomically: a temporary file in the
// same directory, flushed to disk, then renamed over the old one. A crash
// therefore leaves either the old file or the new one, never half of either.
func (s *store) save() error {
	s.data.Version = fileFormatVersion

	body, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("preparing the file: %w", err)
	}
	body = append(body, '\n')

	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(s.path)+".tmp*")
	if err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // a no-op once the rename has succeeded

	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replacing %s: %w", s.path, err)
	}
	return nil
}

// sortCases keeps the file in a stable order: the most recently saved first,
// and within a day by name, so the file and every list read the same way every
// time.
func sortCases(all []Case) {
	sort.SliceStable(all, func(i, j int) bool {
		if !all[i].SavedOn.Equal(all[j].SavedOn) {
			return all[i].SavedOn.After(all[j].SavedOn)
		}
		return all[i].Name < all[j].Name
	})
}

// findCase returns the position of a case by identifier, or -1.
func findCase(all []Case, id string) int {
	id = strings.TrimSpace(id)
	if id == "" {
		return -1
	}
	for i, c := range all {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// nextID gives a new case an identifier the user can read: the credit's
// reference where there is one, made unique.
func nextID(all []Case, c Case) string {
	base := strings.TrimSpace(c.Credit.Reference)
	if base == "" {
		base = c.Name
	}
	base = idSafe(base)
	if base == "" {
		base = "case"
	}
	id := base
	for n := 2; findCase(all, id) >= 0; n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

// idSafe reduces a reference to letters, digits and dashes, so that an
// identifier is safe in a URL and readable in the file.
func idSafe(s string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() < 48 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// saveCase adds a case or replaces one that is already there, and returns it as
// saved.
func (s *store) saveCase(c Case) (Case, error) {
	c = c.normalise()
	if err := c.validate(); err != nil {
		return c, err
	}
	err := s.change(func(d *fileData) error {
		if i := findCase(d.Cases, c.ID); i >= 0 {
			d.Cases[i] = c
			return nil
		}
		if len(d.Cases) >= maxCases {
			return fmt.Errorf("there are already %d cases saved, which is the limit — remove one first", maxCases)
		}
		c.ID = nextID(d.Cases, c)
		d.Cases = append(d.Cases, c)
		return nil
	})
	return c, err
}

// deleteCase removes a case.
func (s *store) deleteCase(id string) error {
	return s.change(func(d *fileData) error {
		i := findCase(d.Cases, id)
		if i < 0 {
			return fmt.Errorf("there is no case %q saved", id)
		}
		d.Cases = append(d.Cases[:i:i], d.Cases[i+1:]...)
		return nil
	})
}

// getCase returns one saved case.
func (s *store) getCase(id string) (Case, bool) {
	data := s.snapshot()
	if i := findCase(data.Cases, id); i >= 0 {
		return data.Cases[i], true
	}
	return Case{}, false
}

// putThresholds replaces the figures in force.
func (s *store) putThresholds(next ucp.Thresholds) (ucp.Thresholds, error) {
	next = next.Normalise()
	err := s.change(func(d *fileData) error {
		d.Thresholds = next
		return nil
	})
	return next, err
}
