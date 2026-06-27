package processor

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/azhai/gx/regex"
	"github.com/azhai/gx/walker"
)

// fakeProc records every result it handles. HandleResult is called from
// worker goroutines, so access is guarded by a mutex.
type fakeProc struct {
	mu      sync.Mutex
	results []Result
}

func (p *fakeProc) ProcessFile(path string) []Result {
	// Emit one synthetic result per file.
	return []Result{{Path: path, LineNum: 1, Line: "x", Matches: [][]int{{0, 1}}}}
}

func (p *fakeProc) HandleResult(r Result) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.results = append(p.results, r)
}

func (p *fakeProc) len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.results)
}

func (p *fakeProc) paths() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.results))
	for i, r := range p.results {
		out[i] = r.Path
	}
	return out
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestEngine_Run(t *testing.T) {
	dir := t.TempDir()
	for i := range 5 {
		writeFile(t, filepath.Join(dir, "f"+string(rune('0'+i))+".txt"), "x")
	}

	wc := walker.NewConfig()
	wc.Paths = []string{dir}
	w := walker.New(wc)

	m, err := regex.NewMatcher(&regex.Config{Pattern: "x"})
	if err != nil {
		t.Fatalf("matcher: %v", err)
	}

	proc := &fakeProc{}
	eng := New(w, m, proc, 4)
	eng.Run()

	if got := proc.len(); got != 5 {
		t.Fatalf("expected 5 results, got %d", got)
	}
}

// TestEngine_RunSingleWorker verifies the pipeline works with workers=1
// (the new default for find/replace, matching grep/sed single-process mode).
func TestEngine_RunSingleWorker(t *testing.T) {
	dir := t.TempDir()
	for i := range 3 {
		writeFile(t, filepath.Join(dir, "f"+string(rune('0'+i))+".txt"), "x")
	}

	wc := walker.NewConfig()
	wc.Paths = []string{dir}
	w := walker.New(wc)

	m, err := regex.NewMatcher(&regex.Config{Pattern: "x"})
	if err != nil {
		t.Fatalf("matcher: %v", err)
	}

	proc := &fakeProc{}
	eng := New(w, m, proc, 1)
	eng.Run()

	if got := proc.len(); got != 3 {
		t.Fatalf("expected 3 results with 1 worker, got %d", got)
	}
}

// TestEngine_RunConcurrent verifies concurrent-safe result collection
// across many files with multiple workers (no lost results, no races).
func TestEngine_RunConcurrent(t *testing.T) {
	dir := t.TempDir()
	const n = 50
	for i := range n {
		writeFile(t, filepath.Join(dir, filepath.Join("sub", "f"+string(rune('0'+i%10))+string(rune('0'+i/10))+".txt")), "x")
	}

	wc := walker.NewConfig()
	wc.Paths = []string{dir}
	w := walker.New(wc)

	m, err := regex.NewMatcher(&regex.Config{Pattern: "x"})
	if err != nil {
		t.Fatalf("matcher: %v", err)
	}

	proc := &fakeProc{}
	eng := New(w, m, proc, 8)
	eng.Run()

	if got := proc.len(); got != n {
		t.Fatalf("expected %d results across %d workers, got %d", n, 8, got)
	}
}

// TestEngine_RunWorkersZeroDefaultsToNumCPU verifies that workers <= 0
// is treated as "use all cores" (the -j 0 convention).
func TestEngine_RunWorkersZeroDefaultsToNumCPU(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "f.txt"), "x")

	wc := walker.NewConfig()
	wc.Paths = []string{dir}
	w := walker.New(wc)

	m, err := regex.NewMatcher(&regex.Config{Pattern: "x"})
	if err != nil {
		t.Fatalf("matcher: %v", err)
	}

	proc := &fakeProc{}
	eng := New(w, m, proc, 0)
	if eng.workers <= 0 {
		t.Fatalf("workers=0 should default to NumCPU (>0), got %d", eng.workers)
	}
	eng.Run()

	if got := proc.len(); got != 1 {
		t.Fatalf("expected 1 result, got %d", got)
	}
}

// TestEngine_Matcher verifies the engine exposes its matcher so
// ProcessFile implementations can reuse the precompiled regex.
func TestEngine_Matcher(t *testing.T) {
	wc := walker.NewConfig()
	w := walker.New(wc)
	m, err := regex.NewMatcher(&regex.Config{Pattern: "x"})
	if err != nil {
		t.Fatalf("matcher: %v", err)
	}
	eng := New(w, m, &fakeProc{}, 1)
	if eng.Matcher() != m {
		t.Error("Matcher() should return the matcher passed to New")
	}
}
