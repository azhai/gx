package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/azhai/gre/pkg/regex"
	"github.com/azhai/gre/pkg/walker"
)

type Config struct {
	Find        string
	Replace     string
	ReplaceSet  bool
	Paths       []string
	IgnoreCase  bool
	FilePattern string
	DryRun      bool
	Force       bool
	IncludeDir  bool
	FixedString bool
}

type RenameOp struct {
	Original string
	NewName  string
	BaseDir  string
	Conflict bool
	Error    error
}

type Renamer struct {
	config  *Config
	matcher *regex.Matcher
	ops     []RenameOp
}

func NewConfig() *Config {
	return &Config{
		DryRun: true,
		Paths:  []string{},
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
		} else if arg == "-f" || arg == "--find" {
			if i+1 < len(args) {
				c.Find = args[i+1]
				i++
			}
		} else if arg == "-r" || arg == "--replace" {
			if i+1 < len(args) {
				c.Replace = args[i+1]
				c.ReplaceSet = true
				i++
			}
		} else if arg == "-g" || arg == "--glob" {
			if i+1 < len(args) {
				c.FilePattern = args[i+1]
				i++
			}
		} else if arg == "-x" || arg == "--exec" {
			c.DryRun = false
		} else if arg == "--force" {
			c.Force = true
		} else if arg == "-d" || arg == "--include-dir" {
			c.IncludeDir = true
		} else if arg == "-F" || arg == "--fixed-strings" {
			c.FixedString = true
		} else if arg == "-h" || arg == "--help" {
			c.printUsage()
			return false
		} else if len(arg) == 0 || arg[0] != '-' {
			if c.Find == "" {
				c.Find = arg
			} else if !c.ReplaceSet {
				c.Replace = arg
				c.ReplaceSet = true
			} else {
				c.Paths = append(c.Paths, arg)
			}
		}
		i++
	}

	if c.Find == "" {
		fmt.Fprintln(os.Stderr, "Error: find pattern is required")
		return false
	}

	if c.Replace == "" {
		c.Replace = ""
	}

	if len(c.Paths) == 0 {
		c.Paths = []string{"."}
	}

	return true
}

func (c *Config) printUsage() {
	fmt.Println(`rename - A batch file renaming tool inspired by f2

Usage: rename [OPTIONS] FIND [REPLACE] [PATH...]

Options:
  -f, --find <PATTERN>      Pattern to find (regex supported)
  -r, --replace <STRING>    Replacement string (use $1, $2 for groups)
  -i, --ignore-case         Case insensitive matching
  -g, --glob <PATTERN>      File glob pattern (e.g., "*.jpg")
  -x, --exec                Execute the rename (default: dry-run)
  --force                   Force rename even with conflicts
  -d, --include-dir         Include directories
  -F, --fixed-strings       Treat pattern as literal string
  -h, --help                Show this help message

Examples:
  rename "foo" "bar"                      Replace 'foo' with 'bar'
  rename -f "\.txt$" ".md"                Change .txt to .md extension
  rename -f "(\d+)" "prefix_$1" -x        Add prefix to numbers
  rename -i "IMG" "img" -g "*.jpg"        Case conversion for jpg files
  rename -f "^" "2024_" -x                Add date prefix to all files`)
}

func NewRenamer(config *Config) (*Renamer, error) {
	matcher, err := regex.NewMatcher(&regex.Config{
		Pattern:     config.Find,
		IgnoreCase:  config.IgnoreCase,
		FixedString: config.FixedString,
	})
	if err != nil {
		return nil, err
	}

	return &Renamer{
		config:  config,
		matcher: matcher,
		ops:     make([]RenameOp, 0),
	}, nil
}

func (r *Renamer) CollectFiles() {
	walkerConfig := walker.NewConfig()
	walkerConfig.Paths = r.config.Paths
	walkerConfig.FilePattern = r.config.FilePattern
	walkerConfig.IncludeDir = r.config.IncludeDir
	walkerConfig.SkipBinary = false

	fileWalker := walker.New(walkerConfig)
	files := fileWalker.Walk()

	for file := range files {
		if r.matcher.MatchString(file.Name) {
			newName := r.matcher.ReplaceAllString(file.Name, r.config.Replace)
			if newName != file.Name && newName != "" {
				r.ops = append(r.ops, RenameOp{
					Original: file.Name,
					NewName:  newName,
					BaseDir:  filepath.Dir(file.Path),
				})
			}
		}
	}

	sort.Slice(r.ops, func(i, j int) bool {
		return len(r.ops[i].Original) > len(r.ops[j].Original)
	})
}

func (r *Renamer) DetectConflicts() {
	newPaths := make(map[string]int)
	for i, op := range r.ops {
		newPath := filepath.Join(op.BaseDir, op.NewName)
		if count, exists := newPaths[newPath]; exists {
			r.ops[i].Conflict = true
			r.ops[count].Conflict = true
		} else {
			newPaths[newPath] = i
		}

		fullPath := filepath.Join(op.BaseDir, op.NewName)
		if _, err := os.Stat(fullPath); err == nil {
			r.ops[i].Conflict = true
		}
	}
}

func (r *Renamer) PrintPreview() {
	if len(r.ops) == 0 {
		fmt.Println("No files to rename")
		return
	}

	fmt.Printf("\n%d file(s) to rename:\n\n", len(r.ops))

	conflictCount := 0
	for _, op := range r.ops {
		originalPath := filepath.Join(op.BaseDir, op.Original)
		newPath := filepath.Join(op.BaseDir, op.NewName)

		if op.Conflict {
			conflictCount++
			fmt.Printf("  \x1b[33mCONFLICT\x1b[0m: %s\n", originalPath)
			fmt.Printf("           -> \x1b[31m%s\x1b[0m\n", newPath)
		} else {
			fmt.Printf("  \x1b[32m%s\x1b[0m\n", originalPath)
			fmt.Printf("  -> \x1b[36m%s\x1b[0m\n", newPath)
		}
	}

	if conflictCount > 0 {
		fmt.Printf("\n\x1b[33m%d conflict(s) detected\x1b[0m\n", conflictCount)
		if !r.config.Force {
			fmt.Println("Use --force to proceed with conflicts")
		}
	}

	if r.config.DryRun {
		fmt.Println("\n\x1b[33mDry run - no changes made\x1b[0m")
		fmt.Println("Use -x or --exec to apply changes")
	}
}

func (r *Renamer) Execute() error {
	if r.config.DryRun {
		return nil
	}

	conflictCount := 0
	for _, op := range r.ops {
		if op.Conflict {
			conflictCount++
		}
	}

	if conflictCount > 0 && !r.config.Force {
		return fmt.Errorf("conflicts detected, use --force to proceed")
	}

	executed := 0
	for _, op := range r.ops {
		if op.Conflict && !r.config.Force {
			continue
		}

		oldPath := filepath.Join(op.BaseDir, op.Original)
		newPath := filepath.Join(op.BaseDir, op.NewName)

		err := os.Rename(oldPath, newPath)
		if err != nil {
			fmt.Printf("\x1b[31mError: %s -> %s: %v\x1b[0m\n", oldPath, newPath, err)
			continue
		}

		executed++
	}

	fmt.Printf("\n\x1b[32mSuccessfully renamed %d file(s)\x1b[0m\n", executed)
	return nil
}

func (r *Renamer) Run() {
	r.CollectFiles()
	r.DetectConflicts()
	r.PrintPreview()

	if err := r.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[31mError: %v\x1b[0m\n", err)
		os.Exit(1)
	}
}

func main() {
	config := NewConfig()
	if !config.ParseArgs() {
		os.Exit(1)
	}

	renamer, err := NewRenamer(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	renamer.Run()
}
