package walker

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

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

type FileInfo struct {
	Path  string
	Name  string
	IsDir bool
	Size  int64
}

type Config struct {
	Paths       []string
	FilePattern string
	SkipDirs    map[string]bool
	Workers     int
	IncludeDir  bool
	SkipBinary  bool
}

type Walker struct {
	config *Config
	files  chan FileInfo
	wg     sync.WaitGroup
}

func NewConfig() *Config {
	return &Config{
		SkipDirs: DefaultSkipDirs,
		Workers:  runtime.NumCPU(),
	}
}

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

func (w *Walker) Walk() <-chan FileInfo {
	go func() {
		for _, path := range w.config.Paths {
			w.walkDir(path)
		}
		close(w.files)
	}()

	return w.files
}

func (w *Walker) walkDir(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			if w.config.SkipDirs[name] {
				return filepath.SkipDir
			}
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

		if !info.Mode().IsRegular() {
			return nil
		}

		if w.config.FilePattern != "" {
			matched, err := filepath.Match(w.config.FilePattern, info.Name())
			if err != nil || !matched {
				return nil
			}
		}

		if w.config.SkipBinary && w.isBinaryFile(path) {
			return nil
		}

		w.files <- FileInfo{
			Path:  path,
			Name:  info.Name(),
			IsDir: false,
			Size:  info.Size(),
		}

		return nil
	})
}

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

	return bytes.IndexByte(buf[:n], 0) != -1
}
