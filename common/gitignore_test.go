package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewGitignorePattern_EmptyAndComment(t *testing.T) {
	if NewGitignorePattern("") != nil {
		t.Error("empty line should return nil")
	}
	if NewGitignorePattern("   ") != nil {
		t.Error("whitespace-only line should return nil")
	}
	if NewGitignorePattern("# comment") != nil {
		t.Error("comment line should return nil")
	}
}

func TestNewGitignorePattern_Glob(t *testing.T) {
	p := NewGitignorePattern("*.log")
	if p == nil {
		t.Fatal("expected non-nil pattern")
	}
	if p.negated {
		t.Error("should not be negated")
	}
	if p.dirOnly {
		t.Error("should not be dirOnly")
	}
	if p.rooted {
		t.Error("should not be rooted")
	}
	if !p.Match("error.log", false) {
		t.Error("*.log should match error.log")
	}
	if p.Match("main.go", false) {
		t.Error("*.log should not match main.go")
	}
}

func TestNewGitignorePattern_DirOnly(t *testing.T) {
	p := NewGitignorePattern("build/")
	if p == nil {
		t.Fatal("expected non-nil pattern")
	}
	if !p.dirOnly {
		t.Error("should be dirOnly")
	}
	if !p.Match("build", true) {
		t.Error("build/ should match directory 'build'")
	}
	if p.Match("build", false) {
		t.Error("build/ should not match file 'build'")
	}
}

func TestNewGitignorePattern_Negated(t *testing.T) {
	p := NewGitignorePattern("!important.log")
	if p == nil {
		t.Fatal("expected non-nil pattern")
	}
	if !p.negated {
		t.Error("should be negated")
	}
	if !p.Match("important.log", false) {
		t.Error("!important.log should match important.log")
	}
}

func TestNewGitignorePattern_Rooted(t *testing.T) {
	p := NewGitignorePattern("/TODO")
	if p == nil {
		t.Fatal("expected non-nil pattern")
	}
	if !p.rooted {
		t.Error("should be rooted")
	}
	if !p.Match("TODO", false) {
		t.Error("/TODO should match TODO in root")
	}
}

func TestNewGitignorePattern_EscapedHash(t *testing.T) {
	p := NewGitignorePattern(`\#hash`)
	if p == nil {
		t.Fatal("expected non-nil pattern")
	}
	if !p.Match("#hash", false) {
		t.Error(`\#hash should match #hash`)
	}
}

func TestNewGitignorePattern_TrailingSpaces(t *testing.T) {
	p := NewGitignorePattern("*.log   ")
	if p == nil {
		t.Fatal("expected non-nil pattern")
	}
	if !p.Match("error.log", false) {
		t.Error("trailing spaces should be trimmed")
	}
}

func TestNewGitignorePattern_PathWithDir(t *testing.T) {
	p := NewGitignorePattern("doc/*.pdf")
	if p == nil {
		t.Fatal("expected non-nil pattern")
	}
	if !p.Match("doc/*.pdf", false) {
		t.Error("doc/*.pdf should match itself as a glob")
	}
}

func TestGitignoreMatcher_EmptyContent(t *testing.T) {
	m := NewGitignoreMatcher("/root", nil, "")
	if m.Match("anything.txt", false) {
		t.Error("empty gitignore should not ignore anything")
	}
}

func TestGitignoreMatcher_CommentsOnly(t *testing.T) {
	m := NewGitignoreMatcher("/root", nil, "# comment\n# another")
	if m.Match("anything.txt", false) {
		t.Error("comments-only gitignore should not ignore anything")
	}
}

func TestGitignoreMatcher_GlobPattern(t *testing.T) {
	m := NewGitignoreMatcher("/root", nil, "*.log\n")
	if !m.Match("error.log", false) {
		t.Error("*.log should ignore error.log")
	}
	if m.Match("main.go", false) {
		t.Error("*.log should not ignore main.go")
	}
}

func TestGitignoreMatcher_DirPattern(t *testing.T) {
	m := NewGitignoreMatcher("/root", nil, "build/\n")
	if !m.Match("build", true) {
		t.Error("build/ should ignore directory 'build'")
	}
	if m.Match("build", false) {
		t.Error("build/ should not ignore file 'build'")
	}
}

func TestGitignoreMatcher_NegatedPattern(t *testing.T) {
	m := NewGitignoreMatcher("/root", nil, "*.log\n!important.log\n")
	if !m.Match("error.log", false) {
		t.Error("*.log should ignore error.log")
	}
	if m.Match("important.log", false) {
		t.Error("!important.log should un-ignore important.log")
	}
}

func TestGitignoreMatcher_MultipleRules(t *testing.T) {
	m := NewGitignoreMatcher("/root", nil, "*.tmp\n*.log\n!important.log\n")
	if !m.Match("file.tmp", false) {
		t.Error("*.tmp should ignore file.tmp")
	}
	if !m.Match("error.log", false) {
		t.Error("*.log should ignore error.log")
	}
	if m.Match("important.log", false) {
		t.Error("!important.log should un-ignore important.log")
	}
}

func TestGitignoreMatcher_MultiLevelInheritance(t *testing.T) {
	parent := NewGitignoreMatcher("/root", nil, "*.log\n")
	child := NewGitignoreMatcher("/root/sub", parent, "!debug.log\n")

	if !child.Match("error.log", false) {
		t.Error("parent *.log should still ignore error.log in child")
	}
	if child.Match("debug.log", false) {
		t.Error("child !debug.log should un-ignore debug.log")
	}
}

func TestGitignoreMatcher_Child(t *testing.T) {
	parent := NewGitignoreMatcher("/root", nil, "*.log\n")
	child := parent.Child("sub")
	if !child.Match("error.log", false) {
		t.Error("child should inherit parent rules")
	}
}

func TestLoadGitignoreFile_NotExists(t *testing.T) {
	parent := &GitignoreMatcher{}
	result := LoadGitignoreFile("/nonexistent/path", parent)
	if result != parent {
		t.Error("should return parent when file doesn't exist")
	}
}

func TestLoadGitignoreFile_Exists(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m := LoadGitignoreFile(dir, nil)
	if m == nil {
		t.Fatal("expected non-nil matcher")
	}
	if !m.Match("error.log", false) {
		t.Error("*.log should ignore error.log")
	}
}
