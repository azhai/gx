// Package cut implements the gx cut subcommand: extract fields from
// delimited text (a subset of POSIX cut, only `-f` mode).
//
// cut is a stdin/stdout filter. It reads one file (or stdin when no path
// is given or `-` is used), splits each line by a delimiter, and prints
// the selected fields joined by an output delimiter.
package cut

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/azhai/gx/args"
	"github.com/azhai/gx/pkg/stream"
)

// fieldRange describes a contiguous run of fields to select.
// lo and hi are 0-based, inclusive. hi == -1 means "to end of line"
// (the `N-` open-ended form).
type fieldRange struct {
	lo int
	hi int
}

// Config holds the configuration for the cut command.
type Config struct {
	args.CommonConfig
	Fields          string // raw -f spec like "1,3-5,7-"
	Delimiter       string // -d (default tab)
	OutputDelimiter string // --output-delimiter (default = Delimiter)
	SkipNoDelimiter bool   // -s
	parsedRanges    []fieldRange
}

// NewConfig creates a Config with cut defaults.
func NewConfig() *Config {
	return &Config{
		Delimiter: "\t",
		CommonConfig: args.CommonConfig{
			DryRun: true,
		},
	}
}

func (c *Config) getOptions() []args.Option {
	return []args.Option{
		{Short: "-f", Long: "--fields", HasValue: true, ValueName: "LIST", Help: "Field list (e.g. 1,3-5,7-,2-4)",
			Handler: func(v string, _ *args.CommonConfig) bool { c.Fields = v; return true }},
		{Short: "-d", Long: "--delimiter", HasValue: true, ValueName: "DELIM", Help: "Field delimiter (default tab, supports \\t \\n \\\\)",
			Handler: func(v string, _ *args.CommonConfig) bool { c.Delimiter = unescapeDelim(v); return true }},
		{Short: "-s", Long: "--only-delimited", Help: "Skip lines without the delimiter",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.SkipNoDelimiter = true; return true }},
		{Long: "--output-delimiter", HasValue: true, ValueName: "DELIM", Help: "Output delimiter (default = -d)",
			Handler: func(v string, _ *args.CommonConfig) bool { c.OutputDelimiter = unescapeDelim(v); return true }},
	}
}

// ParseArgs parses command-line arguments.
func (c *Config) ParseArgs() bool {
	return args.ParseSimple(os.Args[1:], &c.CommonConfig, c.getOptions(), c.printUsage)
}

func (c *Config) printUsage() {
	fmt.Println(`cut - Extract fields from delimited text (only -f mode)

Usage: cut -f LIST [OPTIONS] [FILE]

Options:
` + args.FormatOptions(c.getOptions()) + `
Field LIST syntax:
  N       field N (1-based)
  N-M     fields N through M (inclusive)
  N-      fields N through end of line
  -M      fields 1 through M

Examples:
  cut -f 2 -d , file.csv
  cut -f 1,3 -d '\\t' file.tsv
  cut -f 2- -d ',' file.csv
  cat file | gx cut -f 1 -d ':'`)
}

// unescapeDelim turns the literal "\\t", "\\n", "\\\\" into the actual
// control characters. Single-char delimiters pass through unchanged.
// This matches GNU cut's behavior for backslash escapes.
func unescapeDelim(s string) string {
	switch s {
	case `\t`:
		return "\t"
	case `\n`:
		return "\n"
	case `\\`:
		return `\`
	case "":
		return "\t" // empty -d falls back to tab
	default:
		return s
	}
}

// parseFields parses a field list like "1,3-5,7-,-3" into a sorted list
// of field ranges. Returns an error on invalid syntax.
//
// Field numbers are 1-based in the spec but stored 0-based internally.
// "7-" → {lo:6, hi:-1} (open-ended, to end of line).
// "-3" → {lo:0, hi:2} (fields 1-3).
// Single "N" → {lo:N-1, hi:N-1}.
func parseFields(spec string) ([]fieldRange, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("empty field list")
	}

	var ranges []fieldRange
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty field in list: %q", spec)
		}

		if i := strings.IndexByte(part, '-'); i >= 0 {
			loStr := part[:i]
			hiStr := part[i+1:]

			// "-" alone is invalid (POSIX requires at least one bound).
			if loStr == "" && hiStr == "" {
				return nil, fmt.Errorf("bad field range %q", part)
			}

			var lo, hi int
			var err error
			if loStr == "" {
				lo = 1 // -M → 1..M
			} else {
				lo, err = strconv.Atoi(loStr)
				if err != nil || lo < 1 {
					return nil, fmt.Errorf("bad field number %q", loStr)
				}
			}
			if hiStr == "" {
				hi = -1 // N- → open-ended
			} else {
				hi, err = strconv.Atoi(hiStr)
				if err != nil || hi < 1 {
					return nil, fmt.Errorf("bad field number %q", hiStr)
				}
				if hi < lo {
					return nil, fmt.Errorf("bad range %q: high < low", part)
				}
			}
			// Store 0-based, inclusive. hi stays -1 for open-ended
			// (the -1 sentinel must survive the lo/hi → 0-based shift).
			storeHi := hi - 1
			if hi == -1 {
				storeHi = -1
			}
			ranges = append(ranges, fieldRange{lo: lo - 1, hi: storeHi})
			continue
		}

		n, err := strconv.Atoi(part)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("bad field number %q", part)
		}
		ranges = append(ranges, fieldRange{lo: n - 1, hi: n - 1})
	}

	// Sort by lo so output is deterministic across input order.
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].lo < ranges[j].lo })
	return ranges, nil
}

// Run executes the cut command. Returns the process exit code:
//   - 0: success
//   - 2: argument/IO error
func Run() int {
	c := NewConfig()
	if !c.ParseArgs() {
		return 2
	}
	if c.Fields == "" {
		fmt.Fprintln(os.Stderr, "cut: -f is required")
		return 2
	}

	ranges, err := parseFields(c.Fields)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cut: %v\n", err)
		return 2
	}
	c.parsedRanges = ranges

	// cut has no Pattern concept; ParseCommon stuffed the first positional
	// arg into Pattern. Pull it back as the input path. When no positional
	// arg is given, default to stdin.
	if c.Pattern != "" {
		c.Paths = []string{c.Pattern}
	} else {
		c.Paths = []string{"-"} // stdin marker
	}

	outDelim := c.OutputDelimiter
	if outDelim == "" {
		outDelim = c.Delimiter
	}

	data, _, err := stream.ReadAll(c.Paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cut: %v\n", err)
		return 2
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, c.Delimiter) {
			if c.SkipNoDelimiter {
				continue // -s: skip lines without the delimiter entirely
			}
			w.WriteString(line)
			w.WriteByte('\n')
			continue
		}
		parts := strings.Split(line, c.Delimiter)
		var picked []string
		for _, r := range c.parsedRanges {
			hi := r.hi
			if hi == -1 || hi >= len(parts) {
				hi = len(parts) - 1 // open-ended or out-of-range → cap at end
			}
			for idx := r.lo; idx <= hi; idx++ {
				if idx < len(parts) {
					picked = append(picked, parts[idx])
				}
			}
		}
		w.WriteString(strings.Join(picked, outDelim))
		w.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "cut: %v\n", err)
		return 2
	}
	return 0
}

// cutLine applies the parsed field selection to a single line.
// Exported for testing; the Run loop above does the same inline (with
// -s short-circuit) to avoid allocating a throwaway string per skipped line.
func (c *Config) cutLine(line, outDelim string) string {
	if !strings.Contains(line, c.Delimiter) {
		return line
	}
	parts := strings.Split(line, c.Delimiter)
	var picked []string
	for _, r := range c.parsedRanges {
		hi := r.hi
		if hi == -1 || hi >= len(parts) {
			hi = len(parts) - 1
		}
		for idx := r.lo; idx <= hi; idx++ {
			if idx < len(parts) {
				picked = append(picked, parts[idx])
			}
		}
	}
	return strings.Join(picked, outDelim)
}
