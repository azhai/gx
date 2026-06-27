// Package trans implements the gx trans subcommand: apply a built-in text
// transformation to each line of input (upper/lower/trim/squeeze/reverse).
//
// trans is a stdin/stdout filter. It reads one file (or stdin), applies
// the named transform to each line, and writes the result to stdout.
package trans

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/azhai/gx/args"
	"github.com/azhai/gx/pkg/stream"
)

// transforms is the registry of built-in transformations.
// Each takes a line (without trailing newline) and returns the transformed line.
var transforms = map[string]func(string) string{
	"upper":   strings.ToUpper,
	"lower":   strings.ToLower,
	"trim":    strings.TrimSpace,
	"squeeze": squeeze,
	"reverse": reverse,
}

// squeeze collapses runs of whitespace into a single space and trims ends.
func squeeze(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// reverse reverses the runes of a string.
func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// Config holds the configuration for the trans command.
type Config struct {
	args.CommonConfig
	transform string
}

// NewConfig creates a Config with trans defaults.
func NewConfig() *Config {
	return &Config{
		CommonConfig: args.CommonConfig{
			DryRun: true,
		},
	}
}

func (c *Config) getOptions() []args.Option {
	return []args.Option{}
}

// ParseArgs parses command-line arguments.
func (c *Config) ParseArgs() bool {
	return args.ParseSimple(os.Args[1:], &c.CommonConfig, c.getOptions(), c.printUsage)
}

func (c *Config) printUsage() {
	fmt.Println(`trans - Apply a text transformation to each line

Usage: trans <TRANSFORM> [FILE]

Transforms:
  upper    Uppercase
  lower    Lowercase
  trim     Strip leading/trailing whitespace
  squeeze  Collapse whitespace runs to a single space, trim ends
  reverse  Reverse the string (rune-wise)

When FILE is omitted or '-', reads from stdin.

Examples:
  gx trans upper file.txt
  cat file | gx trans lower
  gx trans squeeze -`)
}

// Run executes the trans command. Returns the process exit code:
//   - 0: success
//   - 2: argument/IO error
func Run() int {
	c := NewConfig()
	if !c.ParseArgs() {
		return 2
	}

	// ParseCommon distributes positional args as Pattern, Replace, Paths...
	// For trans: arg[0] = transform name (Pattern), arg[1] = input file.
	// - 1 positional:  trans upper       → Pattern="upper", Replace unset, Paths=["."]
	// - 2 positionals:  trans upper f.txt → ParseSimple size==2 → Pattern="upper", Paths=["f.txt"]
	// - 3+ positionals: trans upper f.txt → ParseCommon → Pattern="upper", Replace="f.txt", Paths=[...]
	c.transform = c.Pattern
	var paths []string
	switch {
	case c.ReplaceSet && c.Replace != "":
		paths = []string{c.Replace}
	case len(c.Paths) > 0 && c.Paths[0] != ".":
		paths = []string{c.Paths[0]}
	default:
		paths = []string{"-"} // stdin
	}

	fn, ok := transforms[c.transform]
	if !ok {
		fmt.Fprintf(os.Stderr, "trans: unknown transform %q (available: upper lower trim squeeze reverse)\n", c.transform)
		return 2
	}

	data, _, err := stream.ReadAll(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "trans: %v\n", err)
		return 2
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		w.WriteString(fn(scanner.Text()))
		w.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "trans: %v\n", err)
		return 2
	}
	return 0
}
