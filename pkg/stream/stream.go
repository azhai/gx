// Package stream provides unified input handling: read from a file path
// or from stdin when the path is empty or "-".
//
// Used by commands that support stdin/stdout pipeline (cut/trans/script
// in later batches; find/replace/list use it for stdin input).
package stream

import (
	"fmt"
	"io"
	"os"
)

// StdinName is the virtual filename used when reading from stdin.
// find/list report this as the match's file path.
const StdinName = "<stdin>"

// IsStdin reports whether paths indicates stdin input.
// True when paths is empty or contains exactly "-".
func IsStdin(paths []string) bool {
	if len(paths) == 0 {
		return true
	}
	return len(paths) == 1 && paths[0] == "-"
}

// OpenInput opens the given path for reading. If path is "" or "-",
// returns os.Stdin (no closer needed). Otherwise opens the file;
// caller must call the returned closer.
//
// Returns the reader, a closer (nil for stdin), and any error.
func OpenInput(path string) (io.Reader, io.Closer, error) {
	if path == "" || path == "-" {
		return os.Stdin, nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	return f, f, nil
}

// ReadAll reads all content from the input source identified by paths.
// If paths indicates stdin, reads stdin; otherwise reads the file.
// Returns the content and the virtual filename (for match reporting).
func ReadAll(paths []string) ([]byte, string, error) {
	var path string
	if IsStdin(paths) {
		path = "-"
	} else {
		path = paths[0]
	}
	r, closer, err := OpenInput(path)
	if err != nil {
		return nil, "", err
	}
	if closer != nil {
		defer closer.Close()
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}
	name := path
	if path == "-" {
		name = StdinName
	}
	return data, name, nil
}
