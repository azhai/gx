package helper

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func CreateTestDir(t *testing.T, spec map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	for path, content := range spec {
		fullPath := filepath.Join(dir, path)
		if content == "/" {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				t.Fatalf("Failed to create dir %s: %v", fullPath, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create parent dir for %s: %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	return dir
}

func FindGxBinary() (string, error) {
	candidates := []string{
		"./gx",
		"./bin/gx",
		"../gx",
		"../bin/gx",
		"../../gx",
		"../../bin/gx",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs, nil
		}
	}
	return "", os.ErrNotExist
}

func RunGx(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	bin, err := FindGxBinary()
	if err != nil {
		t.Skip("gx binary not found, run 'make build' first")
	}

	cmd := exec.Command(bin, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("Failed to run gx: %v", err)
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

func AssertOutput(t *testing.T, output, expected string, shouldContain bool) {
	t.Helper()
	contains := strings.Contains(output, expected)
	if shouldContain && !contains {
		t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", expected, output)
	}
	if !shouldContain && contains {
		t.Errorf("Expected output NOT to contain %q, but it did.\nOutput:\n%s", expected, output)
	}
}

func AssertExitCode(t *testing.T, actual, expected int) {
	t.Helper()
	if actual != expected {
		t.Errorf("Expected exit code %d, got %d", expected, actual)
	}
}
