package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/azhai/rego/pkg/args"
	"github.com/azhai/rego/pkg/regex"
	"github.com/azhai/rego/pkg/walker"
)

// Config holds the configuration for the rename command.
// It extends args.CommonConfig with rename-specific options.
type Config struct {
	args.CommonConfig
	// Force indicates whether to proceed with conflicts
	Force bool
	// IncludeDir indicates whether to include directories
	IncludeDir bool
	// FixedString indicates whether to treat pattern as literal string
	FixedString bool
}

// RenameOp represents a single rename operation.
type RenameOp struct {
	// Original is the original file name
	Original string
	// NewName is the new file name
	NewName string
	// BaseDir is the directory containing the file
	BaseDir string
	// Conflict indicates whether this operation has a conflict
	Conflict bool
	// Error contains any error that occurred during the operation
	Error error
}

// Renamer performs batch file renaming operations.
type Renamer struct {
	// config holds the renamer configuration
	config *Config
	// matcher is the regex matcher used for pattern matching
	matcher *regex.Matcher
	// ops is the list of rename operations to perform
	ops []RenameOp
}

// NewConfig creates a new Config with default values.
// Default values:
//   - DryRun: true
func NewConfig() *Config {
	return &Config{
		CommonConfig: args.CommonConfig{
			DryRun: true,
		},
	}
}

func (c *Config) getOptions() []args.Option {
	return []args.Option{
		{
			Short: "-d", Long: "--include-dir",
			Help: "Include directories",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				c.IncludeDir = true
				return true
			},
		},
		{
			Short: "-F", Long: "--fixed-strings",
			Help: "Treat pattern as literal string",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				c.FixedString = true
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
			Short: "-g", Long: "--glob", HasValue: true, ValueName: "PATTERN",
			Help: "File glob pattern (e.g., \"*.jpg\")",
			Handler: func(v string, cfg *args.CommonConfig) bool {
				cfg.FilePattern = v
				return true
			},
		},
		{
			Short: "-i", Long: "--ignore-case",
			Help: "Case insensitive matching",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				cfg.IgnoreCase = true
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
			Help: "Execute the rename (default: dry-run)",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				cfg.DryRun = false
				return true
			},
		},
		{
			Long: "--force",
			Help: "Force rename even with conflicts",
			Handler: func(_ string, cfg *args.CommonConfig) bool {
				c.Force = true
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

// printUsage prints the usage information for the rename command.
func (c *Config) printUsage() {
	options := c.getOptions()
	fmt.Println(`rename - A batch file renaming tool inspired by f2

Usage: rename [OPTIONS] FIND [REPLACE] [PATH...]

Options:
` + args.FormatOptions(options) + `
Examples:
  rename "foo" "bar"                      Replace 'foo' with 'bar'
  rename "\.txt$" ".md"                   Change .txt to .md extension
  rename "(\d+)" "prefix_$1" -x           Add prefix to numbers
  rename -i "IMG" "img" -g "*.jpg"        Case conversion for jpg files
  rename "^" "2024_" -x                   Add date prefix to all files`)
}

// NewRenamer creates a new Renamer with the given configuration.
// It initializes the regex matcher and returns an error if the pattern is invalid.
func NewRenamer(config *Config) (*Renamer, error) {
	matcher, err := regex.NewMatcher(&regex.Config{
		Pattern:     config.Pattern,
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

// CollectFiles collects files to rename based on the pattern.
// It walks through the directories and finds files that match the pattern.
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

// DetectConflicts detects conflicts in the rename operations.
// A conflict occurs when:
// - Two files would be renamed to the same name
// - A file would be renamed to an existing file name
func (r *Renamer) DetectConflicts() {
	newPaths := make(map[string]int)
	for i, op := range r.ops {
		newPath := filepath.Join(op.BaseDir, op.NewName)
		key := strings.ToLower(newPath)
		if count, exists := newPaths[key]; exists {
			r.ops[i].Conflict = true
			r.ops[count].Conflict = true
		} else {
			newPaths[key] = i
		}

		fullPath := filepath.Join(op.BaseDir, op.NewName)
		fullPathLower := strings.ToLower(fullPath)
		originalPathLower := strings.ToLower(filepath.Join(op.BaseDir, op.Original))
		if fullPathLower != originalPathLower {
			if _, err := os.Stat(fullPath); err == nil {
				r.ops[i].Conflict = true
			}
		}
	}
}

// PrintPreview prints a preview of the rename operations.
// It shows the original and new file names, and highlights any conflicts.
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

// Execute executes the rename operations.
// In dry-run mode, it returns nil without making any changes.
// If there are conflicts and Force is not set, it returns an error.
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

// Run runs the complete rename workflow:
// 1. Collect files to rename
// 2. Detect conflicts
// 3. Print preview
// 4. Execute rename operations
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
