package common

import (
	"os"
	"path/filepath"
	"strings"
)

type GitignorePattern struct {
	pattern string
	negated bool
	dirOnly bool
	rooted  bool
}

func NewGitignorePattern(line string) *GitignorePattern {
	line = strings.TrimRight(line, " \t")
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}

	p := &GitignorePattern{}

	if strings.HasPrefix(line, "!") {
		p.negated = true
		line = line[1:]
	}

	if strings.HasPrefix(line, `\#`) {
		line = "#" + line[2:]
	}

	if strings.HasSuffix(line, "/") {
		p.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	if strings.HasPrefix(line, "/") {
		p.rooted = true
		line = line[1:]
	}

	if line == "" {
		return nil
	}

	p.pattern = line
	return p
}

func (p *GitignorePattern) Match(name string, isDir bool) bool {
	if p.dirOnly && !isDir {
		return false
	}

	pattern := p.pattern
	if p.rooted {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return false
		}
		return matched
	}

	if strings.Contains(pattern, "/") {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return false
		}
		return matched
	}

	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}

func (p *GitignorePattern) IsNegated() bool { return p.negated }
func (p *GitignorePattern) IsDirOnly() bool { return p.dirOnly }

type GitignoreMatcher struct {
	patterns []GitignorePattern
	parent   *GitignoreMatcher
	dir      string
}

func NewGitignoreMatcher(dir string, parent *GitignoreMatcher, content string) *GitignoreMatcher {
	m := &GitignoreMatcher{
		dir:    dir,
		parent: parent,
	}

	for _, line := range strings.Split(content, "\n") {
		p := NewGitignorePattern(line)
		if p != nil {
			m.patterns = append(m.patterns, *p)
		}
	}

	return m
}

func (m *GitignoreMatcher) Match(name string, isDir bool) bool {
	ignored := false

	if m.parent != nil {
		ignored = m.parent.Match(name, isDir)
	}

	for i := range m.patterns {
		p := &m.patterns[i]
		if p.Match(name, isDir) {
			if p.negated {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

func (m *GitignoreMatcher) Child(dir string) *GitignoreMatcher {
	return &GitignoreMatcher{
		dir:    dir,
		parent: m,
	}
}

func LoadGitignoreFile(dir string, parent *GitignoreMatcher) *GitignoreMatcher {
	gitignorePath := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return parent
	}
	return NewGitignoreMatcher(dir, parent, string(data))
}
