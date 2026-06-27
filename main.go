package main

import (
	"fmt"
	"os"

	"github.com/azhai/gx/cmd/find"
	"github.com/azhai/gx/cmd/rename"
	"github.com/azhai/gx/cmd/replace"
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
	case "replace":
		runReplace()
	case "rename":
		runRename()
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
  replace  Search and replace text in files
  rename   Batch rename files

Global Flags:
  -h, --help       Show help
  -V, --version    Show version

Use "gx <command> --help" for command-specific options.

Examples:
  gx find "pattern" ./src
  gx replace "old" "new" ./src -x
  gx rename "foo" "bar" -x`)
}

func runFind() {
	config := find.NewConfig()
	if !config.ParseArgs() {
		os.Exit(1)
	}

	searcher, err := find.NewSearcher(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	searcher.Search()
	count := searcher.PrintResults()
	if count == 0 {
		os.Exit(1) // exit 1 when no matches (grep convention)
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
		os.Exit(1)
	}

	renamer, err := rename.NewRenamer(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	renamer.Run()
}
