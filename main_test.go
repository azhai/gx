package main

import (
	"bytes"
	"strings"
	"testing"
)

// withVersion overrides the build-time version/commit vars for the
// duration of t and restores them on cleanup.
func withVersion(t *testing.T, v, c string) {
	t.Helper()
	origV, origC := version, commit
	version, commit = v, c
	t.Cleanup(func() { version, commit = origV, origC })
}

func TestVersion(t *testing.T) {
	withVersion(t, "1.2.3", "deadbeef")
	got := Version()
	if !strings.Contains(got, "1.2.3") || !strings.Contains(got, "deadbeef") {
		t.Errorf("Version() = %q, want both version and commit", got)
	}
}

func TestRunMain_NoArgs(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := runMain(nil, out, errOut)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("expected usage on stdout, got %q", out.String())
	}
}

func TestRunMain_Help(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := runMain([]string{"gx", "--help"}, out, errOut)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Commands:") {
		t.Errorf("expected Commands list, got %q", out.String())
	}
}

func TestRunMain_Version(t *testing.T) {
	withVersion(t, "9.9.9", "cafef00d")
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := runMain([]string{"gx", "--version"}, out, errOut)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "9.9.9") {
		t.Errorf("expected version in output, got %q", out.String())
	}
}

func TestRunMain_UnknownCommand(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := runMain([]string{"gx", "bogus"}, out, errOut)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "Unknown command") {
		t.Errorf("expected unknown-command message on stderr, got %q", errOut.String())
	}
	if !strings.Contains(errOut.String(), "Usage:") {
		t.Errorf("expected usage on stderr, got %q", errOut.String())
	}
}

// PrintUsage contains all known subcommands. Catches typos in usage().
func TestPrintUsage_ListsAllCommands(t *testing.T) {
	var buf bytes.Buffer
	printUsageTo(&buf)
	got := buf.String()
	for _, cmd := range []string{"find", "list", "replace", "rename", "cut", "trans", "script"} {
		if !strings.Contains(got, cmd) {
			t.Errorf("usage missing command %q", cmd)
		}
	}
}
