package replace

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/azhai/gx/args"
	"github.com/azhai/gx/processor"
	"github.com/azhai/gx/regex"
	"github.com/azhai/gx/walker"
)

// Config holds the configuration for the replace command.
// It extends args.CommonConfig with replace-specific options.
type Config struct {
	args.CommonConfig
	// ShowLineNum indicates whether to show line numbers in output
	ShowLineNum bool
	// Color indicates whether to use colored output
	Color bool
	// Workers is the number of concurrent worker goroutines
	Workers int
	// FixedString indicates whether to treat pattern as literal string
	FixedString bool
}

// Match represents a single match found in a file.
type Match struct {
	// Path is the file path where the match was found
	Path string
	// LineNum is the line number where the match was found
	LineNum int
	// Line is the content of the line containing the match
	Line string
	// Matches is a slice of [start, end] index pairs for each match in the line
	Matches [][]int
}

// Searcher performs search and replace operations.
//
// Search mode uses processor.Engine for the walk + worker-pool pipeline
// (ProcessFile/HandleResult implement FileProcessor). Replace mode keeps
// its own worker pool because it reads whole files and writes them back,
// a different pattern from line-by-line search.
type Searcher struct {
	// config holds the searcher configuration
	config *Config
	// matcher is the regex matcher used for pattern matching
	matcher *regex.Matcher
	// results is the channel through which match results are sent
	results chan Match
}

// NewConfig creates a new Config with default values.
//
// Default Workers is 1 (single-process, matching grep/sed defaults) for
// predictable resource usage. Use `-j 0` to opt into all CPU cores, or
// `-j N` for an explicit count.
//
// Default values:
//   - ShowLineNum: true
//   - Color: true
//   - Workers: 1
//   - DryRun: true
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
		{
			Short: "-F", Long: "--fixed-strings",
			Help: "Treat pattern as literal string",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				c.FixedString = true
				return true
			},
		},
		{
			Short: "-g", Long: "--glob", HasValue: true, ValueName: "PATTERN",
			Help: "File glob pattern (e.g., \"*.go\")",
			Handler: func(v string, cfg *args.CommonConfig) bool {
				cfg.FilePattern = v
				return true
			},
		},
		{
			Short: "-i", Long: "--ignore-case",
			Help: "Case insensitive search",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				cfg.IgnoreCase = true
				return true
			},
		},
		{
			Short: "-j", Long: "--threads", HasValue: true, ValueName: "N",
			Help: "Worker threads (0 = all cores, default 1)",
			Handler: func(v string, cfg *args.CommonConfig) bool {
				fmt.Sscanf(v, "%d", &c.Workers)
				return true
			},
		},
		{
			Short: "-n", Long: "--line-number",
			Help: "Show line numbers (default)",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				c.ShowLineNum = true
				return true
			},
		},
		{
			Short: "-N", Long: "--no-line-number",
			Help: "Hide line numbers",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				c.ShowLineNum = false
				return true
			},
		},
		{
			Long: "--no-color",
			Help: "Disable colored output",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				c.Color = false
				return true
			},
		},
		{
			Short: "-f", Long: "--find", HasValue: true, ValueName: "PATTERN",
			Help: "Pattern to find (regex supported)",
			Handler: func(v string, cfg *args.CommonConfig) bool {
				cfg.Pattern = v
				return true
			},
		},
		{
			Short: "-r", Long: "--replace", HasValue: true, ValueName: "STRING",
			Help: "Replacement string",
			Handler: func(v string, cfg *args.CommonConfig) bool {
				cfg.Replace = v
				cfg.ReplaceSet = true
				return true
			},
		},
		{
			Short: "-x", Long: "--exec",
			Help: "Execute replacement (default: dry-run)",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				cfg.DryRun = false
				return true
			},
		},
	}
}

// ParseArgs parses command-line arguments using the shared args.ParseSimple function.
// Returns true if parsing was successful, false otherwise.
func (c *Config) ParseArgs() bool {
	options := c.getOptions()
	return args.ParseSimple(os.Args[1:], &c.CommonConfig, options, c.printUsage)
}

// printUsage prints the usage information for the replace command.
func (c *Config) printUsage() {
	options := c.getOptions()
	fmt.Println(`replace - A fast search and replace tool inspired by ripgrep

Usage: replace [OPTIONS] PATTERN [REPLACE] [PATH...]

Options:
` + args.FormatOptions(options) + `
Examples:
  replace "foo" "bar"                      Replace 'foo' with 'bar' (dry-run)
  replace "foo" "bar" -x                   Execute the replacement
  replace "old" "new" ./src                Search and replace in ./src
  replace -i "error" "warning"             Case insensitive replacement
  replace "\d+" "NUM" -g "*.log"           Replace numbers in log files`)
}

// NewSearcher creates a new Searcher with the given configuration.
// It initializes the regex matcher and returns an error if the pattern is invalid.
func NewSearcher(config *Config) (*Searcher, error) {
	matcher, err := regex.NewMatcher(&regex.Config{
		Pattern:     config.Pattern,
		IgnoreCase:  config.IgnoreCase,
		FixedString: config.FixedString,
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

// Search performs a search operation across all files.
// It delegates the walk + worker-pool pipeline to processor.Engine and
// forwards each result to the results channel (drained by PrintResults).
func (s *Searcher) Search() {
	s.results = make(chan Match, 1000)
	go func() {
		s.newEngine(true).Run()
		close(s.results)
	}()
}

// ProcessFile implements processor.FileProcessor.
// It reads the file line by line and returns all matches as results.
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
// It forwards the result to the results channel for PrintResults to drain.
func (s *Searcher) HandleResult(r processor.Result) {
	s.results <- Match{
		Path:    r.Path,
		LineNum: r.LineNum,
		Line:    r.Line,
		Matches: r.Matches,
	}
}

// newEngine builds a processor.Engine wired to this searcher's walker config.
// skipBinary controls whether binary files are skipped (search skips them,
// replace reads them since it may rewrite).
func (s *Searcher) newEngine(skipBinary bool) *processor.Engine {
	walkerConfig := walker.NewConfig()
	walkerConfig.Paths = s.config.Paths
	walkerConfig.FilePattern = s.config.FilePattern
	walkerConfig.SkipBinary = skipBinary
	w := walker.New(walkerConfig)
	return processor.New(w, s.matcher, s, s.config.Workers)
}

// PrintResults prints all matches from the results channel.
func (s *Searcher) PrintResults() {
	for match := range s.results {
		s.printMatch(match)
	}
}

// printMatch prints a single match with optional coloring.
// It shows the file path, line number (if enabled), and the matched line
// with the matched text highlighted in red.
func (s *Searcher) printMatch(match Match) {
	if s.config.ShowLineNum {
		fmt.Printf("%s:%d:", match.Path, match.LineNum)
	} else {
		fmt.Printf("%s:", match.Path)
	}

	if s.config.Color {
		lastEnd := 0
		line := match.Line
		for _, m := range match.Matches {
			fmt.Print(line[lastEnd:m[0]])
			fmt.Printf("\x1b[31m%s\x1b[0m", line[m[0]:m[1]])
			lastEnd = m[1]
		}
		fmt.Println(line[lastEnd:])
	} else {
		fmt.Println(match.Line)
	}
}

// Replace performs a replace operation across all files.
// It uses concurrent workers to process files in parallel.
// In dry-run mode, it shows what would be replaced without making changes.
func (s *Searcher) Replace() {
	walkerConfig := walker.NewConfig()
	walkerConfig.Paths = s.config.Paths
	walkerConfig.FilePattern = s.config.FilePattern
	walkerConfig.SkipBinary = false

	fileWalker := walker.New(walkerConfig)
	files := fileWalker.Walk()

	filesChan := make(chan walker.FileInfo, 1000)
	go func() {
		for f := range files {
			filesChan <- f
		}
		close(filesChan)
	}()

	var wg sync.WaitGroup
	for i := 0; i < s.config.Workers; i++ {
		wg.Add(1)
		go s.replaceWorker(filesChan, &wg)
	}

	wg.Wait()
}

// replaceWorker is a worker function that processes files from the channel.
// It performs replacement on each file.
func (s *Searcher) replaceWorker(files <-chan walker.FileInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range files {
		s.replaceFile(file.Path)
	}
}

// replaceFile performs replacement on a single file.
// It reads the file, performs the replacement, and shows the changes.
// In dry-run mode, it only shows what would be changed without modifying the file.
func (s *Searcher) replaceFile(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	matches := s.matcher.FindAll(content)
	if matches == nil {
		return
	}

	newContent := s.matcher.ReplaceAll(content, []byte(s.config.Replace))

	if s.config.DryRun {
		fmt.Printf("\x1b[33m[DRY-RUN]\x1b[0m %s\n", path)
	} else {
		err := os.WriteFile(path, newContent, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\x1b[31mError writing %s: %v\x1b[0m\n", path, err)
		} else {
			fmt.Printf("\x1b[32m[REPLACED]\x1b[0m %s\n", path)
		}
	}

	// Reuse the already-read content bytes for scanning instead of
	// allocating a second copy via string(content) + strings.NewReader.
	// bytes.NewReader wraps the slice without copying.
	scanner := bufio.NewScanner(bytes.NewReader(content))
	// Match the 1MB buffer used in searchFile so long lines aren't truncated.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		lineMatches := s.matcher.FindAll(line)
		if lineMatches != nil {
			s.printMatch(Match{
				Path:    path,
				LineNum: lineNum,
				Line:    string(line),
				Matches: lineMatches,
			})
			newLine := s.matcher.ReplaceAllString(string(line), s.config.Replace)
			if newLine != string(line) {
				fmt.Printf("\x1b[36m-> %s\x1b[0m\n", newLine)
			}
		}
	}
}
