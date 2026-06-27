package trans

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestTransforms(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"upper", "hello", "HELLO"},
		{"upper", "Hello World", "HELLO WORLD"},
		{"lower", "HELLO", "hello"},
		{"lower", "Hello World", "hello world"},
		{"trim", "  hi  ", "hi"},
		{"trim", "\t\nhi\n\t", "hi"},
		{"squeeze", "  a   b   c  ", "a b c"},
		{"squeeze", "a\tb\n c", "a b c"},
		{"reverse", "abc", "cba"},
		{"reverse", "你好", "好你"}, // rune-wise, not byte-wise
		{"reverse", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := transforms[tt.name]
			if !ok {
				t.Fatalf("unknown transform %q", tt.name)
			}
			if got := fn(tt.in); got != tt.want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
			}
		})
	}
}

func TestAvailableTransforms(t *testing.T) {
	want := []string{"lower", "reverse", "squeeze", "trim", "upper"}
	var got []string
	for k := range transforms {
		got = append(got, k)
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("got %d transforms, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("transform[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRun_Stdin(t *testing.T) {
	oldArgs := os.Args
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() { os.Args = oldArgs; os.Stdin = oldStdin; os.Stdout = oldStdout }()

	// Wire stdin to a temp file.
	inFile, _ := os.CreateTemp("", "trans-in")
	defer os.Remove(inFile.Name())
	inFile.WriteString("hello\nworld")
	inFile.Seek(0, 0)
	defer inFile.Close()
	os.Stdin = inFile

	r, w, _ := os.Pipe()
	os.Stdout = w
	// Simulate main.go: os.Args[0] = "trans".
	os.Args = []string{"trans", "upper"}

	code := Run()
	w.Close()
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	want := "HELLO\nWORLD\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestRun_File(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("  hi  \n  bye  "), 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() { os.Args = oldArgs; os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"trans", "trim", p}

	code := Run()
	w.Close()
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	want := "hi\nbye\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestRun_UnknownTransform(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"trans", "bogus"}

	if code := Run(); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_MissingTransform(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	// No positional arg → ParseSimple sees 0 args → prints usage → returns false.
	os.Args = []string{"trans"}

	if code := Run(); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}
