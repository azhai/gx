package integration

import (
	"testing"

	"github.com/azhai/gx/test/helper"
)

func TestList_BasicList(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"a.txt": "content a\n",
		"b.go":  "content b\n",
	})

	stdout, _, code := helper.RunGx(t, "list", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "a.txt", true)
	helper.AssertOutput(t, stdout, "b.go", true)
}

func TestList_GlobFilter(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"a.txt": "content\n",
		"b.go":  "content\n",
	})

	stdout, _, code := helper.RunGx(t, "list", "-g", "*.go", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "b.go", true)
	helper.AssertOutput(t, stdout, "a.txt", false)
}

func TestList_FormatLong(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"test.txt": "hello world\n",
	})

	stdout, _, code := helper.RunGx(t, "list", "--format", "long", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "test.txt", true)
}

func TestList_SortBySize(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"small.txt": "a\n",
		"big.txt":   "aaaaaa\n",
	})

	stdout, _, code := helper.RunGx(t, "list", "--sort", "size", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "small.txt", true)
	helper.AssertOutput(t, stdout, "big.txt", true)
}

func TestList_SizeFilter(t *testing.T) {
	dir := helper.CreateTestDir(t, map[string]string{
		"small.txt": "a\n",
		"big.txt":   string(make([]byte, 2048)) + "\n",
	})

	stdout, _, code := helper.RunGx(t, "list", "--size", ">1K", dir)
	helper.AssertExitCode(t, code, 0)
	helper.AssertOutput(t, stdout, "big.txt", true)
	helper.AssertOutput(t, stdout, "small.txt", false)
}
