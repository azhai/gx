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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.ReplaceAllString(tt.input, tt.replace); got != tt.want {
				t.Errorf("ReplaceAllString(%q, %q) = %q, want %q", tt.input, tt.replace, got, tt.want)
			}
		})
	}
}

func TestMatcher_ReplaceWithGroups(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: `(\w+)@(\w+)\.com`})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name    string
		input   string
		replace string
		want    string
	}{
		{"email replacement", "user@example.com", "$1 at $2", "user at example"},
		{"no match", "hello world", "$1", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.ReplaceWithSubmatches(tt.input, tt.replace); got != tt.want {
				t.Errorf("ReplaceWithSubmatches(%q, %q) = %q, want %q", tt.input, tt.replace, got, tt.want)
			}
		})
	}
}

func TestMatcher_FixedString(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: "[test]", FixedString: true})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	if !m.MatchString("[test]") {
		t.Error("FixedString mode should match literal [test]")
	}
	if m.MatchString("test") {
		t.Error("FixedString mode should not match 'test' without brackets")
	}
}

func TestMatcher_GetPattern(t *testing.T) {
	m, err := NewMatcher(&Config{Pattern: "hello", IgnoreCase: true})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	pattern := m.GetPattern()
	if pattern != "(?i)hello" {
		t.Errorf("GetPattern() = %q, want %q", pattern, "(?i)hello")
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
