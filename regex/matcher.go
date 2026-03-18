// Package regex provides regular expression matching and replacement functionality.
// It wraps the standard regexp package with additional features like case-insensitive
// matching and fixed string support.
package regex

import (
	"fmt"
	"regexp"
)

// Config holds the configuration for a Matcher.
type Config struct {
	// Pattern is the regular expression pattern to match
	Pattern string
	// Replace is the replacement string (not used in Matcher, kept for compatibility)
	Replace string
	// IgnoreCase enables case-insensitive matching
	IgnoreCase bool
	// Global enables global matching (match all occurrences)
	Global bool
	// FixedString treats the pattern as a literal string instead of regex
	FixedString bool
}

// Matcher provides regular expression matching and replacement operations.
// It wraps regexp.Regexp with additional functionality.
type Matcher struct {
	// regex is the compiled regular expression
	regex *regexp.Regexp
	// config holds the matcher configuration
	config *Config
	// pattern is the processed pattern string (may include flags)
	pattern string
}

// NewMatcher creates a new Matcher with the given configuration.
// It compiles the pattern and applies any necessary transformations
// (e.g., case-insensitive flag, literal string quoting).
//
// Returns an error if the pattern is invalid.
func NewMatcher(config *Config) (*Matcher, error) {
	pattern := config.Pattern

	// Quote the pattern if it should be treated as a literal string
	if config.FixedString {
		pattern = regexp.QuoteMeta(pattern)
	}

	// Add case-insensitive flag if enabled
	if config.IgnoreCase {
		pattern = "(?i)" + pattern
	}

	// Compile the regex
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return &Matcher{
		regex:   regex,
		config:  config,
		pattern: pattern,
	}, nil
}

// FindAll finds all matches in the given byte slice.
// It returns a slice of [start, end] index pairs for each match.
// Returns nil if no matches are found.
func (m *Matcher) FindAll(data []byte) [][]int {
	return m.regex.FindAllIndex(data, -1)
}

// FindAllString finds all matches in the given string.
// It returns a slice of [start, end] index pairs for each match.
// Returns nil if no matches are found.
func (m *Matcher) FindAllString(s string) [][]int {
	return m.regex.FindAllStringIndex(s, -1)
}

// Match checks whether the byte slice matches the pattern.
// It returns true if there is at least one match.
func (m *Matcher) Match(data []byte) bool {
	return m.regex.Match(data)
}

// MatchString checks whether the string matches the pattern.
// It returns true if there is at least one match.
func (m *Matcher) MatchString(s string) bool {
	return m.regex.MatchString(s)
}

// ReplaceAll replaces all matches in the byte slice with the replacement.
// The replacement can contain $1, $2, etc. to reference captured groups.
func (m *Matcher) ReplaceAll(data []byte, replace []byte) []byte {
	return m.regex.ReplaceAll(data, replace)
}

// ReplaceAllString replaces all matches in the string with the replacement.
// The replacement can contain $1, $2, etc. to reference captured groups.
func (m *Matcher) ReplaceAllString(s string, replace string) string {
	return m.regex.ReplaceAllString(s, replace)
}

// FindStringSubmatch returns a slice of strings holding the text of the
// leftmost match of the regular expression in s and the matches of any
// subexpressions. The return value is nil if no match is found.
func (m *Matcher) FindStringSubmatch(s string) []string {
	return m.regex.FindStringSubmatch(s)
}

// ReplaceWithSubmatches replaces matches in the string using a replacement pattern.
// The replacement pattern can contain $1, $2, etc. to reference captured groups.
// This is a custom implementation that handles submatch replacement.
//
// Example:
//
//	pattern: "(\w+)@(\w+)"
//	input: "user@domain"
//	replacePattern: "$2:$1"
//	result: "domain:user"
func (m *Matcher) ReplaceWithSubmatches(s string, replacePattern string) string {
	result := replacePattern
	submatches := m.regex.FindStringSubmatchIndex(s)

	if submatches == nil {
		return s
	}

	// Replace group placeholders from highest to lowest to avoid index shifting
	for i := len(submatches) - 2; i >= 2; i -= 2 {
		if submatches[i] != -1 {
			groupNum := (i-2)/2 + 1
			placeholder := fmt.Sprintf("$%d", groupNum)
			value := s[submatches[i]:submatches[i+1]]
			result = regexp.MustCompile(regexp.QuoteMeta(placeholder)).ReplaceAllString(result, value)
		}
	}

	return result
}

// GetPattern returns the processed pattern string (may include flags).
func (m *Matcher) GetPattern() string {
	return m.pattern
}
