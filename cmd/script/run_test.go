package script

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// newCompiled builds a script source string and compiles it once for tests.
// It mirrors production setup: imports the safe module map, pre-imports the
// safe modules (so `text.to_upper` etc. work in the test body), and
// pre-declares `line`/`lineno`/`filename` so the script can reference them
// at compile time.
func newCompiled(t *testing.T, src string, unsafe bool) *tengo.Compiled {
	t.Helper()
	var prelude strings.Builder
	for _, name := range safeModules {
		fmt.Fprintf(&prelude, "%s := import(%q)\n", name, name)
	}
	s := tengo.NewScript([]byte(prelude.String() + src))
	if unsafe {
		s.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
	} else {
		s.SetImports(stdlib.GetModuleMap(safeModules...))
	}
	if err := declareGlobals(s, false); err != nil {
		t.Fatalf("declare: %v", err)
	}
	c, err := s.Compile()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return c
}

func TestExecOnce_StringOutput(t *testing.T) {
	c := newCompiled(t, `__out := text.to_upper(line)`, false)
	cfg := &Config{timeout: time.Second}
	out, skip, err := cfg.execOnce(c, map[string]tengo.Object{
		"line": &tengo.String{Value: "hi"},
	})
	if err != nil || skip || out != "HI" {
		t.Fatalf("out=%q skip=%v err=%v", out, skip, err)
	}
}

func TestExecOnce_LineNoExpr_WrapsAsOutVar(t *testing.T) {
	// Mirrors Config.loadSource -e behavior: wrap as `__out := (EXPR)`.
	// newCompiled already prepends the safe-module prelude, so we only add
	// the __out wrapper here (not the full loadSource output).
	cfg := &Config{expr: `line + "!"`, timeout: time.Second}
	src := "__out := (" + cfg.expr + ")"
	c := newCompiled(t, src, false)
	out, skip, err := cfg.execOnce(c, map[string]tengo.Object{
		"line": &tengo.String{Value: "hi"},
	})
	if err != nil || skip || out != "hi!" {
		t.Fatalf("out=%q skip=%v err=%v", out, skip, err)
	}
}

func TestExecOnce_LineNoExpr_WithLineno(t *testing.T) {
	cfg := &Config{expr: `lineno % 2 == 0 ? text.to_upper(line) : line`, timeout: time.Second}
	src := "__out := (" + cfg.expr + ")"
	c := newCompiled(t, src, false)

	// Line 1: passes through.
	out, skip, err := cfg.execOnce(c, map[string]tengo.Object{
		"line":   &tengo.String{Value: "foo"},
		"lineno": &tengo.Int{Value: 1},
	})
	if err != nil || skip || out != "foo" {
		t.Fatalf("line1: out=%q skip=%v err=%v", out, skip, err)
	}
	// Line 2: uppercased.
	out, skip, err = cfg.execOnce(c, map[string]tengo.Object{
		"line":   &tengo.String{Value: "foo"},
		"lineno": &tengo.Int{Value: 2},
	})
	if err != nil || skip || out != "FOO" {
		t.Fatalf("line2: out=%q skip=%v err=%v", out, skip, err)
	}
}

func TestExecOnce_SkipOnUndefined(t *testing.T) {
	// Empty body never assigns __out → Undefined → skip.
	c := newCompiled(t, `x := 1`, false)
	cfg := &Config{timeout: time.Second}
	out, skip, err := cfg.execOnce(c, map[string]tengo.Object{
		"line": &tengo.String{Value: "hi"},
	})
	if err != nil || !skip || out != "" {
		t.Fatalf("out=%q skip=%v err=%v", out, skip, err)
	}
}

func TestExecOnce_SkipOnFalse(t *testing.T) {
	c := newCompiled(t, `__out := false`, false)
	cfg := &Config{timeout: time.Second}
	out, skip, err := cfg.execOnce(c, map[string]tengo.Object{
		"line": &tengo.String{Value: "hi"},
	})
	if err != nil || !skip || out != "" {
		t.Fatalf("out=%q skip=%v err=%v", out, skip, err)
	}
}

func TestExecOnce_TrueEmitsLiteralTrue(t *testing.T) {
	c := newCompiled(t, `__out := true`, false)
	cfg := &Config{timeout: time.Second}
	out, skip, err := cfg.execOnce(c, map[string]tengo.Object{
		"line": &tengo.String{Value: "hi"},
	})
	if err != nil || skip || out != "true" {
		t.Fatalf("out=%q skip=%v err=%v", out, skip, err)
	}
}

func TestExecOnce_FilterGrepStyle(t *testing.T) {
	// Lines containing "TODO": emit; else skip.
	src := `__out := text.contains(line, "TODO") ? line : false`
	c := newCompiled(t, src, false)
	cfg := &Config{timeout: time.Second}

	cases := []struct {
		line string
		want string
		skip bool
	}{
		{"TODO: fix", "TODO: fix", false},
		{"nothing here", "", true},
		{"TODO: another", "TODO: another", false},
	}
	for _, tc := range cases {
		out, skip, err := cfg.execOnce(c, map[string]tengo.Object{
			"line": &tengo.String{Value: tc.line},
		})
		if err != nil {
			t.Fatalf("line %q: %v", tc.line, err)
		}
		if skip != tc.skip || (!skip && out != tc.want) {
			t.Errorf("line %q: out=%q skip=%v, want %q/%v", tc.line, out, skip, tc.want, tc.skip)
		}
	}
}

func TestExecOnce_RuntimeError(t *testing.T) {
	// Calling a non-existent method on a string triggers a runtime error.
	c := newCompiled(t, `__out := line.foo()`, false)
	cfg := &Config{timeout: time.Second}
	_, _, err := cfg.execOnce(c, map[string]tengo.Object{
		"line": &tengo.String{Value: "hi"},
	})
	if err == nil {
		t.Fatalf("expected runtime error, got nil")
	}
}

func TestExecOnce_TimeoutKillsInfiniteLoop(t *testing.T) {
	c := newCompiled(t, `for { }; __out := "x"`, false)
	cfg := &Config{timeout: 50 * time.Millisecond}
	_, _, err := cfg.execOnce(c, map[string]tengo.Object{
		"line": &tengo.String{Value: "hi"},
	})
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

func TestRun_LineMode_PipelineStdin(t *testing.T) {
	oldArgs := os.Args
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() { os.Args = oldArgs; os.Stdin = oldStdin; os.Stdout = oldStdout }()

	inFile, _ := os.CreateTemp("", "script-in")
	defer os.Remove(inFile.Name())
	inFile.WriteString("hello\nworld")
	inFile.Seek(0, 0)
	defer inFile.Close()
	os.Stdin = inFile

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"script", "-e", "text.to_upper(line)"}

	code := Run()
	w.Close()
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var buf bytes.Buffer
	buf.ReadFrom(r)
	want := "HELLO\nWORLD\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestRun_LineMode_File(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("a\nb\nc"), 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() { os.Args = oldArgs; os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"script", "-e", `line + "!"`, p}

	code := Run()
	w.Close()
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var buf bytes.Buffer
	buf.ReadFrom(r)
	want := "a!\nb!\nc!\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestRun_LineMode_FilterExit1(t *testing.T) {
	// All lines filtered → no output → exit 1.
	oldArgs := os.Args
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() { os.Args = oldArgs; os.Stdin = oldStdin; os.Stdout = oldStdout }()

	inFile, _ := os.CreateTemp("", "script-in")
	defer os.Remove(inFile.Name())
	inFile.WriteString("abc\nxyz")
	inFile.Seek(0, 0)
	defer inFile.Close()
	os.Stdin = inFile

	_, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"script", "-e", `false`}

	code := Run()
	w.Close()
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (no output)", code)
	}
}

func TestRun_CompileError(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"script", "-e", `line +`}

	if code := Run(); code != 2 {
		t.Errorf("compile error: exit = %d, want 2", code)
	}
}

func TestRun_NoSource(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"script", "input.txt"} // no -e, no -f

	if code := Run(); code != 2 {
		t.Errorf("no source: exit = %d, want 2", code)
	}
}

func TestRun_FileMode_WholeFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() { os.Args = oldArgs; os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w
	// Whole-file mode with -e: count newlines via pre-imported strings module.
	os.Args = []string{"script", "--whole", "-e", `text.count(content, "\n")`, p}

	code := Run()
	w.Close()
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var buf bytes.Buffer
	buf.ReadFrom(r)
	got := strings.TrimSpace(buf.String())
	if !strings.Contains(got, "2") {
		t.Errorf("got %q, want output containing line count 2", got)
	}
}

func TestRun_BadTimeout(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"script", "-e", "line", "--timeout", "not-a-duration"}

	if code := Run(); code != 2 {
		t.Errorf("bad timeout: exit = %d, want 2", code)
	}
}

// Avoid unused-import warnings for context/time helpers used above.
var _ = context.Background
