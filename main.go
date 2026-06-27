package main

import (
	"fmt"
	"os"

	"github.com/azhai/gx/cmd/cut"
	"github.com/azhai/gx/cmd/find"
	"github.com/azhai/gx/cmd/list"
	"github.com/azhai/gx/cmd/rename"
	"github.com/azhai/gx/cmd/replace"
	"github.com/azhai/gx/cmd/trans"
)

// Populated by -ldflags during build (see Makefile).
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2) // no args → exit 2 (align with git/grep convention)
	}

	command := os.Args[1]
	os.Args = os.Args[1:]

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
		os.Exit(cut.Run())
	case "trans":
		os.Exit(trans.Run())
	case "-h", "--help":
		printUsage()
	case "-V", "--version":
		fmt.Printf("gx version %s (commit: %s)\n", version, commit)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println(`gx - A handy text-processing utility (sed/awk style)

Usage: gx <command> [OPTIONS] [ARGS...]

Commands:
  find     Search for patterns in files (like grep)
  list     List files containing matches (like grep -l)
  replace  Search and replace text in files
  rename   Batch rename files
  cut      Extract fields from delimited text (like cut -f)
  trans    Apply text transformations (upper/lower/trim/...)

Global Flags:
  -h, --help       Show help
  -V, --version    Show version

Use "gx <command> --help" for command-specific options.

Examples:
  gx find "pattern" ./src
  gx list "TODO" -g "*.go"
  gx replace "old" "new" ./src -x
  gx rename "foo" "bar" -x
  cut -f 2 -d , file.csv
  cat file | gx trans upper`)
}

func runFind() {
	config := find.NewConfig()
	if !config.ParseArgs() {
		os.Exit(2) // argument error → exit 2 (spec 4.4)
	}

	searcher, err := find.NewSearcher(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	searcher.Search()
	count := searcher.PrintResults()
	if count == 0 {
		os.Exit(1) // no match → exit 1 (grep convention)
	}
}

func runList() {
	config := list.NewConfig()
	if !config.ParseArgs() {
		os.Exit(2)
	}

	searcher, err := list.NewSearcher(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	searcher.Search()
	count := searcher.PrintResults()
	if count == 0 {
		os.Exit(1) // no matching files → exit 1 (grep -l convention)
	}
}

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

func runRename() {
	config := rename.NewConfig()
	if !config.ParseArgs() {
		os.Exit(2) // argument error → exit 2 (spec 4.4)
	}

	renamer, err := rename.NewRenamer(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2) // invalid pattern → exit 2 (spec 4.4)
	}

	os.Exit(renamer.Run())
}
