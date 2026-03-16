package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/azhai/rego/pkg/args"
)

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
