package args

import (
	"strings"
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
		{
			name:            "empty args should call printUsage",
			args:            []string{},
			expectedPattern: "",
			expectedPaths:   nil,
		},
		{
			name:            "help flag -h",
			args:            []string{"-h"},
			expectedPattern: "",
			expectedPaths:   nil,
		},
		{
			name:            "help flag --help",
			args:            []string{"--help"},
			expectedPattern: "",
			expectedPaths:   nil,
		},
		{
			name:               "option with positional args mixed",
			args:               []string{"-x", "TODO", "FIXME", "/src"},
			expectedPattern:    "TODO",
			expectedReplace:    "FIXME",
			expectedReplaceSet: true,
			expectedPaths:      []string{"/src"},
			expectedDryRun:     false,
		},
		{
			name:               "long option names",
			args:               []string{"--find", "TODO", "--replace", "FIXME", "--exec"},
			expectedPattern:    "TODO",
			expectedReplace:    "FIXME",
			expectedReplaceSet: true,
			expectedPaths:      []string{"."},
			expectedDryRun:     false,
		},
	}

	defaultOptions := func() []Option {
		return []Option{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CommonConfig{
				DryRun: true,
			}

			options := tt.options
			if len(options) == 0 {
				options = defaultOptions()
			}

			result := ParseSimple(tt.args, config, options, func() {})

			if len(tt.args) == 0 || (len(tt.args) == 1 && (tt.args[0] == "-h" || tt.args[0] == "--help")) {
				if result {
					t.Errorf("expected false for empty/help args")
				}
				return
			}

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
		{
			name:            "help flag",
			args:            []string{"-h"},
			expectedPattern: "",
			expectedPaths:   nil,
		},
		{
			name:               "unknown option treated as positional",
			args:               []string{"--unknown", "TODO"},
			expectedPattern:    "--unknown",
			expectedReplace:    "TODO",
			expectedReplaceSet: true,
			expectedPaths:      []string{"."},
			expectedDryRun:     true,
		},
	}

	defaultOptions := func() []Option {
		return []Option{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CommonConfig{
				DryRun: true,
			}

			options := tt.options
			if len(options) == 0 {
				options = defaultOptions()
			}

			result := ParseCommon(tt.args, config, options, func() {})

			if tt.name == "help flag" {
				if result {
					t.Errorf("expected false for help flag")
				}
				return
			}

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
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q", expected)
		}
	}
}

func TestFormatOptions_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		contains []string
	}{
		{
			name:     "empty options",
			options:  []Option{},
			contains: []string{},
		},
		{
			name: "only short option",
			options: []Option{
				{Short: "-x", Help: "Execute"},
			},
			contains: []string{"-x", "Execute"},
		},
		{
			name: "only long option",
			options: []Option{
				{Long: "--verbose", Help: "Verbose output"},
			},
			contains: []string{"--verbose", "Verbose output"},
		},
		{
			name: "option with value but no short",
			options: []Option{
				{Long: "--output", HasValue: true, ValueName: "FILE", Help: "Output file"},
			},
			contains: []string{"--output", "FILE", "Output file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatOptions(tt.options)
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got %q", expected, output)
				}
			}
		})
	}
}

func TestOptionWithValueMissing(t *testing.T) {
	config := &CommonConfig{DryRun: true}
	options := []Option{
		{
			Short: "-f", Long: "--find", HasValue: true, ValueName: "PATTERN",
			Help: "Pattern to find",
			Handler: func(v string, cfg *CommonConfig) bool {
				cfg.Pattern = v
				return true
			},
		},
	}

	result := ParseCommon([]string{"-f"}, config, options, func() {})
	if result {
		t.Error("expected false when option value is missing")
	}
}

func TestCommonConfigDefaults(t *testing.T) {
	config := &CommonConfig{}

	if config.Pattern != "" {
		t.Error("default Pattern should be empty")
	}
	if config.Replace != "" {
		t.Error("default Replace should be empty")
	}
	if config.ReplaceSet {
		t.Error("default ReplaceSet should be false")
	}
	if config.Paths != nil {
		t.Error("default Paths should be nil")
	}
	if config.IgnoreCase {
		t.Error("default IgnoreCase should be false")
	}
	if config.FilePattern != "" {
		t.Error("default FilePattern should be empty")
	}
	if config.DryRun {
		t.Error("default DryRun should be false")
	}
}
