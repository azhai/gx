# gre

A fast file search and batch rename tool written in Go.

## Features

- **replace**: Fast file content search and replace (inspired by ripgrep)
  - Multi-threaded search and replace
  - Regular expression support
  - Colored output with highlighting
  - Line number display
  - File filtering by glob pattern
  - Binary file detection
  - Dry-run mode for safe preview

- **rename**: Batch file renaming tool (inspired by f2)
  - Regular expression matching
  - Capture group replacement ($1, $2, etc.)
  - Dry-run mode (preview before executing)
  - Conflict detection
  - Directory support
  - Force mode for resolving conflicts

## Installation

```bash
go install github.com/azhai/gre/cmd/replace@latest
go install github.com/azhai/gre/cmd/rename@latest
```

## Usage

### replace - File Content Search and Replace

```bash
# Basic search
replace "pattern"                    # Search in current directory
replace "pattern" /path/to/dir       # Search in specific directory

# Search options
replace -i "pattern"                 # Case insensitive search
replace -n "pattern"                 # Show line numbers (default)
replace -N "pattern"                 # Hide line numbers
replace --no-color "pattern"         # Disable colored output
replace -g "*.go" "func"             # Search only in Go files
replace -j 4 "pattern"               # Use 4 worker threads

# Replace (dry-run by default)
replace "TODO" "FIXME"               # Preview replace TODO with FIXME
replace "foo" "bar" -x               # Execute replace

# Examples
replace "TODO" src/                  # Search for TODO in src/
replace -i "error" -g "*.go"         # Case insensitive search in Go files
replace "TODO" src/ test/            # Search in multiple directories
replace "foo" "bar" -x               # Replace all 'foo' with 'bar'
```

### rename - Batch File Renaming

```bash
# Basic usage
rename "foo" "bar"                   # Replace 'foo' with 'bar' (dry-run)
rename "foo" "bar" /path/to/dir      # With specific directory

# Options
rename -f "pattern" "replace"        # Explicit find pattern
rename -r "replace"                  # Explicit replace string
rename -i "pattern" "replace"        # Case insensitive matching
rename -g "*.jpg" "pattern" "replace" # Filter by file pattern
rename -x "pattern" "replace"        # Execute (default: dry-run)
rename --force "pattern" "replace"   # Force rename with conflicts
rename -d "pattern" "replace"        # Include directories
rename -F "pattern" "replace"        # Treat pattern as literal string

# Examples
rename "foo" "bar"                   # Preview: replace 'foo' with 'bar'
rename "foo" "bar" -x                # Execute: replace 'foo' with 'bar'
rename -f "\.txt$" ".md" -x          # Change .txt to .md extension
rename -f "(\d+)" "prefix_$1" -x     # Add prefix to numbers
rename -i "IMG" "img" -g "*.jpg"     # Case conversion for jpg files
rename -f "^" "2024_" -x             # Add date prefix to all files
```

## Build from Source

```bash
git clone https://github.com/azhai/gre.git
cd gre
go build ./cmd/replace
go build ./cmd/rename
```

## Running Tests

```bash
go test ./... -v
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- **replace** inspired by [ripgrep](https://github.com/BurntSushi/ripgrep)
- **rename** inspired by [f2](https://github.com/ivek/Vim)
