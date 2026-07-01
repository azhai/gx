package integration

import (
	"testing"

	"github.com/azhai/gx/test/helper"
)

func TestFind_BasicSearch(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"a.txt": "TODO: fix this\n",
		"b.txt": "nothing here\n",
	})

	stdout, _, code := helper.RunGx(t, "find", "TODO", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "TODO", true)
}

func TestFind_AllFlag(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		".gitignore": "*.log\n",
		"error.log":  "error content\n",
		"main.go":    "package main\n",
	})

	stdout, _, _ := helper.RunGx(t, "find", "error", dir)
	helper.AssertOutput(t, stdout, "error.log", false)

	stdout, _, code := helper.RunGx(t, "find", "--all", "error", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "error.log", true)
}

func TestFind_FilesWithMatches(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"a.txt": "TODO: fix\n",
		"b.txt": "nothing\n",
	})

	stdout, _, code := helper.RunGx(t, "find", "-l", "TODO", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "a.txt", true)
	helper.AssertOutput(t, stdout, ":", false)
}

func TestFind_NoMatch(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"a.txt": "hello\n",
	})

	_, _, code := helper.RunGx(t, "find", "NONEXISTENT_PATTERN", dir)
	helper.AssertExitCode(t, code, 1)
}
