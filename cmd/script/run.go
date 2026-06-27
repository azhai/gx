// Package script implements the gx script subcommand: run a Tengo script
// over input text in sed (line) or awk (whole-file) style.
//
// Line mode (default): the script runs once per input line with `line`,
// `lineno`, `filename` injected. The result variable `__out` decides what
// is emitted: string → printed + newline; undefined/false/nil → line
// skipped (filtering); other → fmt.Sprint().
//
// File mode (--whole): the script runs once per file with `content` and
// `filename` injected; `__out` (string) is written to stdout, undefined
// skips the file.
//
// `-e EXPR` wraps the expression as `__out := (EXPR)` so the trailing
// expression becomes the per-line/per-file output. `-f FILE` loads a
// full script that must assign `__out` itself.
//
// Security: only pure-computation Tengo stdlib modules are loaded by
// default. The whitelist (`safeModules` below) is `fmt / text / json /
// math / times / base64 / hex / enum`. Note that Tengo bundles strings,
// strconv and regexp into the single `text` module (snake_case API:
// to_upper, contains, atoi, re_match, ...). `--unsafe` loads the full
// stdlib (`os`, `exec`, `file`, `bin`, ...).
package script

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/azhai/gx/args"
	"github.com/azhai/gx/stream"
	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// safeModules is the whitelist of pure-computation Tengo stdlib modules.
// Tengo does NOT ship separate strings/strconv/regexp/time modules — those
// are bundled into `text` (strings+strconv+regexp, snake_case: to_upper,
// contains, atoi, re_match, …) and `times` (time). See Tengo stdlib docs.
var safeModules = []string{
	"fmt", "text", "json", "math", "times", "base64", "hex", "enum",
}

// Config holds configuration for the script command.
type Config struct {
	args.CommonConfig

	expr    string
	file    string
	whole   bool
	unsafe  bool
	timeout time.Duration
}

// NewConfig creates a Config with script defaults.
func NewConfig() *Config {
	return &Config{
		CommonConfig: args.CommonConfig{
			DryRun: true,
		},
		timeout: time.Second,
	}
}

func (c *Config) getOptions() []args.Option {
	return []args.Option{
		{Short: "-e", Long: "--expr", HasValue: true, ValueName: "EXPR",
			Help:    "inline expression (wrapped as __out := (EXPR))",
			Handler: func(v string, _ *args.CommonConfig) bool { c.expr = v; return true }},
		{Short: "-f", Long: "--file", HasValue: true, ValueName: "FILE",
			Help:    "path to a .tengo script file",
			Handler: func(v string, _ *args.CommonConfig) bool { c.file = v; return true }},
		{Long: "--whole", Help: "whole-file mode (awk-style): inject `content` instead of per-line",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.whole = true; return true }},
		{Long: "--unsafe", Help: "enable ALL Tengo stdlib modules (os, rand, ...) — use with care",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.unsafe = true; return true }},
		{Long: "--timeout", HasValue: true, ValueName: "D",
			Help: "per-execution timeout (e.g. 500ms, 2s); default 1s",
			Handler: func(v string, _ *args.CommonConfig) bool {
				d, err := time.ParseDuration(v)
				if err != nil {
					return false
				}
				c.timeout = d
				return true
			}},
	}
}

// ParseArgs parses command-line arguments.
func (c *Config) ParseArgs() bool {
	return args.ParseSimple(os.Args[1:], &c.CommonConfig, c.getOptions(), c.printUsage)
}

func (c *Config) printUsage() {
	fmt.Println(`script - Run a Tengo script over text (sed/awk style)

Usage: script (-e EXPR | -f FILE) [--whole] [--unsafe] [--timeout D] [FILE...]

Modes:
  line (default)  Per-line: injects line, lineno, filename. Result in __out.
  --whole          Per-file: injects content, filename. Result in __out.

Result convention (__out):
  string           emitted + newline
  undefined/false  line/file skipped (filtering)
  other            fmt.Sprint() then emitted + newline

Security:
  Default imports only fmt/text/json/math/times/base64/hex/enum.
  --unsafe enables os and rand too. Dangerous.

Examples:
  echo "hi" | gx script -e 'text.to_upper(line)'
  gx script -e 'lineno % 2 == 0 ? text.to_upper(line) : line' file
  gx script --whole -f agg.tengo *.log
  gx script -e 'line + "!"' input.txt`)
}

// loadSource resolves the script source from -e or -f.
// -e takes precedence; -f is read as a file. Exactly one is required.
// For -e, the expression is wrapped so its value flows into __out and the
// safe stdlib modules are pre-imported (so `strings.ToUpper(line)` works
// without an explicit `strings := import("strings")` line).
func (c *Config) loadSource() (string, error) {
	if c.expr != "" {
		var b strings.Builder
		for _, name := range safeModules {
			fmt.Fprintf(&b, "%s := import(%q)\n", name, name)
		}
		b.WriteString("__out := (")
		b.WriteString(c.expr)
		b.WriteString(")")
		return b.String(), nil
	}
	if c.file == "" {
		return "", fmt.Errorf("script: exactly one of -e or -f is required")
	}
	data, err := os.ReadFile(c.file)
	if err != nil {
		return "", fmt.Errorf("script: read %s: %w", c.file, err)
	}
	return string(data), nil
}

// buildModuleMap returns the Tengo module map per the security policy.
func (c *Config) buildModuleMap() *tengo.ModuleMap {
	if c.unsafe {
		return stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	}
	return stdlib.GetModuleMap(safeModules...)
}

// declareGlobals pre-declares the input variables the script will read so
// Tengo's compiler resolves references like `line` / `content`. The values
// set here are placeholders; runtime values are injected per iteration via
// Compiled.Clone + Compiled.Set.
func declareGlobals(s *tengo.Script, whole bool) error {
	if whole {
		if err := s.Add("content", ""); err != nil {
			return err
		}
		return s.Add("filename", "")
	}
	if err := s.Add("line", ""); err != nil {
		return err
	}
	if err := s.Add("lineno", int64(0)); err != nil {
		return err
	}
	return s.Add("filename", "")
}

// Run executes the script command. Returns exit code 0 (success with output),
// 1 (success, no output produced), or 2 (compile/IO error).
func Run() int {
	c := NewConfig()
	if !c.ParseArgs() {
		return 2
	}
	if c.timeout <= 0 {
		c.timeout = time.Second
	}

	src, err := c.loadSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "script: %v\n", err)
		return 2
	}

	// Compile once. Set the module map before compiling so imports resolve.
	// Declare the variables the script will reference (line, lineno,
	// filename in line mode; content, filename in whole mode). Tengo
	// requires them to exist at compile time even though we'll set
	// per-iteration values via Compiled.Clone + Set at runtime.
	s := tengo.NewScript([]byte(src))
	s.SetImports(c.buildModuleMap())
	if err := declareGlobals(s, c.whole); err != nil {
		fmt.Fprintf(os.Stderr, "script: declare globals: %v\n", err)
		return 2
	}
	compiled, err := s.Compile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "script: compile: %v\n", err)
		return 2
	}

	// Resolve input paths. ParseCommon stuffed the first positional into
	// Pattern (cut's pattern) and any extras into Paths; for script every
	// positional is an input path. Paths may contain the default "." set
	// by ParseCommon when no positional was given — drop it so we fall
	// through to stdin.
	var paths []string
	if c.Pattern != "" {
		paths = append(paths, c.Pattern)
	}
	for _, p := range c.Paths {
		if p == "." {
			continue
		}
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		paths = []string{"-"} // stdin
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	var produced int
	if c.whole {
		produced, err = c.runFileMode(compiled, paths, w)
	} else {
		produced, err = c.runLineMode(compiled, paths, w)
	}
	if err != nil {
		w.Flush()
		fmt.Fprintf(os.Stderr, "script: %v\n", err)
		return 2
	}
	if produced == 0 {
		return 1 // no output produced (grep-like "no match")
	}
	return 0
}

// runLineMode runs the compiled script once per line of input.
func (c *Config) runLineMode(compiled *tengo.Compiled, paths []string, w *bufio.Writer) (int, error) {
	data, name, err := stream.ReadAll(paths)
	if err != nil {
		return 0, err
	}
	if name == "" || name == "-" {
		name = "<stdin>"
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineno := 0
	produced := 0
	var errs []string
	for scanner.Scan() {
		lineno++
		out, skip, runErr := c.execOnce(compiled, map[string]tengo.Object{
			"line":     &tengo.String{Value: scanner.Text()},
			"lineno":   &tengo.Int{Value: int64(lineno)},
			"filename": &tengo.String{Value: name},
		})
		if runErr != nil {
			errs = append(errs, fmt.Sprintf("line %d: %v", lineno, runErr))
			if len(errs) >= 10 {
				return produced, fmt.Errorf("%d+ runtime errors (showing first 10):\n  %s",
					len(errs), strings.Join(errs, "\n  "))
			}
			continue
		}
		if skip {
			continue
		}
		w.WriteString(out)
		w.WriteByte('\n')
		produced++
	}
	if err := scanner.Err(); err != nil {
		return produced, err
	}
	if len(errs) > 0 {
		return produced, fmt.Errorf("%d runtime errors:\n  %s",
			len(errs), strings.Join(errs, "\n  "))
	}
	return produced, nil
}

// runFileMode runs the compiled script once per input file.
func (c *Config) runFileMode(compiled *tengo.Compiled, paths []string, w *bufio.Writer) (int, error) {
	var errs []string
	produced := 0
	for _, p := range paths {
		content, name, err := readOne(p)
		if err != nil {
			return produced, err
		}
		out, skip, runErr := c.execOnce(compiled, map[string]tengo.Object{
			"content":  &tengo.String{Value: content},
			"filename": &tengo.String{Value: name},
		})
		if runErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, runErr))
			if len(errs) >= 10 {
				return produced, fmt.Errorf("%d+ runtime errors (showing first 10):\n  %s",
					len(errs), strings.Join(errs, "\n  "))
			}
			continue
		}
		if skip {
			continue
		}
		w.WriteString(out)
		produced++
	}
	if len(errs) > 0 {
		return produced, fmt.Errorf("%d runtime errors:\n  %s",
			len(errs), strings.Join(errs, "\n  "))
	}
	return produced, nil
}

// execOnce clones the compiled script, sets variables, runs with timeout,
// and extracts __out. Returns (output, skip, err). skip=true when __out
// is undefined/false.
func (c *Config) execOnce(compiled *tengo.Compiled, vars map[string]tengo.Object) (string, bool, error) {
	cloned := compiled.Clone()
	for k, v := range vars {
		if err := cloned.Set(k, v); err != nil {
			return "", false, fmt.Errorf("set %s: %w", k, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	if err := cloned.RunContext(ctx); err != nil {
		return "", false, err
	}

	v := cloned.Get("__out")
	val := v.Value()
	if val == nil {
		return "", true, nil
	}
	switch o := val.(type) {
	case nil:
		return "", true, nil
	case bool:
		if !o {
			return "", true, nil // false → skip
		}
		return "true", false, nil // true → emit "true"
	case string:
		return o, false, nil
	case *tengo.Undefined:
		return "", true, nil
	case *tengo.String:
		return o.Value, false, nil
	case *tengo.Bool:
		// Bool.value is private; use IsFalsy + String().
		if o.IsFalsy() {
			return "", true, nil
		}
		return o.String(), false, nil
	default:
		return fmt.Sprint(o), false, nil
	}
}

// readOne reads a single input file (or stdin) with its display name.
func readOne(path string) (string, string, error) {
	if path == "-" || path == "" {
		data, name, err := stream.ReadAll([]string{"-"})
		if err != nil {
			return "", "", err
		}
		if name == "" || name == "-" {
			name = "<stdin>"
		}
		return string(data), name, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return string(data), path, nil
}
