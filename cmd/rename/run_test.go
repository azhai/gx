package rename

import (
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

func createTestFile(t *testing.T, dir, name string) {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}
	if !config.DryRun {
		t.Error("DryRun should be true by default")
	}
}

func TestNewRenamer(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid pattern",
			config:  &Config{CommonConfig: args.CommonConfig{Pattern: "test"}},
			wantErr: false,
		},
		{
			name:    "invalid regex",
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
			renamer, err := NewRenamer(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRenamer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && renamer == nil {
				t.Error("NewRenamer() returned nil renamer")
			}
		})
	}
}

func TestDetectConflicts(t *testing.T) {
	tests := []struct {
		name          string
		ops           []RenameOp
		expectedConfs []bool
	}{
		{
			name: "no conflicts",
			ops: []RenameOp{
				{Original: "file1.txt", NewName: "file1_new.txt", BaseDir: "/tmp"},
				{Original: "file2.txt", NewName: "file2_new.txt", BaseDir: "/tmp"},
			},
			expectedConfs: []bool{false, false},
		},
		{
			name: "same new name conflict",
			ops: []RenameOp{
				{Original: "file1.txt", NewName: "renamed.txt", BaseDir: "/tmp"},
				{Original: "file2.txt", NewName: "renamed.txt", BaseDir: "/tmp"},
			},
			expectedConfs: []bool{true, true},
		},
		{
			name: "case insensitive conflict - same file",
			ops: []RenameOp{
				{Original: "Makefile", NewName: "makefile", BaseDir: "/tmp"},
			},
			expectedConfs: []bool{false},
		},
		{
			name: "multiple conflicts",
			ops: []RenameOp{
				{Original: "a.txt", NewName: "same.txt", BaseDir: "/tmp"},
				{Original: "b.txt", NewName: "same.txt", BaseDir: "/tmp"},
				{Original: "c.txt", NewName: "same.txt", BaseDir: "/tmp"},
			},
			expectedConfs: []bool{true, true, true},
		},
		{
			name: "case insensitive conflict - different files",
			ops: []RenameOp{
				{Original: "File.txt", NewName: "FILE.txt", BaseDir: "/tmp"},
			},
			expectedConfs: []bool{false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Renamer{
				config: &Config{
					CommonConfig: args.CommonConfig{},
				},
				ops: tt.ops,
			}

			r.DetectConflicts()

			if len(r.ops) != len(tt.expectedConfs) {
				t.Errorf("expected %d ops, got %d", len(tt.expectedConfs), len(r.ops))
				return
			}

			for i, op := range r.ops {
				if op.Conflict != tt.expectedConfs[i] {
					t.Errorf("op %d: expected conflict %v, got %v", i, tt.expectedConfs[i], op.Conflict)
				}
			}
		})
	}
}

func TestCaseInsensitiveFilesystem(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gre_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "TestFile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	lowercasePath := filepath.Join(tmpDir, "testfile.txt")
	_, err = os.Stat(lowercasePath)
	caseInsensitive := err == nil

	t.Logf("Filesystem is case-insensitive: %v", caseInsensitive)

	r := &Renamer{
		config: &Config{
			CommonConfig: args.CommonConfig{},
		},
		ops: []RenameOp{
			{Original: "TestFile.txt", NewName: "testfile.txt", BaseDir: tmpDir},
		},
	}

	r.DetectConflicts()

	if len(r.ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(r.ops))
	}

	if r.ops[0].Conflict {
		t.Errorf("expected no conflict for case-only rename on case-insensitive filesystem")
	}
}

func TestConflictWithExistingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gre_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	r := &Renamer{
		config: &Config{
			CommonConfig: args.CommonConfig{},
		},
		ops: []RenameOp{
			{Original: "new.txt", NewName: "existing.txt", BaseDir: tmpDir},
		},
	}

	r.DetectConflicts()

	if len(r.ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(r.ops))
	}

	if !r.ops[0].Conflict {
		t.Errorf("expected conflict when renaming to existing file")
	}
}

func TestCollectFiles(t *testing.T) {
	dir := createTestDir(t, "rename_collect_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test1.txt")
	createTestFile(t, dir, "test2.txt")
	createTestFile(t, dir, "other.txt")

	config := NewConfig()
	config.Pattern = "test"
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) < 2 {
		t.Errorf("Expected at least 2 files to rename, got %d", len(renamer.ops))
	}
}

func TestCollectFilesWithGlob(t *testing.T) {
	dir := createTestDir(t, "rename_glob_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test1.txt")
	createTestFile(t, dir, "test2.go")
	createTestFile(t, dir, "test3.txt")

	config := NewConfig()
	config.Pattern = "test"
	config.FilePattern = "*.txt"
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) != 2 {
		t.Errorf("Expected 2 txt files to rename, got %d", len(renamer.ops))
	}
}

func TestCollectFilesWithReplace(t *testing.T) {
	dir := createTestDir(t, "rename_replace_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "test_file.txt")

	config := NewConfig()
	config.Pattern = "test_"
	config.Replace = ""
	config.ReplaceSet = true
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) != 1 {
		t.Fatalf("Expected 1 file to rename, got %d", len(renamer.ops))
	}

	if renamer.ops[0].NewName != "file.txt" {
		t.Errorf("Expected new name 'file.txt', got %q", renamer.ops[0].NewName)
	}
}

func TestCollectFilesIncludeDir(t *testing.T) {
	dir := createTestDir(t, "rename_include_dir_test")
	defer os.RemoveAll(dir)

	subDir := filepath.Join(dir, "test_subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	createTestFile(t, dir, "test_file.txt")

	config := NewConfig()
	config.Pattern = "test"
	config.IncludeDir = true
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	foundDir := false
	for _, op := range renamer.ops {
		if op.Original == "test_subdir" {
			foundDir = true
		}
	}

	if !foundDir {
		t.Error("Expected to find directory when IncludeDir is true")
	}
}

func TestCollectFilesFixedString(t *testing.T) {
	dir := createTestDir(t, "rename_fixed_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "[test].txt")
	createTestFile(t, dir, "test.txt")

	config := NewConfig()
	config.Pattern = "[test]"
	config.FixedString = true
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) != 1 {
		t.Errorf("Expected 1 file with fixed string match, got %d", len(renamer.ops))
	}
}

func TestCollectFilesIgnoreCase(t *testing.T) {
	dir := createTestDir(t, "rename_ignore_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "TEST_file.txt")
	createTestFile(t, dir, "test_other.txt")
	createTestFile(t, dir, "TeSt_another.txt")

	config := NewConfig()
	config.Pattern = "test"
	config.IgnoreCase = true
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) != 3 {
		t.Errorf("Expected 3 files with ignore case, got %d", len(renamer.ops))
	}
}

func TestExecuteDryRun(t *testing.T) {
	dir := createTestDir(t, "rename_dryrun_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "old.txt")

	config := NewConfig()
	config.Pattern = "old"
	config.Replace = "new"
	config.ReplaceSet = true
	config.Paths = []string{dir}
	config.DryRun = true

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if err := renamer.Execute(); err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "old.txt")); os.IsNotExist(err) {
		t.Error("Dry-run should not rename files")
	}

	if _, err := os.Stat(filepath.Join(dir, "new.txt")); err == nil {
		t.Error("Dry-run should not create new files")
	}
}

func TestExecuteRename(t *testing.T) {
	dir := createTestDir(t, "rename_execute_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "old.txt")

	config := NewConfig()
	config.Pattern = "old"
	config.Replace = "new"
	config.ReplaceSet = true
	config.Paths = []string{dir}
	config.DryRun = false

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()
	renamer.DetectConflicts()

	if err := renamer.Execute(); err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "old.txt")); err == nil {
		t.Error("Old file should be renamed")
	}

	if _, err := os.Stat(filepath.Join(dir, "new.txt")); os.IsNotExist(err) {
		t.Error("New file should exist")
	}
}

func TestExecuteWithConflicts(t *testing.T) {
	dir := createTestDir(t, "rename_conflict_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "file_a.txt")
	createTestFile(t, dir, "file_b.txt")

	config := NewConfig()
	config.Pattern = "file_[ab]"
	config.Replace = "same"
	config.ReplaceSet = true
	config.Paths = []string{dir}
	config.DryRun = false

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()
	renamer.DetectConflicts()

	conflictCount := 0
	for _, op := range renamer.ops {
		if op.Conflict {
			conflictCount++
		}
	}

	if conflictCount == 0 {
		t.Error("Expected conflicts to be detected")
	}

	err = renamer.Execute()
	if err == nil && conflictCount > 0 {
		t.Error("Expected error when conflicts detected without force")
	}
}

func TestExecuteWithForce(t *testing.T) {
	dir := createTestDir(t, "rename_force_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "file1.txt")
	createTestFile(t, dir, "file2.txt")

	config := NewConfig()
	config.Pattern = "file"
	config.Replace = "same"
	config.ReplaceSet = true
	config.Paths = []string{dir}
	config.DryRun = false
	config.Force = true

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()
	renamer.DetectConflicts()

	if err := renamer.Execute(); err != nil {
		t.Errorf("Execute() with force error = %v", err)
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
			args:            []string{"old"},
			expectedPattern: "old",
			expectedPaths:   []string{"."},
		},
		{
			name:            "pattern and path",
			args:            []string{"old", "/src"},
			expectedPattern: "old",
			expectedPaths:   []string{"/src"},
		},
		{
			name:            "pattern, replace and path",
			args:            []string{"old", "new", "/src"},
			expectedPattern: "old",
			expectedReplace: "new",
			expectedPaths:   []string{"/src"},
		},
		{
			// Two positional args are now PATTERN PATH; quote detection removed.
			name:            "pattern and literal-quoted path",
			args:            []string{"old", `"new"`},
			expectedPattern: "old",
			expectedPaths:   []string{`"new"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig()
			origArgs := os.Args
			defer func() { os.Args = origArgs }()
			os.Args = append([]string{"rename"}, tt.args...)

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

func TestRenameOp(t *testing.T) {
	op := RenameOp{
		Original: "old.txt",
		NewName:  "new.txt",
		BaseDir:  "/tmp",
		Conflict: false,
	}

	if op.Original != "old.txt" {
		t.Errorf("Expected Original 'old.txt', got %q", op.Original)
	}
	if op.NewName != "new.txt" {
		t.Errorf("Expected NewName 'new.txt', got %q", op.NewName)
	}
	if op.BaseDir != "/tmp" {
		t.Errorf("Expected BaseDir '/tmp', got %q", op.BaseDir)
	}
	if op.Conflict {
		t.Error("Expected Conflict to be false")
	}
}

func TestCollectFilesNoMatch(t *testing.T) {
	dir := createTestDir(t, "rename_nomatch_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "file.txt")

	config := NewConfig()
	config.Pattern = "notfound"
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) != 0 {
		t.Errorf("Expected 0 files when no match, got %d", len(renamer.ops))
	}
}

func TestCollectFilesEmptyReplace(t *testing.T) {
	dir := createTestDir(t, "rename_empty_replace_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "prefix_file.txt")

	config := NewConfig()
	config.Pattern = "prefix_"
	config.Replace = ""
	config.ReplaceSet = true
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(renamer.ops))
	}

	if renamer.ops[0].NewName != "file.txt" {
		t.Errorf("Expected 'file.txt', got %q", renamer.ops[0].NewName)
	}
}

func TestCollectFilesWithGroups(t *testing.T) {
	dir := createTestDir(t, "rename_groups_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "IMG_1234.jpg")
	createTestFile(t, dir, "IMG_5678.jpg")

	config := NewConfig()
	config.Pattern = `IMG_(\d+)`
	config.Replace = "photo_$1"
	config.ReplaceSet = true
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(renamer.ops))
	}

	expectedNames := map[string]string{
		"IMG_1234.jpg": "photo_1234.jpg",
		"IMG_5678.jpg": "photo_5678.jpg",
	}

	for _, op := range renamer.ops {
		expected, ok := expectedNames[op.Original]
		if !ok {
			t.Errorf("Unexpected file: %s", op.Original)
			continue
		}
		if op.NewName != expected {
			t.Errorf("For %s: expected %q, got %q", op.Original, expected, op.NewName)
		}
	}
}

func TestCollectFilesNoChange(t *testing.T) {
	dir := createTestDir(t, "rename_nochange_test")
	defer os.RemoveAll(dir)

	createTestFile(t, dir, "file.txt")

	config := NewConfig()
	config.Pattern = "xyz"
	config.Replace = "abc"
	config.ReplaceSet = true
	config.Paths = []string{dir}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) != 0 {
		t.Errorf("Expected 0 files when pattern doesn't match, got %d", len(renamer.ops))
	}
}

func TestMultipleDirectories(t *testing.T) {
	dir1 := createTestDir(t, "rename_multi_test1")
	dir2 := createTestDir(t, "rename_multi_test2")
	defer os.RemoveAll(dir1)
	defer os.RemoveAll(dir2)

	createTestFile(t, dir1, "test1.txt")
	createTestFile(t, dir2, "test2.txt")

	config := NewConfig()
	config.Pattern = "test"
	config.Paths = []string{dir1, dir2}

	renamer, err := NewRenamer(config)
	if err != nil {
		t.Fatalf("NewRenamer() error = %v", err)
	}

	renamer.CollectFiles()

	if len(renamer.ops) != 2 {
		t.Errorf("Expected 2 files from 2 directories, got %d", len(renamer.ops))
	}
}
