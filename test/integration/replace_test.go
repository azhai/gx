package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azhai/gx/test/helper"
)

func TestReplace_DryRun(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"a.txt": "old value here\n",
	})

	stdout, _, code := helper.RunGx(t, "replace", "old", "new", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "DRY-RUN", true)

	content, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	if strings.Contains(string(content), "new value") {
		t.Error("dry-run should not modify files")
	}
}

func TestReplace_AllFlag(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		".gitignore": "*.log\n",
		"error.log":  "old content\n",
		"main.go":    "old package\n",
	})

	stdout, _, _ := helper.RunGx(t, "replace", "old", "new", dir)
	helper.AssertOutput(t, stdout, "error.log", false)

	stdout, _, code := helper.RunGx(t, "replace", "--all", "old", "new", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "error.log", true)
}

func TestReplace_ExecReplace(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"a.txt": "old value\n",
	})

	stdout, _, code := helper.RunGx(t, "replace", "old", "new", "-x", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "REPLACED", true)
}
