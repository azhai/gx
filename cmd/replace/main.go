package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/azhai/gre/pkg/regex"
	"github.com/azhai/gre/pkg/walker"
)

type Config struct {
	Pattern     string
	Paths       []string
	IgnoreCase  bool
	ShowLineNum bool
	Color       bool
	FilePattern string
	Workers     int
}

type Match struct {
	Path    string
	LineNum int
	Line    string
	Matches [][]int
}

type Searcher struct {
	config  *Config
	matcher *regex.Matcher
	results chan Match
	wg      sync.WaitGroup
}

func NewConfig() *Config {
	return &Config{
		ShowLineNum: true,
		Color:       true,
		Workers:     runtime.NumCPU(),
	}
}

func (c *Config) ParseArgs() bool {
	args := os.Args[1:]
	if len(args) == 0 {
		c.printUsage()
		return false
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "-i" || arg == "--ignore-case" {
			c.IgnoreCase = true
		} else if arg == "-n" || arg == "--line-number" {
			c.ShowLineNum = true
		} else if arg == "-N" || arg == "--no-line-number" {
			c.ShowLineNum = false
		} else if arg == "--no-color" {
			c.Color = false
		} else if arg == "-g" || arg == "--glob" {
			if i+1 < len(args) {
				c.FilePattern = args[i+1]
				i++
			}
		} else if arg == "-j" || arg == "--threads" {
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &c.Workers)
				i++
			}
		} else if arg == "-h" || arg == "--help" {
			c.printUsage()
			return false
		} else if arg[0] != '-' {
			if c.Pattern == "" {
				c.Pattern = arg
			} else {
				c.Paths = append(c.Paths, arg)
			}
		}
		i++
	}

	if c.Pattern == "" {
		fmt.Fprintln(os.Stderr, "Error: pattern is required")
		return false
	}

	if len(c.Paths) == 0 {
		c.Paths = []string{"."}
	}

	return true
}

func (c *Config) printUsage() {
	fmt.Println(`replace - A fast search tool inspired by ripgrep

Usage: replace [OPTIONS] PATTERN [PATH...]

Options:
  -i, --ignore-case    Case insensitive search
  -n, --line-number    Show line numbers (default: true)
  -N, --no-line-number Hide line numbers
  --no-color           Disable colored output
  -g, --glob <PATTERN> File glob pattern (e.g., "*.go")
  -j, --threads <NUM>  Number of worker threads (default: CPU count)
  -h, --help           Show this help message

Examples:
  replace "pattern"                    Search for pattern in current directory
  replace -i "pattern" /path           Case insensitive search
  replace -g "*.go" "func"             Search only in Go files
  replace "TODO" src/ test/            Search in multiple directories`)
}

func NewSearcher(config *Config) (*Searcher, error) {
	matcher, err := regex.NewMatcher(&regex.Config{
		Pattern:    config.Pattern,
		IgnoreCase: config.IgnoreCase,
	})
	if err != nil {
		return nil, err
	}

	return &Searcher{
		config:  config,
		matcher: matcher,
		results: make(chan Match, 1000),
	}, nil
}

func (s *Searcher) Search() {
	walkerConfig := walker.NewConfig()
	walkerConfig.Paths = s.config.Paths
	walkerConfig.FilePattern = s.config.FilePattern
	walkerConfig.SkipBinary = true

	fileWalker := walker.New(walkerConfig)
	files := fileWalker.Walk()

	filesChan := make(chan walker.FileInfo, 1000)
	go func() {
		for f := range files {
			filesChan <- f
		}
		close(filesChan)
	}()

	for i := 0; i < s.config.Workers; i++ {
		s.wg.Add(1)
		go s.worker(filesChan)
	}

	go func() {
		s.wg.Wait()
		close(s.results)
	}()
}

func (s *Searcher) worker(files <-chan walker.FileInfo) {
	defer s.wg.Done()

	for file := range files {
		s.searchFile(file.Path)
	}
}

func (s *Searcher) searchFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		matches := s.matcher.FindAll(line)
		if matches != nil {
			s.results <- Match{
				Path:    path,
				LineNum: lineNum,
				Line:    string(line),
				Matches: matches,
			}
		}
	}
}

func (s *Searcher) PrintResults() {
	for match := range s.results {
		s.printMatch(match)
	}
}

func (s *Searcher) printMatch(m Match) {
	path := m.Path

	if s.config.Color {
		path = fmt.Sprintf("\x1b[35m%s\x1b[0m", path)
	}

	if s.config.ShowLineNum {
		lineNumStr := fmt.Sprintf("%d", m.LineNum)
		if s.config.Color {
			lineNumStr = fmt.Sprintf("\x1b[32m%s\x1b[0m", lineNumStr)
		}
		fmt.Printf("%s:%s:%s\n", path, lineNumStr, s.highlightLine(m.Line, m.Matches))
	} else {
		fmt.Printf("%s:%s\n", path, s.highlightLine(m.Line, m.Matches))
	}
}

func (s *Searcher) highlightLine(line string, matches [][]int) string {
	if !s.config.Color || len(matches) == 0 {
		return line
	}

	result := ""
	lastEnd := 0

	for _, match := range matches {
		start := match[0]
		end := match[1]

		if start > lastEnd {
			result += line[lastEnd:start]
		}

		result += fmt.Sprintf("\x1b[31;1m%s\x1b[0m", line[start:end])
		lastEnd = end
	}

	if lastEnd < len(line) {
		result += line[lastEnd:]
	}

	return result
}

func main() {
	config := NewConfig()
	if !config.ParseArgs() {
		os.Exit(1)
	}

	searcher, err := NewSearcher(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	searcher.Search()
	searcher.PrintResults()
}
