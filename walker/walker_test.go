package walker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
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
	if len(config.SkipDirs) == 0 {
		t.Error("NewConfig() SkipDirs should not be empty")
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

func TestWalker_EmptyDirectory(t *testing.T) {
	dir := createTestDir(t, "walker_empty_test")
	defer os.RemoveAll(dir)

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	count := 0
	for range files {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d", count)
	}
}

func TestWalker_NonExistentPath(t *testing.T) {
	config := NewConfig()
	config.Paths = []string{"/non/existent/path/12345"}

	walker := New(config)
	files := walker.Walk()

	count := 0
	for range files {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 files for non-existent path, got %d", count)
	}
}

func TestWalker_DeepNestedDirectories(t *testing.T) {
	dir := createTestDir(t, "walker_deep_test")
	defer os.RemoveAll(dir)

	deepPath := filepath.Join(dir, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatalf("Failed to create deep path: %v", err)
	}

	createTestFile(t, deepPath, "deep.txt", "deep content")

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	found := false
	for file := range files {
		if file.Name == "deep.txt" {
			found = true
		}
	}

	if !found {
		t.Error("Expected to find file in deep nested directory")
	}
}

func TestWalker_HiddenFiles(t *testing.T) {
	dir := createTestDir(t, "walker_hidden_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "visible.txt", "visible")
	createTestFile(t, dir, ".hidden", "hidden")

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	var foundHidden, foundVisible bool
	for file := range files {
		if file.Name == ".hidden" {
			foundHidden = true
		}
		if file.Name == "visible.txt" {
			foundVisible = true
		}
	}

	if !foundHidden {
		t.Error("Expected to find hidden file")
	}
	if !foundVisible {
		t.Error("Expected to find visible file")
	}
}

func TestWalker_Symlinks(t *testing.T) {
	dir := createTestDir(t, "walker_symlink_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "original.txt", "original content")

	linkPath := filepath.Join(dir, "link.txt")
	originalPath := filepath.Join(dir, "original.txt")
	if err := os.Symlink(originalPath, linkPath); err != nil {
		t.Skipf("Symlink not supported: %v", err)
	}

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	count := 0
	for range files {
		count++
	}

	if count < 1 {
		t.Errorf("Expected at least 1 file, got %d", count)
	}
}

func TestWalker_SpecialCharactersInFilenames(t *testing.T) {
	dir := createTestDir(t, "walker_special_test")
	defer os.RemoveAll(dir)

	specialNames := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.multiple.dots.txt",
	}

	for _, name := range specialNames {
		createTestFile(t, dir, name, "content")
	}

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	count := 0
	for range files {
		count++
	}

	if count != len(specialNames) {
		t.Errorf("Expected %d files, got %d", len(specialNames), count)
	}
}

func TestWalker_LargeDirectory(t *testing.T) {
	dir := createTestDir(t, "walker_large_test")
	defer os.RemoveAll(dir)

	numFiles := 100
	for i := range numFiles {
		createTestFile(t, dir, filepath.Join("subdir", fmt.Sprintf("file%d.txt", i)), "content")
	}

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	count := 0
	for range files {
		count++
	}

	if count != numFiles {
		t.Errorf("Expected %d files, got %d", numFiles, count)
	}
}

func TestWalker_GlobPatterns(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		files    []string
		expected int
	}{
		{"all go files", "*.go", []string{"a.go", "b.go", "c.txt"}, 2},
		{"all txt files", "*.txt", []string{"a.txt", "b.txt", "c.go"}, 2},
		{"all files", "*", []string{"a.txt", "b.go", "c.md"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTestDir(t, "walker_glob_"+tt.name)
			defer os.RemoveAll(dir)

			for _, f := range tt.files {
				createTestFile(t, dir, f, "content")
			}

			config := NewConfig()
			config.Paths = []string{dir}
			config.FilePattern = tt.pattern

			walker := New(config)
			files := walker.Walk()

			count := 0
			for range files {
				count++
			}

			if count != tt.expected {
				t.Errorf("Expected %d files, got %d", tt.expected, count)
			}
		})
	}
}

func TestWalker_SkipDirsWithCustomConfig(t *testing.T) {
	dir := createTestDir(t, "walker_custom_skip_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test.txt", "content")

	customSkipDir := filepath.Join(dir, "custom_skip")
	if err := os.MkdirAll(customSkipDir, 0755); err != nil {
		t.Fatalf("Failed to create custom_skip dir: %v", err)
	}
	createTestFile(t, dir, "custom_skip/file.txt", "should be skipped")

	config := NewConfig()
	config.Paths = []string{dir}
	config.SkipDirs = map[string]bool{"custom_skip": true}

	walker := New(config)
	files := walker.Walk()

	for file := range files {
		if file.Name == "file.txt" {
			t.Error("Should skip custom_skip directory")
		}
	}
}

func TestWalker_FileInfoFields(t *testing.T) {
	dir := createTestDir(t, "walker_fields_test")
	defer os.RemoveAll(dir)

	content := "test content with specific length"
	createTestFile(t, dir, "test.txt", content)

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	for file := range files {
		if file.Name != "test.txt" {
			continue
		}

		if file.IsDir {
			t.Error("File should not be a directory")
		}

		if file.Size != int64(len(content)) {
			t.Errorf("Expected size %d, got %d", len(content), file.Size)
		}

		expectedPath := filepath.Join(dir, "test.txt")
		if file.Path != expectedPath {
			t.Errorf("Expected path %q, got %q", expectedPath, file.Path)
		}
	}
}

func TestWalker_ConcurrentWalk(t *testing.T) {
	dir := createTestDir(t, "walker_concurrent_test")
	defer os.RemoveAll(dir)

	for i := range 50 {
		createTestFile(t, dir, filepath.Join("subdir", fmt.Sprintf("file%d.txt", i)), "content")
	}

	config := NewConfig()
	config.Paths = []string{dir}

	walker := New(config)
	files := walker.Walk()

	count := 0
	for range files {
		count++
	}

	if count != 50 {
		t.Errorf("Expected 50 files, got %d", count)
	}
}

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    bool
	}{
		{"text file", []byte("hello world"), false},
		{"binary with null", []byte{0x00, 0x01, 0x02}, true},
		{"empty file", []byte{}, false},
		{"utf-8 text", []byte("你好世界"), false},
		{"mixed content", []byte("hello\x00world"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTestDir(t, "walker_binary_check_test")
			defer os.RemoveAll(dir)

			filePath := filepath.Join(dir, "test.bin")
			if err := os.WriteFile(filePath, tt.content, 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			config := NewConfig()
			config.Paths = []string{dir}
			config.SkipBinary = true

			walker := New(config)

			result := walker.isBinaryFile(filePath)
			if result != tt.want {
				t.Errorf("isBinaryFile() = %v, want %v", result, tt.want)
			}
		})
	}
}
