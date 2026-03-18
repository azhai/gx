package main

import (
	"fmt"
	"os"

	"github.com/azhai/gx/cmd/rename"
	"github.com/azhai/gx/cmd/replace"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
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
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gx - A collection of file utilities

Usage: gx <command> [OPTIONS] [ARGS...]

Commands:
  find    Search for patterns in files (like grep)
  replace Search and replace text in files
  rename  Batch rename files

Use "gx <command> --help" for more information about a command.

Examples:
  gx find "pattern" ./src
  gx replace "old" "new" ./src -x
  gx rename "foo" "bar" -x`)
}

func runFind() {
	config := replace.NewConfig()
	if !config.ParseArgs() {
		os.Exit(1)
	}

	searcher, err := replace.NewSearcher(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	searcher.Search()
	searcher.PrintResults()
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
