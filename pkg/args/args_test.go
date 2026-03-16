package args

import (
	"testing"
)

func TestParseSimple(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		options            []Option
		expectedPattern    string
		expectedReplace    string
		expectedReplaceSet bool
		expectedPaths      []string
		expectedDryRun     bool
	}{
		{
			name:            "single argument - pattern only",
			args:            []string{"TODO"},
			expectedPattern: "TODO",
			expectedPaths:   []string{"."},
			expectedDryRun:  true,
		},
		{
			name:            "two arguments - pattern and path",
			args:            []string{"TODO", "/src"},
			expectedPattern: "TODO",
			expectedPaths:   []string{"/src"},
			expectedDryRun:  true,
		},
		{
			name:               "two arguments - pattern and quoted replacement",
			args:               []string{"TODO", `"FIXME"`},
			expectedPattern:    "TODO",
			expectedReplace:    "FIXME",
			expectedReplaceSet: true,
			expectedPaths:      []string{"."},
			expectedDryRun:     true,
		},
		{
			name:               "three arguments - pattern, replace, path",
			args:               []string{"TODO", "FIXME", "/src"},
			expectedPattern:    "TODO",
			expectedReplace:    "FIXME",
			expectedReplaceSet: true,
			expectedPaths:      []string{"/src"},
			expectedDryRun:     true,
		},
		{
			name:            "with find option and value",
			args:            []string{"-f", "TODO"},
			expectedPattern: "TODO",
			expectedPaths:   []string{"."},
			expectedDryRun:  true,
		},
		{
			name:               "with find and replace options",
			args:               []string{"-f", "TODO", "-r", "FIXME"},
			expectedPattern:    "TODO",
			expectedReplace:    "FIXME",
			expectedReplaceSet: true,
			expectedPaths:      []string{"."},
			expectedDryRun:     true,
		},
		{
			name:               "with exec option",
			args:               []string{"TODO", "FIXME", "-x"},
			expectedPattern:    "TODO",
			expectedReplace:    "FIXME",
			expectedReplaceSet: true,
			expectedPaths:      []string{"."},
			expectedDryRun:     false,
		},
		{
			name: "with ignore case option",
			args: []string{"-i", "TODO"},
			options: []Option{
				{
					Short: "-i", Long: "--ignore-case",
					Help: "Case insensitive",
					Handler: func(_ string, cfg *CommonConfig) bool {
						cfg.IgnoreCase = true
						return true
					},
				},
			},
			expectedPattern: "TODO",
			expectedPaths:   []string{"."},
			expectedDryRun:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CommonConfig{
				DryRun: true,
			}

			options := tt.options
			if len(options) == 0 {
				options = []Option{
					{
						Short: "-f", Long: "--find", HasValue: true, ValueName: "PATTERN",
						Help: "Pattern to find",
						Handler: func(v string, cfg *CommonConfig) bool {
							cfg.Pattern = v
							return true
						},
					},
					{
						Short: "-r", Long: "--replace", HasValue: true, ValueName: "STRING",
						Help: "Replacement string",
						Handler: func(v string, cfg *CommonConfig) bool {
							cfg.Replace = v
							cfg.ReplaceSet = true
							return true
						},
					},
					{
						Short: "-x", Long: "--exec",
						Help: "Execute",
						Handler: func(_ string, cfg *CommonConfig) bool {
							cfg.DryRun = false
							return true
						},
					},
				}
			}

			result := ParseSimple(tt.args, config, options, func() {})
			if !result {
				t.Errorf("ParseSimple returned false")
				return
			}

			if config.Pattern != tt.expectedPattern {
				t.Errorf("expected pattern %q, got %q", tt.expectedPattern, config.Pattern)
			}

			if config.Replace != tt.expectedReplace {
				t.Errorf("expected replace %q, got %q", tt.expectedReplace, config.Replace)
			}

			if config.ReplaceSet != tt.expectedReplaceSet {
				t.Errorf("expected replaceSet %v, got %v", tt.expectedReplaceSet, config.ReplaceSet)
			}

			if len(config.Paths) != len(tt.expectedPaths) {
				t.Errorf("expected %d paths, got %d", len(tt.expectedPaths), len(config.Paths))
				return
			}

			for i, path := range config.Paths {
				if path != tt.expectedPaths[i] {
					t.Errorf("expected path %q at index %d, got %q", tt.expectedPaths[i], i, path)
				}
			}

			if config.DryRun != tt.expectedDryRun {
				t.Errorf("expected dry-run %v, got %v", tt.expectedDryRun, config.DryRun)
			}
		})
	}
}

func TestParseCommon(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		options            []Option
		expectedPattern    string
		expectedReplace    string
		expectedReplaceSet bool
		expectedPaths      []string
		expectedDryRun     bool
	}{
		{
			name:               "with multiple options",
			args:               []string{"-f", "TODO", "-r", "FIXME", "-x", "/src"},
			expectedPattern:    "TODO",
			expectedReplace:    "FIXME",
			expectedReplaceSet: true,
			expectedPaths:      []string{"/src"},
			expectedDryRun:     false,
		},
		{
			name: "with glob option",
			args: []string{"-g", "*.go", "TODO"},
			options: []Option{
				{
					Short: "-g", Long: "--glob", HasValue: true, ValueName: "PATTERN",
					Help: "Glob pattern",
					Handler: func(v string, cfg *CommonConfig) bool {
						cfg.FilePattern = v
						return true
					},
				},
			},
			expectedPattern: "TODO",
			expectedPaths:   []string{"."},
			expectedDryRun:  true,
		},
		{
			name:               "multiple paths",
			args:               []string{"TODO", "FIXME", "/src", "/test"},
			expectedPattern:    "TODO",
			expectedReplace:    "FIXME",
			expectedReplaceSet: true,
			expectedPaths:      []string{"/src", "/test"},
			expectedDryRun:     true,
		},
		{
			name:            "pattern only with default path",
			args:            []string{"TODO"},
			expectedPattern: "TODO",
			expectedPaths:   []string{"."},
			expectedDryRun:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CommonConfig{
				DryRun: true,
			}

			options := tt.options
			if len(options) == 0 {
				options = []Option{
					{
						Short: "-f", Long: "--find", HasValue: true, ValueName: "PATTERN",
						Help: "Pattern to find",
						Handler: func(v string, cfg *CommonConfig) bool {
							cfg.Pattern = v
							return true
						},
					},
					{
						Short: "-r", Long: "--replace", HasValue: true, ValueName: "STRING",
						Help: "Replacement string",
						Handler: func(v string, cfg *CommonConfig) bool {
							cfg.Replace = v
							cfg.ReplaceSet = true
							return true
						},
					},
					{
						Short: "-g", Long: "--glob", HasValue: true, ValueName: "PATTERN",
						Help: "Glob pattern",
						Handler: func(v string, cfg *CommonConfig) bool {
							cfg.FilePattern = v
							return true
						},
					},
					{
						Short: "-x", Long: "--exec",
						Help: "Execute",
						Handler: func(_ string, cfg *CommonConfig) bool {
							cfg.DryRun = false
							return true
						},
					},
				}
			}

			result := ParseCommon(tt.args, config, options, func() {})
			if !result {
				t.Errorf("ParseCommon returned false")
				return
			}

			if config.Pattern != tt.expectedPattern {
				t.Errorf("expected pattern %q, got %q", tt.expectedPattern, config.Pattern)
			}

			if config.Replace != tt.expectedReplace {
				t.Errorf("expected replace %q, got %q", tt.expectedReplace, config.Replace)
			}

			if config.ReplaceSet != tt.expectedReplaceSet {
				t.Errorf("expected replaceSet %v, got %v", tt.expectedReplaceSet, config.ReplaceSet)
			}

			if len(config.Paths) != len(tt.expectedPaths) {
				t.Errorf("expected %d paths, got %d", len(tt.expectedPaths), len(config.Paths))
				return
			}

			for i, path := range config.Paths {
				if path != tt.expectedPaths[i] {
					t.Errorf("expected path %q at index %d, got %q", tt.expectedPaths[i], i, path)
				}
			}

			if config.DryRun != tt.expectedDryRun {
				t.Errorf("expected dry-run %v, got %v", tt.expectedDryRun, config.DryRun)
			}
		})
	}
}

func TestFormatOptions(t *testing.T) {
	options := []Option{
		{Short: "-f", Long: "--find", HasValue: true, ValueName: "PATTERN", Help: "Pattern to find"},
		{Short: "-r", Long: "--replace", HasValue: true, ValueName: "STRING", Help: "Replacement string"},
		{Short: "-x", Long: "--exec", Help: "Execute"},
	}

	output := FormatOptions(options)

	if output == "" {
		t.Error("FormatOptions returned empty string")
	}

	expectedStrings := []string{"-f", "--find", "PATTERN", "Pattern to find", "-r", "--replace", "STRING", "Replacement string", "-x", "--exec", "Execute"}
	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("expected output to contain %q", expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
