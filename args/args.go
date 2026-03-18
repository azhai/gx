// Package args provides common argument parsing functionality for CLI commands.
// It supports both simple and advanced argument parsing with options and positional arguments.
package args

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// CommonConfig holds the common configuration shared between different commands.
// It contains fields that are used by both replace and rename commands.
type CommonConfig struct {
	// Pattern is the search pattern (regex or literal string)
	Pattern string
	// Replace is the replacement string
	Replace string
	// ReplaceSet indicates whether a replacement string was provided
	ReplaceSet bool
	// Paths are the directories or files to search
	Paths []string
	// IgnoreCase enables case-insensitive matching
	IgnoreCase bool
	// FilePattern is a glob pattern to filter files
	FilePattern string
	// DryRun enables dry-run mode (preview without making changes)
	DryRun bool
}

// OptionHandler is a function type that handles an option value.
// It receives the option value (empty string for flags) and the config to modify.
// Returns true if the option was handled successfully, false otherwise.
type OptionHandler func(value string, config *CommonConfig) bool

// Option represents a command-line option (flag or option with value).
type Option struct {
	// Short is the short form of the option (e.g., "-i")
	Short string
	// Long is the long form of the option (e.g., "--ignore-case")
	Long string
	// HasValue indicates whether the option requires a value
	HasValue bool
	// ValueName is the placeholder name for the value in help text
	ValueName string
	// Help is the help text for the option
	Help string
	// Handler is the function that processes the option
	Handler OptionHandler
}

// ParseCommon parses command-line arguments using the provided options.
// It handles options (both short and long forms), positional arguments,
// and automatically sets default paths if none are provided.
//
// The function processes arguments in order:
// 1. Options (with or without values)
// 2. Positional arguments (pattern, replace, paths)
//
// Returns true if parsing was successful, false otherwise.
func ParseCommon(args []string, config *CommonConfig, options []Option, printUsage func()) bool {
	// Build a map for quick option lookup
	optionMap := make(map[string]*Option)
	for i := range options {
		if options[i].Short != "" {
			optionMap[options[i].Short] = &options[i]
		}
		if options[i].Long != "" {
			optionMap[options[i].Long] = &options[i]
		}
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if opt, exists := optionMap[arg]; exists {
			// Handle option with value
			if opt.HasValue {
				if i+1 >= len(args) {
					fmt.Fprintf(os.Stderr, "Error: %s requires a value\n", arg)
					return false
				}
				if !opt.Handler(args[i+1], config) {
					return false
				}
				i++ // Skip the value
			} else {
				// Handle flag option
				if !opt.Handler("", config) {
					return false
				}
			}
		} else if arg == "-h" || arg == "--help" {
			printUsage()
			return false
		} else {
			// Handle positional arguments (including unknown options treated as positional)
			if config.Pattern == "" {
				config.Pattern = arg
			} else if !config.ReplaceSet {
				config.Replace = arg
				config.ReplaceSet = true
			} else {
				config.Paths = append(config.Paths, arg)
			}
		}
	}

	// Set default path if none provided
	if len(config.Paths) == 0 {
		config.Paths = []string{"."}
	}

	return true
}

// ParseSimple provides simplified argument parsing for common use cases.
// It handles special cases for 1 or 2 arguments before falling back to ParseCommon.
//
// Special cases:
//   - 1 argument: treated as pattern, path defaults to "."
//   - 2 arguments: if second arg is quoted, it's treated as replacement;
//     otherwise, it's treated as a path
//
// This allows for more intuitive command-line usage:
//
//	replace "pattern" /path     # search in /path
//	replace "pattern" "replace" # replace pattern with replace
func ParseSimple(args []string, config *CommonConfig, options []Option, printUsage func()) bool {
	var size int
	if size = len(args); size == 0 {
		printUsage()
		return false
	}

	// Check for help flag first
	if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
		printUsage()
		return false
	}

	// Check if any argument is an option
	optionMap := make(map[string]bool)
	for _, opt := range options {
		if opt.Short != "" {
			optionMap[opt.Short] = true
		}
		if opt.Long != "" {
			optionMap[opt.Long] = true
		}
	}

	hasOption := false
	for _, arg := range args {
		if optionMap[arg] {
			hasOption = true
			break
		}
	}

	// If there are options, use ParseCommon
	if hasOption {
		return ParseCommon(args, config, options, printUsage)
	}

	// Handle simple cases
	if size == 1 {
		config.Pattern = args[0]
		config.Paths = []string{"."}
		return true
	}

	if size == 2 {
		config.Pattern = args[0]
		// Check if second argument is quoted (treat as replacement)
		if args[1][0] == '"' && args[1][len(args[1])-1] == '"' {
			config.Replace = args[1][1 : len(args[1])-1]
			config.ReplaceSet = true
			config.Paths = []string{"."}
		} else {
			// Treat as path
			config.Paths = []string{args[1]}
		}
		return true
	}

	// Fall back to common parsing for complex cases
	return ParseCommon(args, config, options, printUsage)
}

// FormatOptions formats a slice of options into a help text string.
// It generates a formatted list of options with their short and long forms,
// value placeholders (if applicable), and help text.
//
// Example output:
//
//	-i, --ignore-case    Case insensitive matching
//	-g, --glob <PATTERN>    File glob pattern
func FormatOptions(options []Option) string {
	var result strings.Builder
	for _, opt := range options {
		if opt.Short != "" && opt.Long != "" {
			if opt.HasValue {
				result.WriteString(fmt.Sprintf("  %s, %s <%s>    %s\n", opt.Short, opt.Long, opt.ValueName, opt.Help))
			} else {
				result.WriteString(fmt.Sprintf("  %s, %s    %s\n", opt.Short, opt.Long, opt.Help))
			}
		} else if opt.Short != "" {
			if opt.HasValue {
				result.WriteString(fmt.Sprintf("  %s <%s>    %s\n", opt.Short, opt.ValueName, opt.Help))
			} else {
				result.WriteString(fmt.Sprintf("  %s    %s\n", opt.Short, opt.Help))
			}
		} else if opt.Long != "" {
			if opt.HasValue {
				result.WriteString(fmt.Sprintf("  %s <%s>    %s\n", opt.Long, opt.ValueName, opt.Help))
			} else {
				result.WriteString(fmt.Sprintf("  %s    %s\n", opt.Long, opt.Help))
			}
		}
	}
	return result.String()
}
