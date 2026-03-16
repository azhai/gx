package walker

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/azhai/gre/pkg/regex"
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
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}
	if len(config.SkipDirs) == 0 {
		t.Error("NewConfig() SkipDirs should not be empty")
	}
	if config.Workers <= 0 {
		t.Error("NewConfig() Workers should be positive")
	}
}

func TestWalker_Walk(t *testing.T) {
	dir := createTestDir(t, "walker_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test1.txt", "content1")
	createTestFile(t, dir, "test2.go", "content2")

	subDir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	createTestFile(t, dir, "subdir/test3.txt", "content3")

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	var fileNames []string
	for file := range files {
		fileNames = append(fileNames, file.Name)
	}

	if len(fileNames) < 3 {
		t.Errorf("Expected at least 3 files, got %d", len(fileNames))
	}
}

func TestWalker_WalkWithGlob(t *testing.T) {
	dir := createTestDir(t, "walker_glob_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test1.txt", "content1")
	createTestFile(t, dir, "test2.go", "content2")
	createTestFile(t, dir, "test3.txt", "content3")

	config := NewConfig()
	config.Paths = []string{dir}
	config.FilePattern = "*.txt"

	walker := New(config)
	files := walker.Walk()

	var fileNames []string
	for file := range files {
		fileNames = append(fileNames, file.Name)
	}

	sort.Strings(fileNames)

	if len(fileNames) != 2 {
		t.Errorf("Expected 2 txt files, got %d: %v", len(fileNames), fileNames)
	}
}

func TestWalker_SkipDirs(t *testing.T) {
	dir := createTestDir(t, "walker_skip_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test.txt", "content")

	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}
	createTestFile(t, dir, ".git/config", "git config")

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	for file := range files {
		if file.Name == "config" {
			t.Error("Should not walk into .git directory")
		}
	}
}

func TestWalker_IncludeDir(t *testing.T) {
	dir := createTestDir(t, "walker_include_test")
	defer os.RemoveAll(dir)

	subDir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	config := NewConfig()
	config.Paths = []string{dir}
	config.IncludeDir = true

	walker := New(config)
	files := walker.Walk()

	foundDir := false
	for file := range files {
		if file.IsDir && file.Name == "subdir" {
			foundDir = true
		}
	}

	if !foundDir {
		t.Error("Expected to find subdir when IncludeDir is true")
	}
}

func TestWalker_MultiplePaths(t *testing.T) {
	dir1 := createTestDir(t, "walker_multi_test1")
	dir2 := createTestDir(t, "walker_multi_test2")
	defer os.RemoveAll(dir1)
	defer os.RemoveAll(dir2)

	createTestFile(t, dir1, "file1.txt", "content1")
	createTestFile(t, dir2, "file2.txt", "content2")

	config := NewConfig()
	config.Paths = []string{dir1, dir2}

	walker := New(config)
	files := walker.Walk()

	count := 0
	for range files {
		count++
	}

	if count != 2 {
		t.Errorf("Expected 2 files from 2 paths, got %d", count)
	}
}

func TestWalker_FileInfo(t *testing.T) {
	dir := createTestDir(t, "walker_info_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test.txt", "hello world")

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	for file := range files {
		if file.Name != "test.txt" {
			continue
		}
		if file.IsDir {
			t.Error("test.txt should not be a directory")
		}
		if file.Size != 11 {
			t.Errorf("Expected size 11, got %d", file.Size)
		}
		if file.Path != filepath.Join(dir, "test.txt") {
			t.Errorf("Unexpected path: %s", file.Path)
		}
	}
}

func TestDefaultSkipDirs(t *testing.T) {
	expectedDirs := []string{".git", ".hg", ".svn", "node_modules", "vendor"}
	for _, dir := range expectedDirs {
		if !DefaultSkipDirs[dir] {
			t.Errorf("Expected %q in DefaultSkipDirs", dir)
		}
	}
}

func TestWalker_SkipBinary(t *testing.T) {
	dir := createTestDir(t, "walker_binary_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "text.txt", "hello world")

	binaryContent := []byte{0x00, 0x01, 0x02, 0x03}
	binaryPath := filepath.Join(dir, "binary.bin")
	if err := os.WriteFile(binaryPath, binaryContent, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	config := NewConfig()
	config.Paths = []string{dir}
	config.SkipBinary = true

	walker := New(config)
	files := walker.Walk()

	for file := range files {
		if file.Name == "binary.bin" {
			t.Error("Should skip binary file when SkipBinary is true")
		}
	}
}

func TestWalker_ReplaceAllString(t *testing.T) {
	m, err := regex.NewMatcher(&regex.Config{Pattern: `foo`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name    string
		input   string
		replace string
		want    string
	}{
		{"simple replace", "foo bar", "baz", "baz bar"},
		{"multiple replace", "foo foo foo", "x", "x x x"},
		{"no match", "hello world", "baz", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.ReplaceAllString(tt.input, tt.replace); got != tt.want {
				t.Errorf("ReplaceAllString(%q, %q) = %q, want %q", tt.input, tt.replace, got, tt.want)
			}
		})
	}
}
