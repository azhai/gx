package list

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azhai/gx/walker"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if c.TypeFilter != "a" {
		t.Errorf("TypeFilter = %q, want a", c.TypeFilter)
	}
	if c.Format != "path" {
		t.Errorf("Format = %q, want path", c.Format)
	}
	if c.MaxDepth != 0 {
		t.Errorf("MaxDepth = %d, want 0", c.MaxDepth)
	}
}

func TestLister_BasicList(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("world"), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	l, err := NewLister(c)
	if err != nil {
		t.Fatal(err)
	}
	code := l.Run()
	if code != 0 {
		t.Errorf("Run() = %d, want 0", code)
	}
}

func TestLister_NoResults(t *testing.T) {
	dir := t.TempDir()

	c := NewConfig()
	c.Paths = []string{dir}
	l, _ := NewLister(c)
	code := l.Run()
	if code != 1 {
		t.Errorf("Run() = %d, want 1 (no results)", code)
	}
}

func TestLister_GlobFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("world"), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.Glob = "*.go"
	l, err := NewLister(c)
	if err != nil {
		t.Fatal(err)
	}

	entries := collectEntries(l, c)
	if len(entries) != 1 {
		t.Errorf("expected 1 .go file, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].Name != "b.go" {
		t.Errorf("expected b.go, got %s", entries[0].Name)
	}
}

func TestLister_SizeFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "small.txt"), []byte("hi"), 0644)
	os.WriteFile(filepath.Join(dir, "big.txt"), make([]byte, 2048), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.SizeExpr = ">1K"
	l, err := NewLister(c)
	if err != nil {
		t.Fatal(err)
	}

	entries := collectEntries(l, c)
	if len(entries) != 1 {
		t.Errorf("expected 1 file >1K, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].Name != "big.txt" {
		t.Errorf("expected big.txt, got %s", entries[0].Name)
	}
}

func TestLister_MtimeFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "recent.txt"), []byte("recent"), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.MtimeExpr = "<=1h"
	l, err := NewLister(c)
	if err != nil {
		t.Fatal(err)
	}

	entries := collectEntries(l, c)
	if len(entries) != 1 {
		t.Errorf("expected 1 file modified within 1h, got %d", len(entries))
	}
}

func TestLister_TypeFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	c := NewConfig()
	c.Paths = []string{dir}
	c.TypeFilter = "d"
	l, err := NewLister(c)
	if err != nil {
		t.Fatal(err)
	}

	entries := collectEntries(l, c)
	for _, e := range entries {
		if !e.IsDir {
			t.Errorf("expected only directories, got file %s", e.Name)
		}
	}
}

func TestLister_SortByName(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("c"), 0644)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.SortField = "name"
	l, _ := NewLister(c)

	entries := collectEntries(l, c)
	l.sortEntries(entries)

	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].Name < entries[i-1].Name {
			t.Errorf("entries not sorted by name: %s before %s", entries[i-1].Name, entries[i].Name)
		}
	}
}

func TestLister_SortBySize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "small.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "big.txt"), make([]byte, 1000), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.SortField = "size"
	l, _ := NewLister(c)

	entries := collectEntries(l, c)
	l.sortEntries(entries)

	if len(entries) < 2 {
		t.Fatalf("expected at least 2 entries, got %d", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].Size < entries[i-1].Size {
			t.Errorf("entries not sorted by size: %d before %d", entries[i-1].Size, entries[i].Size)
		}
	}
}

func TestLister_ReverseSort(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bb"), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.SortField = "size"
	c.Reverse = true
	l, _ := NewLister(c)

	entries := collectEntries(l, c)
	l.sortEntries(entries)

	if len(entries) < 2 {
		t.Fatalf("expected at least 2 entries, got %d", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].Size > entries[i-1].Size {
			t.Errorf("entries not reverse-sorted by size: %d before %d", entries[i-1].Size, entries[i].Size)
		}
	}
}

func TestLister_FormatLong(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.Format = "long"
	l, err := NewLister(c)
	if err != nil {
		t.Fatal(err)
	}
	code := l.Run()
	if code != 0 {
		t.Errorf("Run() = %d, want 0", code)
	}
}

func TestLister_FormatName(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.Format = "name"
	l, err := NewLister(c)
	if err != nil {
		t.Fatal(err)
	}
	code := l.Run()
	if code != 0 {
		t.Errorf("Run() = %d, want 0", code)
	}
}

func TestLister_InvalidSizeExpr(t *testing.T) {
	c := NewConfig()
	c.SizeExpr = ">1X"
	_, err := NewLister(c)
	if err == nil {
		t.Error("expected error for invalid size expression")
	}
}

func TestLister_InvalidMtimeExpr(t *testing.T) {
	c := NewConfig()
	c.MtimeExpr = "<=1x"
	_, err := NewLister(c)
	if err == nil {
		t.Error("expected error for invalid mtime expression")
	}
}

func TestFormatMode(t *testing.T) {
	tests := []struct {
		name     string
		isDir    bool
		isLink   bool
		expected string
	}{
		{"regular file", false, false, "-rw-r--r--"},
		{"directory", true, false, "drwxr-xr-x"},
		{"symlink", false, true, "lrwxr-xr-x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mode os.FileMode
			if tt.isDir {
				mode = os.ModeDir | 0755
			} else {
				mode = 0644
			}
			if tt.isLink {
				mode = os.ModeSymlink | 0755
			}
			got := formatMode(mode, tt.isDir, tt.isLink)
			if got != tt.expected {
				t.Errorf("formatMode() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLister_CtimeFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.CtimeExpr = "<=1h"
	l, err := NewLister(c)
	if err != nil {
		t.Fatal(err)
	}

	entries := collectEntries(l, c)
	if len(entries) != 1 {
		t.Errorf("expected 1 file with ctime within 1h, got %d", len(entries))
	}
}

func TestLister_CombinedFilters(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "small.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "big.txt"), make([]byte, 2048), 0644)

	c := NewConfig()
	c.Paths = []string{dir}
	c.Glob = "*.go"
	c.SizeExpr = ">1K"
	l, err := NewLister(c)
	if err != nil {
		t.Fatal(err)
	}

	entries := collectEntries(l, c)
	if len(entries) != 0 {
		t.Errorf("expected 0 files matching *.go AND >1K, got %d", len(entries))
	}
}

func collectEntries(l *Lister, c *Config) []Entry {
	wc := walkerConfig(c)
	w := walker.New(wc)

	rootSet := make(map[string]bool)
	for _, p := range c.Paths {
		abs, _ := filepath.Abs(p)
		rootSet[abs] = true
	}

	var entries []Entry
	for fi := range w.Walk() {
		abs, _ := filepath.Abs(fi.Path)
		if rootSet[abs] {
			continue
		}
		e := l.toEntry(fi)
		if l.matchEntry(e) {
			entries = append(entries, e)
		}
	}
	return entries
}

func walkerConfig(c *Config) *walker.Config {
	wc := walker.NewConfig()
	wc.Paths = c.Paths
	wc.IncludeDir = c.TypeFilter == "d" || c.TypeFilter == "a"
	wc.IncludeSymlink = c.TypeFilter == "l" || c.TypeFilter == "a"
	wc.MaxDepth = c.MaxDepth
	if c.Glob != "" {
		wc.FilePattern = c.Glob
	}
	return wc
}

func TestGetChangeTime_Fallback(t *testing.T) {
	now := time.Now()
	result := getChangeTime("/nonexistent/path", now)
	if !result.Equal(now) {
		t.Errorf("expected fallback time, got %v", result)
	}
}
