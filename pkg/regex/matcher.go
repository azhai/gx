package regex

import (
	"fmt"
	"regexp"
)

type Config struct {
	Pattern     string
	Replace     string
	IgnoreCase  bool
	Global      bool
	FixedString bool
}

type Matcher struct {
	regex   *regexp.Regexp
	config  *Config
	pattern string
}

func NewMatcher(config *Config) (*Matcher, error) {
	pattern := config.Pattern

	if config.FixedString {
		pattern = regexp.QuoteMeta(pattern)
	}

	if config.IgnoreCase {
		pattern = "(?i)" + pattern
	}

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

func (m *Matcher) FindAll(data []byte) [][]int {
	return m.regex.FindAllIndex(data, -1)
}

func (m *Matcher) FindAllString(s string) [][]int {
	return m.regex.FindAllStringIndex(s, -1)
}

func (m *Matcher) Match(data []byte) bool {
	return m.regex.Match(data)
}

func (m *Matcher) MatchString(s string) bool {
	return m.regex.MatchString(s)
}

func (m *Matcher) ReplaceAll(data []byte, replace []byte) []byte {
	return m.regex.ReplaceAll(data, replace)
}

func (m *Matcher) ReplaceAllString(s string, replace string) string {
	return m.regex.ReplaceAllString(s, replace)
}

func (m *Matcher) FindStringSubmatch(s string) []string {
	return m.regex.FindStringSubmatch(s)
}

func (m *Matcher) ReplaceWithSubmatches(s string, replacePattern string) string {
	result := replacePattern
	submatches := m.regex.FindStringSubmatchIndex(s)

	if submatches == nil {
		return s
	}

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

func (m *Matcher) GetPattern() string {
	return m.pattern
}
