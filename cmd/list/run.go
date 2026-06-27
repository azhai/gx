// Package list implements the gx list subcommand: print the path of every
// file that contains at least one match (like `grep -l`).
//
// list reuses processor.Engine and short-circuits each file on the first
// match (no further lines are scanned once a match is found).
package list

import (
	"bufio"
	"fmt"
	"os"

	"github.com/azhai/gx/args"
	"github.com/azhai/gx/pkg/processor"
	"github.com/azhai/gx/regex"
	"github.com/azhai/gx/walker"
)

// Config holds the configuration for the list command.
type Config struct {
	args.CommonConfig
	Color       bool
	Workers     int
	FixedString bool
}

// NewConfig creates a Config with list defaults.
//
// Default Workers is 1 (single-process, matching `grep -l`). Use `-j 0`
// for all cores or `-j N` for an explicit count.
func NewConfig() *Config {
	return &Config{
		Color:   true,
		Workers: 1,
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
	fmt.Println(`list - List files containing matches (like grep -l)

Usage: list [OPTIONS] PATTERN [PATH...]

Options:
` + args.FormatOptions(c.getOptions()) + `
Examples:
  list "TODO" ./src
  list -i "error" -g "*.go"`)
}

// Searcher prints the path of every file that contains a match.
type Searcher struct {
	config  *Config
	matcher *regex.Matcher
	// seen dedupes file paths so each file is printed once even when
	// multiple workers race to match different lines of the same file
	// (shouldn't happen since ProcessFile is per-file, but cheap insurance).
	results chan string
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
//
// Short-circuits on the first match: returns a single Result whose Path
// is the matched file. The line content is not surfaced — list only
// reports file paths.
func (s *Searcher) ProcessFile(path string) []processor.Result {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Match the 1MB buffer used elsewhere so long lines aren't truncated.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		if s.matcher.FindAll(scanner.Bytes()) != nil {
			return []processor.Result{{Path: path}}
		}
	}
	return nil
}

// HandleResult implements processor.FileProcessor.
// Sends the matched path to the results channel.
func (s *Searcher) HandleResult(r processor.Result) {
	s.results <- r.Path
}

// Search runs the search pipeline.
func (s *Searcher) Search() {
	s.results = make(chan string, 1000)
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

// PrintResults drains the results channel and prints each matched path.
// Returns the number of files printed.
func (s *Searcher) PrintResults() int {
	count := 0
	for path := range s.results {
		fmt.Println(path)
		count++
	}
	return count
}
