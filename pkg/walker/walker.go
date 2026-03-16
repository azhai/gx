// Package walker provides concurrent file system traversal functionality.
// It walks through directories and returns file information through a channel,
// with support for filtering by glob patterns and skipping binary files.
package walker

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// DefaultSkipDirs contains the default directories to skip during traversal.
// These are typically version control, dependency, and IDE directories.
var DefaultSkipDirs = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	"node_modules": true,
	"vendor":       true,
	"target":       true,
	"build":        true,
	".idea":        true,
	".vscode":      true,
}

// FileInfo represents information about a file or directory.
type FileInfo struct {
	// Path is the full path to the file or directory
	Path string
	// Name is the base name of the file or directory
	Name string
	// IsDir indicates whether this is a directory
	IsDir bool
	// Size is the file size in bytes (0 for directories)
	Size int64
}

// Config holds the configuration for a Walker.
type Config struct {
	// Paths are the root directories or files to start walking from
	Paths []string
	// FilePattern is a glob pattern to filter files (e.g., "*.go")
	FilePattern string
	// SkipDirs is a set of directory names to skip during traversal
	SkipDirs map[string]bool
	// Workers is the number of concurrent workers (not currently used)
	Workers int
	// IncludeDir indicates whether to include directories in the output
	IncludeDir bool
	// SkipBinary indicates whether to skip binary files
	SkipBinary bool
}

// Walker traverses file systems and returns file information.
// It uses filepath.Walk for directory traversal and supports concurrent processing.
type Walker struct {
	// config holds the walker configuration
	config *Config
	// files is the channel through which file information is sent
	files chan FileInfo
	// wg is used for synchronization (reserved for future use)
	wg sync.WaitGroup
}

// NewConfig creates a new Config with default values.
// Default values:
//   - SkipDirs: DefaultSkipDirs
//   - Workers: number of CPU cores
func NewConfig() *Config {
	return &Config{
		SkipDirs: DefaultSkipDirs,
		Workers:  runtime.NumCPU(),
	}
}

// New creates a new Walker with the given configuration.
// It sets default values for any missing configuration fields.
//
// Default values:
//   - Workers: number of CPU cores (if <= 0)
//   - SkipDirs: DefaultSkipDirs (if nil)
//   - Paths: ["."] (if empty)
func New(config *Config) *Walker {
	if config.Workers <= 0 {
		config.Workers = runtime.NumCPU()
	}
	if config.SkipDirs == nil {
		config.SkipDirs = DefaultSkipDirs
	}
	if len(config.Paths) == 0 {
		config.Paths = []string{"."}
	}

	return &Walker{
		config: config,
		files:  make(chan FileInfo, 1000),
	}
}

// Walk starts the file system traversal and returns a channel of FileInfo.
// The traversal is performed concurrently in a goroutine, and the channel
// is closed when all files have been processed.
//
// Example:
//
//	walker := walker.New(config)
//	files := walker.Walk()
//	for file := range files {
//	    fmt.Println(file.Path)
//	}
func (w *Walker) Walk() <-chan FileInfo {
	go func() {
		for _, path := range w.config.Paths {
			w.walkDir(path)
		}
		close(w.files)
	}()

	return w.files
}

// walkDir traverses a single directory recursively.
// It applies the configured filters and sends matching files to the channel.
func (w *Walker) walkDir(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Handle directories
		if info.IsDir() {
			name := info.Name()
			// Skip configured directories
			if w.config.SkipDirs[name] {
				return filepath.SkipDir
			}
			// Include directories if configured
			if w.config.IncludeDir {
				w.files <- FileInfo{
					Path:  path,
					Name:  name,
					IsDir: true,
					Size:  0,
				}
			}
			return nil
		}

		// Skip non-regular files (symlinks, devices, etc.)
		if !info.Mode().IsRegular() {
			return nil
		}

		// Apply glob pattern filter
		if w.config.FilePattern != "" {
			matched, err := filepath.Match(w.config.FilePattern, info.Name())
			if err != nil || !matched {
				return nil
			}
		}

		// Skip binary files if configured
		if w.config.SkipBinary && w.isBinaryFile(path) {
			return nil
		}

		// Send file info to channel
		w.files <- FileInfo{
			Path:  path,
			Name:  info.Name(),
			IsDir: false,
			Size:  info.Size(),
		}

		return nil
	})
}

// isBinaryFile checks whether a file is binary by reading the first 512 bytes.
// A file is considered binary if it contains a null byte (0x00) in the first 512 bytes.
// This is a heuristic that works well for most text and binary files.
func (w *Walker) isBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return true
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return true
	}

	// Check for null byte (common in binary files)
	return bytes.IndexByte(buf[:n], 0) != -1
}
