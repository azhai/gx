package replace

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/azhai/gx/args"
)

func createTestDir(t *testing.T, name string) string {
	dir := filepath.Join(os.TempDir(), name)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to remove test dir: %v", err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	return dir
}

func createTestFile(t *testing.T, dir, name, content string) {
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create parent dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}
	if !config.ShowLineNum {
		t.Error("ShowLineNum should be true by default")
	}
	if !config.Color {
		t.Error("Color should be true by default")
	}
	if config.Workers <= 0 {
		t.Error("Workers should be positive")
	}
	if !config.DryRun {
		t.Error("DryRun should be true by default")
	}
}

func TestNewSearcher(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid pattern",
			config:  &Config{CommonConfig: args.CommonConfig{Pattern: "hello"}},
			wantErr: false,
		},
		{
			name:    "invalid regex pattern",
			config:  &Config{CommonConfig: args.CommonConfig{Pattern: "[invalid"}},
			wantErr: true,
		},
		{
			name:    "empty pattern",
			config:  &Config{CommonConfig: args.CommonConfig{Pattern: ""}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher, err := NewSearcher(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSearcher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && searcher == nil {
				t.Error("NewSearcher() returned nil searcher")
			}
		})
	}
}

func TestSearcher_Search(t *testing.T) {
	dir := createTestDir(t, "replace_search_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test1.txt", "hello world\nhello universe")
	createTestFile(t, dir, "test2.txt", "no match here")
	createTestFile(t, dir, "test3.go", "hello golang")

	config := NewConfig()
	config.Pattern = "hello"
	config.Paths = []string{dir}

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	var matches []Match
	for match := range searcher.results {
		matches = append(matches, match)
	}

	if len(matches) < 3 {
		t.Errorf("Expected at least 3 matches, got %d", len(matches))
	}
}

func TestSearcher_SearchWithGlob(t *testing.T) {
	dir := createTestDir(t, "replace_search_glob_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test1.txt", "hello world")
	createTestFile(t, dir, "test2.go", "hello golang")

	config := NewConfig()
	config.Pattern = "hello"
	config.Paths = []string{dir}
	config.FilePattern = "*.go"

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	var matches []Match
	for match := range searcher.results {
		matches = append(matches, match)
	}

	if len(matches) != 1 {
		t.Errorf("Expected 1 match in go files, got %d", len(matches))
	}
}

func TestSearcher_SearchIgnoreCase(t *testing.T) {
	dir := createTestDir(t, "replace_search_ignore_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test.txt", "HELLO World\nhello universe\nHeLLo")

	config := NewConfig()
	config.Pattern = "hello"
	config.Paths = []string{dir}
	config.IgnoreCase = true

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	var matches []Match
	for match := range searcher.results {
		matches = append(matches, match)
	}

	if len(matches) != 3 {
		t.Errorf("Expected 3 matches with ignore case, got %d", len(matches))
	}
}

func TestSearcher_SearchFixedString(t *testing.T) {
	dir := createTestDir(t, "replace_search_fixed_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test.txt", "[test] is here\nbut test without brackets")

	config := NewConfig()
	config.Pattern = "[test]"
	config.Paths = []string{dir}
	config.FixedString = true

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	var matches []Match
	for match := range searcher.results {
		matches = append(matches, match)
	}

	if len(matches) != 1 {
		t.Errorf("Expected 1 match with fixed string, got %d", len(matches))
	}
}

func TestSearcher_Replace(t *testing.T) {
	dir := createTestDir(t, "replace_replace_test")
	defer os.RemoveAll(dir)

	testFile := filepath.Join(dir, "test.txt")
	createTestFile(t, dir, "test.txt", "hello world\nhello universe")

	config := NewConfig()
	config.Pattern = "hello"
	config.Replace = "hi"
	config.ReplaceSet = true
	config.Paths = []string{dir}
	config.DryRun = false

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Replace()

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := "hi world\nhi universe"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}

func TestSearcher_ReplaceDryRun(t *testing.T) {
	dir := createTestDir(t, "replace_dryrun_test")
	defer os.RemoveAll(dir)

	testFile := filepath.Join(dir, "test.txt")
	originalContent := "hello world"
	createTestFile(t, dir, "test.txt", originalContent)

	config := NewConfig()
	config.Pattern = "hello"
	config.Replace = "hi"
	config.ReplaceSet = true
	config.Paths = []string{dir}
	config.DryRun = true

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Replace()

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != originalContent {
		t.Error("Dry-run should not modify files")
	}
}

func TestSearcher_ReplaceWithGroups(t *testing.T) {
	dir := createTestDir(t, "replace_groups_test")
	defer os.RemoveAll(dir)

	testFile := filepath.Join(dir, "test.txt")
	createTestFile(t, dir, "test.txt", "user@example.com")

	config := NewConfig()
	config.Pattern = `(\w+)@(\w+)\.com`
	config.Replace = "$1 at $2"
	config.ReplaceSet = true
	config.Paths = []string{dir}
	config.DryRun = false

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Replace()

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := "user at example"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}

func TestSearcher_MultipleFiles(t *testing.T) {
	dir := createTestDir(t, "replace_multi_test")
	defer os.RemoveAll(dir)

	for i := range 10 {
		createTestFile(t, dir, filepath.Join("subdir", fmt.Sprintf("file%d.txt", i)), "hello world")
	}

	config := NewConfig()
	config.Pattern = "hello"
	config.Paths = []string{dir}

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	var matches []Match
	for match := range searcher.results {
		matches = append(matches, match)
	}

	if len(matches) != 10 {
		t.Errorf("Expected 10 matches, got %d", len(matches))
	}
}

func TestSearcher_NoMatch(t *testing.T) {
	dir := createTestDir(t, "replace_nomatch_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test.txt", "hello world")

	config := NewConfig()
	config.Pattern = "notfound"
	config.Paths = []string{dir}

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	var matches []Match
	for match := range searcher.results {
		matches = append(matches, match)
	}

	if len(matches) != 0 {
		t.Errorf("Expected 0 matches, got %d", len(matches))
	}
}

func TestSearcher_Workers(t *testing.T) {
	dir := createTestDir(t, "replace_workers_test")
	defer os.RemoveAll(dir)

	for i := range 20 {
		createTestFile(t, dir, filepath.Join("subdir", fmt.Sprintf("file%d.txt", i)), "hello world")
	}

	config := NewConfig()
	config.Pattern = "hello"
	config.Paths = []string{dir}
	config.Workers = 4

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	var matches []Match
	for match := range searcher.results {
		matches = append(matches, match)
	}

	if len(matches) != 20 {
		t.Errorf("Expected 20 matches, got %d", len(matches))
	}
}

func TestConfig_ParseArgs(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedPattern string
		expectedReplace string
		expectedPaths   []string
	}{
		{
			name:            "simple pattern",
			args:            []string{"hello"},
			expectedPattern: "hello",
			expectedPaths:   []string{"."},
		},
		{
			name:            "pattern and path",
			args:            []string{"hello", "/src"},
			expectedPattern: "hello",
			expectedPaths:   []string{"/src"},
		},
		{
			name:            "pattern, replace and path",
			args:            []string{"hello", "hi", "/src"},
			expectedPattern: "hello",
			expectedReplace: "hi",
			expectedPaths:   []string{"/src"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig()
			origArgs := os.Args
			defer func() { os.Args = origArgs }()
			os.Args = append([]string{"replace"}, tt.args...)

			if !config.ParseArgs() {
				t.Error("ParseArgs returned false")
				return
			}

			if config.Pattern != tt.expectedPattern {
				t.Errorf("Expected pattern %q, got %q", tt.expectedPattern, config.Pattern)
			}
			if config.Replace != tt.expectedReplace {
				t.Errorf("Expected replace %q, got %q", tt.expectedReplace, config.Replace)
			}
		})
	}
}

func TestSearcher_MatchStructure(t *testing.T) {
	dir := createTestDir(t, "replace_match_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test.txt", "hello world")

	config := NewConfig()
	config.Pattern = "hello"
	config.Paths = []string{dir}

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	for match := range searcher.results {
		if match.Path == "" {
			t.Error("Match path should not be empty")
		}
		if match.LineNum == 0 {
			t.Error("Match line number should not be zero")
		}
		if match.Line == "" {
			t.Error("Match line should not be empty")
		}
		if len(match.Matches) == 0 {
			t.Error("Match should have at least one match pair")
		}
		break
	}
}

func TestSearcher_MultipleMatchesPerLine(t *testing.T) {
	dir := createTestDir(t, "replace_multi_match_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test.txt", "hello hello hello")

	config := NewConfig()
	config.Pattern = "hello"
	config.Paths = []string{dir}

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	for match := range searcher.results {
		if len(match.Matches) != 3 {
			t.Errorf("Expected 3 matches in line, got %d", len(match.Matches))
		}
		break
	}
}

func TestSearcher_EmptyFile(t *testing.T) {
	dir := createTestDir(t, "replace_empty_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "empty.txt", "")

	config := NewConfig()
	config.Pattern = "hello"
	config.Paths = []string{dir}

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	var matches []Match
	for match := range searcher.results {
		matches = append(matches, match)
	}

	if len(matches) != 0 {
		t.Errorf("Expected 0 matches in empty file, got %d", len(matches))
	}
}

// TestSearcher_LongLine locks in the 1MB scanner buffer:
// the default bufio.Scanner token limit is 64KB, which would silently
// truncate a >64KB line and miss matches beyond the boundary.
func TestSearcher_LongLine(t *testing.T) {
	dir := createTestDir(t, "replace_longline_test")
	defer os.RemoveAll(dir)

	// 100KB of padding before the needle — exceeds the default 64KB limit.
	padding := make([]byte, 100*1024)
	for i := range padding {
		padding[i] = 'x'
	}
	content := string(padding) + "NEEDLE" + string(padding)
	createTestFile(t, dir, "long.txt", content)

	config := NewConfig()
	config.Pattern = "NEEDLE"
	config.Paths = []string{dir}

	searcher, err := NewSearcher(config)
	if err != nil {
		t.Fatalf("NewSearcher() error = %v", err)
	}

	searcher.Search()

	var matches []Match
	for match := range searcher.results {
		matches = append(matches, match)
	}

	if len(matches) != 1 {
		t.Fatalf("Expected 1 match in long-line file, got %d (scanner buffer may have truncated the line)", len(matches))
	}
	if matches[0].LineNum != 1 {
		t.Errorf("Expected match on line 1, got line %d", matches[0].LineNum)
	}
}
