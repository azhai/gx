// Package processor provides a shared engine for file-by-file processing
// with a worker pool. It is used by find/list/replace commands to avoid
// duplicating walker + worker-pool boilerplate.
package processor

import (
	"runtime"
	"sync"

	"github.com/azhai/gx/regex"
	"github.com/azhai/gx/walker"
)

// Result is a match found in a file.
type Result struct {
	// Path is the file path where the match was found
	Path string
	// LineNum is the line number (1-based)
	LineNum int
	// Line is the matched line content (without trailing newline)
	Line string
	// Matches is a slice of [start, end] index pairs for each match in the line
	Matches [][]int
}

// FileProcessor handles one file and emits results.
//
// Implementations decide how to read the file (line-by-line, whole-file,
// short-circuit on first match) so the engine does not impose a strategy.
//
// Implementations must be safe for concurrent use across multiple worker
// goroutines, because Run dispatches files to a pool of workers. In
// practice this means HandleResult (which the engine calls from worker
// goroutines) must be goroutine-safe, or results must be funneled through
// a channel.
type FileProcessor interface {
	// ProcessFile reads and processes a file, returning the results to emit.
	ProcessFile(path string) []Result
	// HandleResult emits one result (print, collect, etc.).
	HandleResult(r Result)
}

// Engine drives the walk + worker-pool pipeline.
//
// The default worker count is 1 (single-process, matching grep/sed defaults
// for predictable resource usage). Pass workers=0 to use runtime.NumCPU()
// (all cores), or workers=N for an explicit count.
type Engine struct {
	walker  *walker.Walker
	matcher *regex.Matcher
	proc    FileProcessor
	workers int
}

// New creates an Engine. workers <= 0 selects runtime.NumCPU() (use this
// for `-j 0`). Pass an explicit positive value for `-j N`. The zero-value
// default of 1 is enforced by callers setting Workers=1 in their Config.
func New(w *walker.Walker, m *regex.Matcher, p FileProcessor, workers int) *Engine {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &Engine{walker: w, matcher: m, proc: p, workers: workers}
}

// Run starts the pipeline and blocks until all files are processed.
//
// Pipeline: walker emits files → fan out to N workers → each file processed
// by proc.ProcessFile → each result passed to proc.HandleResult.
func (e *Engine) Run() {
	files := e.walker.Walk()

	filesChan := make(chan walker.FileInfo, 1000)
	go func() {
		for f := range files {
			filesChan <- f
		}
		close(filesChan)
	}()

	var wg sync.WaitGroup
	for i := 0; i < e.workers; i++ {
		wg.Add(1)
		go e.worker(filesChan, &wg)
	}
	wg.Wait()
}

func (e *Engine) worker(files <-chan walker.FileInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	for file := range files {
		for _, r := range e.proc.ProcessFile(file.Path) {
			e.proc.HandleResult(r)
		}
	}
}

// Matcher returns the engine's matcher (for ProcessFile implementations
// that need to run matches, e.g. to reuse a precompiled regex).
func (e *Engine) Matcher() *regex.Matcher { return e.matcher }
