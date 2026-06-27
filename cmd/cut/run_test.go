package cut

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestUnescapeDelim(t *testing.T) {
	tests := []struct{ in, want string }{
		{`\t`, "\t"},
		{`\n`, "\n"},
		{`\\`, `\`},
		{"", "\t"},
		{",", ","},
		{":", ":"},
	}
	for _, tt := range tests {
		if got := unescapeDelim(tt.in); got != tt.want {
			t.Errorf("unescapeDelim(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseFields_Errors(t *testing.T) {
	bad := []string{"", "0", "a", "-", "2-1", "1,,2", "1.5"}
	for _, spec := range bad {
		t.Run(spec, func(t *testing.T) {
			if _, err := parseFields(spec); err == nil {
				t.Errorf("expected error for %q", spec)
			}
		})
	}
}

// cutSpec is a behavior-driven helper: parse the spec, configure a Config,
// and return cutLine applied to the input. This avoids coupling tests to
// the internal []fieldRange representation.
func cutSpec(t *testing.T, spec, delim, line string) string {
	t.Helper()
	c := &Config{Delimiter: delim}
	ranges, err := parseFields(spec)
	if err != nil {
		t.Fatalf("parseFields(%q): %v", spec, err)
	}
	c.parsedRanges = ranges
	return c.cutLine(line, delim)
}

func TestCutLine_Single(t *testing.T) {
	got := cutSpec(t, "1", ",", "a,b,c,d")
	if got != "a" {
		t.Errorf("got %q, want a", got)
	}
}

func TestCutLine_Multi(t *testing.T) {
	got := cutSpec(t, "1,3,5", ",", "a,b,c,d,e")
	want := "a,c,e"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCutLine_Range(t *testing.T) {
	got := cutSpec(t, "2-4", ",", "a,b,c,d,e")
	want := "b,c,d"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCutLine_OpenEnded(t *testing.T) {
	got := cutSpec(t, "2-", ",", "a,b,c,d")
	want := "b,c,d"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCutLine_StartOpen(t *testing.T) {
	got := cutSpec(t, "-3", ",", "a,b,c,d,e")
	want := "a,b,c"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCutLine_Mixed(t *testing.T) {
	got := cutSpec(t, "1,3-5,7-", ",", "a,b,c,d,e,f,g,h,i")
	want := "a,c,d,e,g,h,i"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCutLine_Unordered(t *testing.T) {
	// Spec order is normalized; output is in ascending field order.
	got := cutSpec(t, "5,1,3", ",", "a,b,c,d,e")
	want := "a,c,e"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCutLine_OutOfRange(t *testing.T) {
	got := cutSpec(t, "1,5,10", ",", "a,b,c")
	want := "a"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCutLine_NoDelimiter(t *testing.T) {
	// Without delimiter, line passes through unchanged.
	got := cutSpec(t, "1", ",", "hello")
	if got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

func TestCutLine_OutputDelimiter(t *testing.T) {
	c := &Config{Delimiter: ","}
	c.parsedRanges, _ = parseFields("1,3")
	got := c.cutLine("a,b,c,d", "|")
	want := "a|c"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRun_File(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.csv")
	if err := os.WriteFile(p, []byte("a,b,c\n1,2,3\nx,y,z"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save/restore os.Args and stdout.
	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() { os.Args = oldArgs; os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w
	// Simulate main.go: it strips "gx" and sets os.Args[0] = "cut",
	// so Run() sees os.Args[1:] = the real args.
	os.Args = []string{"cut", "-f", "2,1", "-d", ",", p}

	code := Run()
	w.Close()
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	// GNU cut always outputs fields in ascending order regardless of -f
	// spec order, so `-f 2,1` yields fields 1,2 (not 2,1).
	want := "a,b\n1,2\nx,y\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestRun_NoFieldsFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cut", "-d", ","}

	if code := Run(); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_BadFieldSpec(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cut", "-f", "0", "-d", ","}

	if code := Run(); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_SkipNoDelimiter(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("a,b,c\nNO_DELIM\n1,2,3"), 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() { os.Args = oldArgs; os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"cut", "-f", "1", "-d", ",", "-s", p}

	code := Run()
	w.Close()
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	want := "a\n1\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestRun_OutputDelimiter(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.csv")
	if err := os.WriteFile(p, []byte("a,b,c"), 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() { os.Args = oldArgs; os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"cut", "-f", "1,2", "-d", ",", "--output-delimiter", "|", p}

	code := Run()
	w.Close()
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	want := "a|b\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}
