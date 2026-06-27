package find

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/azhai/gx/args"
	"github.com/azhai/gx/pkg/processor"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if !c.ShowLineNum || !c.Color || !c.DryRun {
		t.Error("defaults wrong")
	}
	if c.Workers != 1 {
		t.Errorf("Workers default = %d, want 1", c.Workers)
	}
}

func TestSearcher_Search(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\nhello again"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("no match"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewConfig()
	c.Pattern = "hello"
	c.Paths = []string{dir}
	s, err := NewSearcher(c)
	if err != nil {
		t.Fatalf("NewSearcher: %v", err)
	}
	s.Search()

	count := 0
	for range s.results {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 matches, got %d", count)
	}
}

func TestSearcher_PrintResultsCount(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\nhello\nhello"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewConfig()
	c.Pattern = "hello"
	c.Paths = []string{dir}
	c.Color = false
	s, _ := NewSearcher(c)
	s.Search()

	if got := s.PrintResults(); got != 3 {
		t.Errorf("expected 3 results, got %d", got)
	}
}

func TestSearcher_ProcessFileImplementsInterface(t *testing.T) {
	var _ processor.FileProcessor = (*Searcher)(nil)
}

func TestNewSearcher_InvalidRegex(t *testing.T) {
	c := &Config{CommonConfig: args.CommonConfig{Pattern: "[invalid"}}
	if _, err := NewSearcher(c); err == nil {
		t.Error("expected error for invalid regex")
	}
}
