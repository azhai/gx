// Package find implements the gx find subcommand: search file contents
// for a pattern (grep-like) and print matches with line numbers.
package find

import (
	"bufio"
	"fmt"
	"os"

	"github.com/azhai/gx/args"
	"github.com/azhai/gx/pkg/processor"
	"github.com/azhai/gx/regex"
	"github.com/azhai/gx/walker"
)

// Config holds the configuration for the find command.
type Config struct {
	args.CommonConfig
	ShowLineNum bool
	Color       bool
	Workers     int
	FixedString bool
}

// NewConfig creates a Config with find defaults.
//
// Default Workers is 1 (single-process, matching grep defaults). Use
// `-j 0` to opt into all CPU cores, or `-j N` for an explicit count.
func NewConfig() *Config {
	return &Config{
		ShowLineNum: true,
		Color:       true,
		Workers:     1,
		CommonConfig: args.CommonConfig{
			DryRun: true,
		},
	}
}

func (c *Config) getOptions() []args.Option {
	return []args.Option{
		{Short: "-F", Long: "--fixed-strings", Help: "Treat pattern as literal string",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.FixedString = true; return true }},
		{Short: "-g", Long: "--glob", HasValue: true, ValueName: "PATTERN", Help: "File glob pattern",
			Handler: func(v string, cfg *args.CommonConfig) bool { cfg.FilePattern = v; return true }},
		{Short: "-i", Long: "--ignore-case", Help: "Case insensitive search",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.CommonConfig.IgnoreCase = true; return true }},
		{Short: "-j", Long: "--threads", HasValue: true, ValueName: "N", Help: "Worker threads (0 = all cores, default 1)",
			Handler: func(v string, _ *args.CommonConfig) bool { fmt.Sscanf(v, "%d", &c.Workers); return true }},
		{Short: "-n", Long: "--line-number", Help: "Show line numbers (default)",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.ShowLineNum = true; return true }},
		{Short: "-N", Long: "--no-line-number", Help: "Hide line numbers",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.ShowLineNum = false; return true }},
		{Long: "--no-color", Help: "Disable colored output",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.Color = false; return true }},
		{Short: "-f", Long: "--find", HasValue: true, ValueName: "PATTERN", Help: "Pattern to find (regex supported)",
			Handler: func(v string, cfg *args.CommonConfig) bool { cfg.Pattern = v; return true }},
	}
}

// ParseArgs parses command-line arguments.
func (c *Config) ParseArgs() bool {
	return args.ParseSimple(os.Args[1:], &c.CommonConfig, c.getOptions(), c.printUsage)
}

func (c *Config) printUsage() {
	fmt.Println(`find - A fast file content search tool inspired by ripgrep

Usage: find [OPTIONS] PATTERN [PATH...]

Options:
` + args.FormatOptions(c.getOptions()) + `
Examples:
  find "TODO" ./src
  find -i "error" -g "*.go"`)
}

// Searcher performs search operations.
type Searcher struct {
	config  *Config
	matcher *regex.Matcher
	results chan processor.Result
}

// NewSearcher creates a Searcher.
func NewSearcher(config *Config) (*Searcher, error) {
	m, err := regex.NewMatcher(&regex.Config{
		Pattern:     config.Pattern,
		IgnoreCase:  config.IgnoreCase,
		FixedString: config.FixedString,
	})
	if err != nil {
		return nil, err
	}
	return &Searcher{config: config, matcher: m}, nil
}

// ProcessFile implements processor.FileProcessor.
// Reads the file line by line and returns all matches.
func (s *Searcher) ProcessFile(path string) []processor.Result {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// bufio.Scanner's default max token size is 64KB, which silently
	// truncates long lines (e.g. minified JS). Raise to 1MB.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var results []processor.Result
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		matches := s.matcher.FindAll(line)
		if matches != nil {
			results = append(results, processor.Result{
				Path:    path,
				LineNum: lineNum,
				Line:    string(line),
				Matches: matches,
			})
		}
	}
	return results
}

// HandleResult implements processor.FileProcessor.
// Forwards the result to the results channel for PrintResults to drain.
func (s *Searcher) HandleResult(r processor.Result) {
	s.results <- r
}

// Search runs the search pipeline.
func (s *Searcher) Search() {
	s.results = make(chan processor.Result, 1000)
	go func() {
		wc := walker.NewConfig()
		wc.Paths = s.config.Paths
		wc.FilePattern = s.config.FilePattern
		wc.SkipBinary = true
		w := walker.New(wc)
		processor.New(w, s.matcher, s, s.config.Workers).Run()
		close(s.results)
	}()
}

// PrintResults drains the results channel and prints each match.
// Returns the number of matches printed.
func (s *Searcher) PrintResults() int {
	count := 0
	for r := range s.results {
		s.printMatch(r)
		count++
	}
	return count
}

func (s *Searcher) printMatch(r processor.Result) {
	if s.config.ShowLineNum {
		fmt.Printf("%s:%d:", r.Path, r.LineNum)
	} else {
		fmt.Printf("%s:", r.Path)
	}
	if s.config.Color {
		lastEnd := 0
		for _, m := range r.Matches {
			fmt.Print(r.Line[lastEnd:m[0]])
			fmt.Printf("\x1b[31m%s\x1b[0m", r.Line[m[0]:m[1]])
			lastEnd = m[1]
		}
		fmt.Println(r.Line[lastEnd:])
	} else {
		fmt.Println(r.Line)
	}
}
