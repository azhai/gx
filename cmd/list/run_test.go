package list

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/azhai/gx/args"
	"github.com/azhai/gx/pkg/processor"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if !c.Color || !c.DryRun {
		t.Error("defaults wrong")
	}
	if c.Workers != 1 {
		t.Errorf("Workers default = %d, want 1", c.Workers)
	}
}

func TestSearcher_ShortCircuitsOnFirstMatch(t *testing.T) {
	dir := t.TempDir()
	// File with match on line 50 — ProcessFile should stop after line 50,
	// not scan all 100 lines. We can't observe the scan count directly,
	// but we verify the result is exactly one entry per file.
	p := filepath.Join(dir, "a.txt")
	var content []byte
	for i := 0; i < 100; i++ {
		if i == 49 {
			content = append(content, []byte("TARGET line\n")...)
		} else {
			content = append(content, []byte("nomatch line\n")...)
		}
	}
	if err := os.WriteFile(p, content, 0644); err != nil {
		t.Fatal(err)
	}

	c := NewConfig()
	c.Pattern = "TARGET"
	c.Paths = []string{dir}
	s, err := NewSearcher(c)
	if err != nil {
		t.Fatalf("NewSearcher: %v", err)
	}

	results := s.ProcessFile(p)
	if len(results) != 1 {
		t.Errorf("expected 1 result (short-circuit), got %d", len(results))
	}
	if results[0].Path != p {
		t.Errorf("Path = %q, want %q", results[0].Path, p)
	}
}

func TestSearcher_NoMatch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("nothing here\n"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewConfig()
	c.Pattern = "TARGET"
	c.Paths = []string{dir}
	s, _ := NewSearcher(c)

	if got := s.ProcessFile(p); len(got) != 0 {
		t.Errorf("expected 0 results, got %d", len(got))
	}
}

func TestSearcher_SearchPrintsPathOnce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hit\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("hit\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.txt"), []byte("miss\n"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewConfig()
	c.Pattern = "hit"
	c.Paths = []string{dir}
	s, _ := NewSearcher(c)
	s.Search()

	if got := s.PrintResults(); got != 2 {
		t.Errorf("expected 2 files, got %d", got)
	}
}

func TestSearcher_ImplementsInterface(t *testing.T) {
	var _ processor.FileProcessor = (*Searcher)(nil)
}

func TestNewSearcher_InvalidRegex(t *testing.T) {
	c := &Config{CommonConfig: args.CommonConfig{Pattern: "[invalid"}}
	if _, err := NewSearcher(c); err == nil {
		t.Error("expected error for invalid regex")
	}
}
