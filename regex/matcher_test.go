package regex

import (
	"testing"
)

func TestNewMatcher(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "simple pattern",
			config: &Config{
				Pattern: "hello",
			},
			wantErr: false,
		},
		{
			name: "regex pattern",
			config: &Config{
				Pattern: `\d+`,
			},
			wantErr: false,
		},
		{
			name: "invalid regex",
			config: &Config{
				Pattern: `[invalid`,
			},
			wantErr: true,
		},
		{
			name: "ignore case",
			config: &Config{
				Pattern:    "HELLO",
				IgnoreCase: true,
			},
			wantErr: false,
		},
		{
			name: "fixed string",
			config: &Config{
				Pattern:     "[test]",
				FixedString: true,
			},
			wantErr: false,
		},
		{
			name: "empty pattern",
			config: &Config{
				Pattern: "",
			},
			wantErr: false,
		},
		{
			name: "complex regex",
			config: &Config{
				Pattern: `^(?:[a-zA-Z0-9._%+-]+)@(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$`,
			},
			wantErr: false,
		},
		{
			name: "unicode pattern",
			config: &Config{
				Pattern: `[\p{Han}]+`,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMatcher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && m == nil {
				t.Error("NewMatcher() returned nil matcher")
			}
		})
	}
}

func TestMatcher_MatchString(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: `hello`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		input string
		want  bool
	}{
		{"hello world", true},
		{"say hello", true},
		{"HELLO", false},
		{"world", false},
		{"", false},
		{"hellohello", true},
		{"helloworld", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.MatchString(tt.input); got != tt.want {
				t.Errorf("MatchString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatcher_MatchStringIgnoreCase(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: "hello", IgnoreCase: true})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		input string
		want  bool
	}{
		{"hello world", true},
		{"HELLO", true},
		{"HeLLo", true},
		{"world", false},
		{"hElLo WoRlD", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.MatchString(tt.input); got != tt.want {
				t.Errorf("MatchString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatcher_FindAll(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: `\d+`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"no match", "hello world", 0},
		{"one match", "number 123 here", 1},
		{"multiple matches", "1 and 2 and 3", 3},
		{"adjacent matches", "123456", 1},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := m.FindAll([]byte(tt.input))
			if len(matches) != tt.want {
				t.Errorf("FindAll(%q) returned %d matches, want %d", tt.input, len(matches), tt.want)
			}
		})
	}
}

func TestMatcher_FindAllString(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: `\w+`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"multiple words", "hello world test", 3},
		{"single word", "hello", 1},
		{"no match", "   ", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := m.FindAllString(tt.input)
			if len(matches) != tt.want {
				t.Errorf("FindAllString(%q) returned %d matches, want %d", tt.input, len(matches), tt.want)
			}
		})
	}
}

func TestMatcher_ReplaceAllString(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: `foo`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name    string
		input   string
		replace string
		want    string
	}{
		{"simple replace", "foo bar", "baz", "baz bar"},
		{"multiple replace", "foo foo foo", "x", "x x x"},
		{"no match", "hello world", "baz", "hello world"},
		{"empty replacement", "foo bar", "", " bar"},
		{"empty input", "", "baz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.ReplaceAllString(tt.input, tt.replace); got != tt.want {
				t.Errorf("ReplaceAllString(%q, %q) = %q, want %q", tt.input, tt.replace, got, tt.want)
			}
		})
	}
}

func TestMatcher_FixedString(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: "[test]", FixedString: true})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"exact match", "[test]", true},
		{"no brackets", "test", false},
		{"partial match", "[tes", false},
		{"contains pattern", "value [test] here", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.MatchString(tt.input); got != tt.want {
				t.Errorf("MatchString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatcher_FixedStringWithSpecialChars(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"dot literal", ".", "a.b", true},
		{"star literal", "*", "a*b", true},
		{"plus literal", "+", "a+b", true},
		{"question literal", "?", "a?b", true},
		{"caret literal", "^", "a^b", true},
		{"dollar literal", "$", "a$b", true},
		{"pipe literal", "|", "a|b", true},
		{"parentheses literal", "()", "a()b", true},
		{"brackets literal", "[]", "a[]b", true},
		{"braces literal", "{}", "a{}b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(&Config{Pattern: tt.pattern, FixedString: true})
			if err != nil {
				t.Fatalf("NewMatcher() error = %v", err)
			}
			if got := m.MatchString(tt.input); got != tt.want {
				t.Errorf("MatchString(%q) with pattern %q = %v, want %v", tt.input, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatcher_GetPattern(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		ignoreCase bool
		want       string
	}{
		{"simple pattern", "hello", false, "hello"},
		{"ignore case pattern", "hello", true, "(?i)hello"},
		{"regex pattern", `\d+`, false, `\d+`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(&Config{Pattern: tt.pattern, IgnoreCase: tt.ignoreCase})
			if err != nil {
				t.Fatalf("NewMatcher() error = %v", err)
			}

			pattern := m.GetPattern()
			if pattern != tt.want {
				t.Errorf("GetPattern() = %q, want %q", pattern, tt.want)
			}
		})
	}
}

func TestMatcher_ReplaceAll(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: `foo`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name    string
		input   []byte
		replace []byte
		want    []byte
	}{
		{"simple replace", []byte("foo bar"), []byte("baz"), []byte("baz bar")},
		{"multiple replace", []byte("foo foo foo"), []byte("x"), []byte("x x x")},
		{"no match", []byte("hello world"), []byte("baz"), []byte("hello world")},
		{"empty replacement", []byte("foo bar"), []byte(""), []byte(" bar")},
		{"empty input", []byte(""), []byte("baz"), []byte("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.ReplaceAll(tt.input, tt.replace); string(got) != string(tt.want) {
				t.Errorf("ReplaceAll(%q, %q) = %q, want %q", tt.input, tt.replace, got, tt.want)
			}
		})
	}
}

func TestMatcher_FixedStringReplace(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: "[test]", FixedString: true})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	input := []byte("value [test] and [test]")
	expected := []byte("value replace and replace")
	result := m.ReplaceAll(input, []byte("replace"))

	if string(result) != string(expected) {
		t.Errorf("ReplaceAll with FixedString failed.\nGot: %q\nWant: %q", result, expected)
	}
}

func TestMatcher_Match(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: `hello`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"match", []byte("hello world"), true},
		{"no match", []byte("world"), false},
		{"empty", []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.Match(tt.input); got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatcher_FindStringSubmatch(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: `(\w+)@(\w+)\.com`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"match with groups", "user@example.com", 3},
		{"no match", "hello world", 0},
		{"partial match", "contact: user@example.com", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := m.FindStringSubmatch(tt.input)
			if len(matches) != tt.want {
				t.Errorf("FindStringSubmatch(%q) returned %d matches, want %d", tt.input, len(matches), tt.want)
			}
		})
	}
}

func TestMatcher_ComplexPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"ipv4 address", `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "192.168.1.1", true},
		{"date format", `\d{4}-\d{2}-\d{2}`, "2024-01-15", true},
		{"url pattern", `https?://[^\s]+`, "https://example.com", true},
		{"phone number", `\+?\d{1,3}[-.\s]?\d{3}[-.\s]?\d{4}`, "+1-234-5678", true},
		{"hex color", `#[0-9a-fA-F]{6}`, "#FF5733", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(&Config{Pattern: tt.pattern})
			if err != nil {
				t.Fatalf("NewMatcher() error = %v", err)
			}
			if got := m.MatchString(tt.input); got != tt.want {
				t.Errorf("MatchString(%q) with pattern %q = %v, want %v", tt.input, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatcher_UnicodeSupport(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"chinese characters", `[\p{Han}]+`, "你好世界", true},
		{"japanese hiragana", `[\p{Hiragana}]+`, "こんにちは", true},
		{"emoji", `😀`, "😀", true},
		{"arabic", `[\p{Arabic}]+`, "مرحبا", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(&Config{Pattern: tt.pattern})
			if err != nil {
				t.Fatalf("NewMatcher() error = %v", err)
			}
			if got := m.MatchString(tt.input); got != tt.want {
				t.Errorf("MatchString(%q) with pattern %q = %v, want %v", tt.input, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatcher_MultilinePatterns(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: `^hello`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"starts with hello", "hello world", true},
		{"does not start with hello", "world hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.MatchString(tt.input); got != tt.want {
				t.Errorf("MatchString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
