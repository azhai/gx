package stream

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsStdin(t *testing.T) {
	tests := []struct {
		paths []string
		want  bool
	}{
		{nil, true},
		{[]string{}, true},
		{[]string{"-"}, true},
		{[]string{".", "-"}, false},
		{[]string{"file.txt"}, false},
		{[]string{"a", "b"}, false},
	}
	for _, tt := range tests {
		if got := IsStdin(tt.paths); got != tt.want {
			t.Errorf("IsStdin(%v) = %v, want %v", tt.paths, got, tt.want)
		}
	}
}

func TestOpenInput_Stdin(t *testing.T) {
	r, closer, err := OpenInput("")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r != os.Stdin {
		t.Error("expected os.Stdin for empty path")
	}
	if closer != nil {
		t.Error("closer should be nil for stdin")
	}
}

func TestOpenInput_File(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("hello"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	r, closer, err := OpenInput(p)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if closer == nil {
		t.Error("closer should be non-nil for file")
	}
	defer closer.Close()
	buf := make([]byte, 5)
	n, _ := r.Read(buf)
	if string(buf[:n]) != "hello" {
		t.Errorf("got %q, want hello", buf[:n])
	}
}

func TestOpenInput_Nonexistent(t *testing.T) {
	_, _, err := OpenInput("/no/such/file/12345")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadAll_File(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("content"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, name, err := ReadAll([]string{p})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(data) != "content" {
		t.Errorf("got %q, want content", data)
	}
	if name != p {
		t.Errorf("name = %q, want %q", name, p)
	}
}

func TestReadAll_Stdin(t *testing.T) {
	// Simulate stdin.
	old := os.Stdin
	defer func() { os.Stdin = old }()

	tmp, err := os.CreateTemp("", "stdin")
	if err != nil {
		t.Fatalf("createtemp: %v", err)
	}
	defer tmp.Close()
	if err := os.WriteFile(tmp.Name(), []byte("stdin data"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	f, err := os.Open(tmp.Name())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	os.Stdin = f

	data, name, err := ReadAll(nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(data) != "stdin data" {
		t.Errorf("got %q, want 'stdin data'", data)
	}
	if name != StdinName {
		t.Errorf("name = %q, want %q", name, StdinName)
	}
}
