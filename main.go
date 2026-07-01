package main

import (
	"fmt"
	"io"
	"os"

	"github.com/azhai/gx/cmd/cut"
	"github.com/azhai/gx/cmd/find"
	"github.com/azhai/gx/cmd/list"
	"github.com/azhai/gx/cmd/rename"
	"github.com/azhai/gx/cmd/replace"
	"github.com/azhai/gx/cmd/script"
	"github.com/azhai/gx/cmd/trans"
)

// Populated by -ldflags during build (see Makefile).
var (
	version = "dev"
	commit  = "unknown"
)

// Version returns the build version string, e.g. "1.2.0 (commit: abc1234)".
// Exposed for tests and for any future subcommand that needs to report it.
func Version() string {
	return fmt.Sprintf("%s (commit: %s)", version, commit)
}

func main() {
	os.Exit(runMain(os.Args, os.Stdout, os.Stderr))
}

// runMain is the testable entry point. It dispatches on argv[1] and
// returns the process exit code. Subcommand implementations own their
// own stdout/stderr (they call os.Stdout directly), so we only pass
// writers down for the top-level usage/version output. The writers are
// still useful for tests to capture and assert on those two paths.
//
// Exit codes:
//
//	0  success with output / matches
//	1  success without output (grep "no match" convention)
//	2  argument or runtime error
func runMain(argv []string, out, errOut io.Writer) int {
	if len(argv) < 2 {
		printUsageTo(out)
		return 2
	}

	command := argv[1]
	// Hand the per-command argv (without the gx binary name) down.
	os.Args = argv[1:]

	switch command {
	case "find":
		runFind()
	case "list":
		runList()
	case "replace":
		runReplace()
	case "rename":
		runRename()
	case "cut":
		return cut.Run()
	case "trans":
		return trans.Run()
	case "script":
		return script.Run()
	case "-h", "--help":
		printUsageTo(out)
		return 0
	case "-V", "--version":
		fmt.Fprintf(out, "gx version %s\n", Version())
		return 0
	default:
		fmt.Fprintf(errOut, "Unknown command: %s\n\n", command)
		printUsageTo(errOut)
		return 2
	}
	return 0
}

func printUsageTo(w io.Writer) {
	fmt.Fprintln(w, `gx - A handy text-processing utility (sed/awk style)

Usage: gx <command> [OPTIONS] [ARGS...]

Commands:
  find     Search for patterns in files (like grep)
  list     List files and directories with filters (like ls)
  replace  Search and replace text in files
  rename   Batch rename files
  cut      Extract fields from delimited text (like cut -f)
  trans    Apply text transformations (upper/lower/trim/...)
  script   Run a Tengo script over input (sed/awk style)

Global Flags:
  -h, --help       Show help
  -V, --version    Show version

Use "gx <command> --help" for command-specific options.

Examples:
  gx find "pattern" ./src
  gx list -g "*.go" ./src
  gx replace "old" "new" ./src -x
  gx rename "foo" "bar" -x
  cut -f 2 -d , file.csv
  cat file | gx trans upper
  echo "hi" | gx script -e 'text.to_upper(line)'`)
}

// runFind dispatches the find subcommand. Subcommand owns exit code via
// os.Exit (kept as-is to avoid touching its tested entry path).
func runFind() {
	config := find.NewConfig()
	if !config.ParseArgs() {
		os.Exit(2)
	}
	searcher, err := find.NewSearcher(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
	searcher.Search()
	if searcher.PrintResults() == 0 {
		os.Exit(1) // no match → grep convention
	}
}

// runList dispatches the list subcommand. Lists files and directories
// with filters (like ls). Returns 0 for results, 1 for no results, 2 for errors.
func runList() {
	config := list.NewConfig()
	if !config.ParseArgs() {
		os.Exit(2)
	}
	lister, err := list.NewLister(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
	os.Exit(lister.Run())
}

// runReplace dispatches the replace subcommand. Exit 1 covers both
// "no match found" and "argument error" (matches historical behavior of
// the subcommand).
func runReplace() {
	config := replace.NewConfig()
	if !config.ParseArgs() {
		os.Exit(1)
	}
	searcher, err := replace.NewSearcher(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if config.ReplaceSet {
		searcher.Replace()
	} else {
		searcher.Search()
		searcher.PrintResults()
	}
}

// runRename dispatches the rename subcommand. Returns the exit code
// from rename.Run (0 success, 1 no-op, 2 partial failure) to the caller.
func runRename() {
	config := rename.NewConfig()
	if !config.ParseArgs() {
		os.Exit(2)
	}
	renamer, err := rename.NewRenamer(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
	os.Exit(renamer.Run())
}
