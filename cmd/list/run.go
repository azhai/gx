package list

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"time"

	"github.com/azhai/gx/args"
	"github.com/azhai/gx/common"
	"github.com/azhai/gx/walker"
)

type Config struct {
	Glob       string
	SizeExpr   string
	MtimeExpr  string
	CtimeExpr  string
	SortField  string
	Reverse    bool
	TypeFilter string
	Format     string
	MaxDepth   int
	Paths      []string
	ShowFields bool
}

func NewConfig() *Config {
	return &Config{
		TypeFilter: "a",
		Format:     "path",
		MaxDepth:   0,
	}
}

func (c *Config) getOptions() []args.Option {
	return []args.Option{
		{Short: "-g", Long: "--glob", HasValue: true, ValueName: "PATTERN", Help: "File name glob pattern",
			Handler: func(v string, _ *args.CommonConfig) bool { c.Glob = v; return true }},
		{Short: "-t", Long: "--type", HasValue: true, ValueName: "f|d|l|a", Help: "File type filter (default a)",
			Handler: func(v string, _ *args.CommonConfig) bool {
				if v != "f" && v != "d" && v != "l" && v != "a" {
					fmt.Fprintf(os.Stderr, "Error: invalid type %q (must be f, d, l, or a)\n", v)
					return false
				}
				c.TypeFilter = v
				return true
			}},
		{Short: "-S", Long: "--size", HasValue: true, ValueName: "EXPR", Help: "Size filter expression (e.g. >1M, <=100K)",
			Handler: func(v string, _ *args.CommonConfig) bool { c.SizeExpr = v; return true }},
		{Short: "-M", Long: "--mtime", HasValue: true, ValueName: "EXPR", Help: "Modification time filter (e.g. <=1h, >=2025-01-01)",
			Handler: func(v string, _ *args.CommonConfig) bool { c.MtimeExpr = v; return true }},
		{Long: "--ctime", HasValue: true, ValueName: "EXPR", Help: "Creation time filter (e.g. <=1d, >=2025-01-01)",
			Handler: func(v string, _ *args.CommonConfig) bool { c.CtimeExpr = v; return true }},
		{Short: "-s", Long: "--sort", HasValue: true, ValueName: "FIELD", Help: "Sort by field (name/size/mtime/ctime/path)",
			Handler: func(v string, _ *args.CommonConfig) bool {
				validFields := map[string]bool{"name": true, "size": true, "mtime": true, "ctime": true, "path": true}
				if !validFields[v] {
					fmt.Fprintf(os.Stderr, "Error: invalid sort field %q (must be name, size, mtime, ctime, or path)\n", v)
					return false
				}
				c.SortField = v
				return true
			}},
		{Short: "-r", Long: "--reverse", Help: "Reverse sort order",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.Reverse = true; return true }},
		{Short: "-L", Long: "--max-depth", HasValue: true, ValueName: "N", Help: "Max recursion depth (0 = unlimited)",
			Handler: func(v string, _ *args.CommonConfig) bool { fmt.Sscanf(v, "%d", &c.MaxDepth); return true }},
		{Long: "--format", HasValue: true, ValueName: "path|long|name|show", Help: "Output format (default path)",
			Handler: func(v string, _ *args.CommonConfig) bool {
				if v != "path" && v != "long" && v != "name" && v != "show" {
					fmt.Fprintf(os.Stderr, "Error: invalid format %q (must be path, long, name, or show)\n", v)
					return false
				}
				c.Format = v
				return true
			}},
		{Long: "--show", Help: "Show size, mtime, and ctime in fixed-width columns",
			Handler: func(_ string, _ *args.CommonConfig) bool { c.ShowFields = true; return true }},
	}
}

func (c *Config) ParseArgs() bool {
	optionMap := make(map[string]bool)
	for _, opt := range c.getOptions() {
		if opt.Short != "" {
			optionMap[opt.Short] = true
		}
		if opt.Long != "" {
			optionMap[opt.Long] = true
		}
	}

	argv := os.Args[1:]
	if len(argv) == 0 {
		c.printUsage()
		return false
	}

	for _, arg := range argv {
		if arg == "-h" || arg == "--help" {
			c.printUsage()
			return false
		}
	}

	remaining, err := parseListArgs(argv, c, optionMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return false
	}

	for _, arg := range remaining {
		c.Paths = append(c.Paths, arg)
	}

	if len(c.Paths) == 0 {
		c.Paths = []string{"."}
	}

	return true
}

func parseListArgs(argv []string, config *Config, optionMap map[string]bool) ([]string, error) {
	var positional []string
	opts := config.getOptions()
	optionHandlers := make(map[string]*args.Option)
	for i := range opts {
		if opts[i].Short != "" {
			optionHandlers[opts[i].Short] = &opts[i]
		}
		if opts[i].Long != "" {
			optionHandlers[opts[i].Long] = &opts[i]
		}
	}

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if opt, exists := optionHandlers[arg]; exists {
			if opt.HasValue {
				if i+1 >= len(argv) {
					return nil, fmt.Errorf("%s requires a value", arg)
				}
				dummy := args.CommonConfig{}
				if !opt.Handler(argv[i+1], &dummy) {
					return nil, fmt.Errorf("invalid value for %s", arg)
				}
				i++
			} else {
				dummy := args.CommonConfig{}
				if !opt.Handler("", &dummy) {
					return nil, fmt.Errorf("invalid option %s", arg)
				}
			}
		} else if strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("unknown option %s", arg)
		} else {
			positional = append(positional, arg)
		}
	}
	return positional, nil
}

func (c *Config) printUsage() {
	fmt.Println(`list - List files and directories with filters (like ls)

Usage: list [OPTIONS] [PATH...]

Options:
` + args.FormatOptions(c.getOptions()) + `
Examples:
  list ./src
  list -g "*.go" --size ">1K" --mtime "<=1d" ./src
  list --format long --sort size --reverse ./data
  list --type d --max-depth 1 .`)
}

type Entry struct {
	Path          string
	Name          string
	Size          int64
	ModTime       time.Time
	ChangeTime    time.Time
	Mode          fs.FileMode
	IsDir         bool
	IsSymlink     bool
	SymlinkTarget string
}

type Lister struct {
	config      *Config
	sizeFilter  *common.SizeFilter
	mtimeFilter *common.TimeFilter
	ctimeFilter *common.TimeFilter
}

func NewLister(config *Config) (*Lister, error) {
	l := &Lister{config: config}

	if config.SizeExpr != "" {
		sf, err := common.ParseSizeExpr(config.SizeExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid size expression: %w", err)
		}
		l.sizeFilter = sf
	}

	now := time.Now()
	if config.MtimeExpr != "" {
		tf, err := common.ParseTimeExpr(config.MtimeExpr, now)
		if err != nil {
			return nil, fmt.Errorf("invalid mtime expression: %w", err)
		}
		l.mtimeFilter = tf
	}

	if config.CtimeExpr != "" {
		tf, err := common.ParseTimeExpr(config.CtimeExpr, now)
		if err != nil {
			return nil, fmt.Errorf("invalid ctime expression: %w", err)
		}
		l.ctimeFilter = tf
	}

	return l, nil
}

func (l *Lister) shouldShowSize() bool {
	return l.config.ShowFields || l.config.SizeExpr != ""
}

func (l *Lister) shouldShowMtime() bool {
	return l.config.ShowFields || l.config.MtimeExpr != ""
}

func (l *Lister) shouldShowCtime() bool {
	return l.config.ShowFields || l.config.CtimeExpr != ""
}

func (l *Lister) Run() int {

	wc := walker.NewConfig()
	wc.Paths = l.config.Paths
	wc.IncludeDir = l.config.TypeFilter == "d" || l.config.TypeFilter == "a"
	wc.IncludeSymlink = l.config.TypeFilter == "l" || l.config.TypeFilter == "a"
	wc.MaxDepth = l.config.MaxDepth
	if l.config.Glob != "" {
		wc.FilePattern = l.config.Glob
	}

	w := walker.New(wc)
	files := w.Walk()

	rootSet := make(map[string]bool)
	for _, p := range l.config.Paths {
		abs, _ := filepath.Abs(p)
		rootSet[abs] = true
	}

	var entries []Entry
	for fi := range files {
		abs, _ := filepath.Abs(fi.Path)
		if rootSet[abs] {
			continue
		}
		e := l.toEntry(fi)
		if l.matchEntry(e) {
			entries = append(entries, e)
		}
	}

	if l.config.SortField != "" {
		l.sortEntries(entries)
	}

	if len(entries) == 0 {
		return 1
	}

	widths := l.calculateColumnWidths(entries)

	for _, e := range entries {
		l.printEntry(e, widths)
	}

	return 0
}

func (l *Lister) toEntry(fi walker.FileInfo) Entry {
	e := Entry{
		Path:       fi.Path,
		Name:       fi.Name,
		Size:       fi.Size,
		ModTime:    fi.ModTime,
		Mode:       fi.Mode,
		IsDir:      fi.IsDir,
		IsSymlink:  fi.Mode&os.ModeSymlink != 0,
		ChangeTime: getChangeTime(fi.Path, fi.ModTime),
	}
	if e.IsSymlink {
		if target, err := os.Readlink(fi.Path); err == nil {
			e.SymlinkTarget = target
		}
	}
	return e
}

func getChangeTime(path string, fallback time.Time) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return fallback
	}
	return getChangeTimeFromSys(info, fallback)
}

func (l *Lister) matchEntry(e Entry) bool {
	if l.config.TypeFilter != "a" {
		switch l.config.TypeFilter {
		case "f":
			if e.IsDir || e.IsSymlink {
				return false
			}
		case "d":
			if !e.IsDir {
				return false
			}
		case "l":
			if !e.IsSymlink {
				return false
			}
		}
	}

	if l.config.Glob != "" && e.IsDir {
		matched, err := filepath.Match(l.config.Glob, e.Name)
		if err != nil || !matched {
			return false
		}
	}

	if l.sizeFilter != nil && !l.sizeFilter.Match(e.Size) {
		return false
	}

	if l.mtimeFilter != nil && !l.mtimeFilter.Match(e.ModTime) {
		return false
	}

	if l.ctimeFilter != nil && !l.ctimeFilter.Match(e.ChangeTime) {
		return false
	}

	return true
}

func (l *Lister) sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		var less bool
		switch l.config.SortField {
		case "name":
			less = entries[i].Name < entries[j].Name
		case "size":
			less = entries[i].Size < entries[j].Size
		case "mtime":
			less = entries[i].ModTime.Before(entries[j].ModTime)
		case "ctime":
			less = entries[i].ChangeTime.Before(entries[j].ChangeTime)
		case "path":
			less = entries[i].Path < entries[j].Path
		default:
			return false
		}
		if l.config.Reverse {
			return !less
		}
		return less
	})
}

func (l *Lister) calculateColumnWidths(entries []Entry) common.ColumnWidths {
	widths := common.DefaultColumnWidths()

	maxPathLen := 0
	for _, e := range entries {
		path := l.formatPathWithMarker(e)
		pathLen := len(path)
		if pathLen > maxPathLen {
			maxPathLen = pathLen
		}
	}

	if maxPathLen > widths.Path {
		if maxPathLen > 120 {
			widths.Path = 120
		} else {
			widths.Path = maxPathLen
		}
	}

	return widths
}

func (l *Lister) formatPathWithMarker(e Entry) string {
	path := e.Path
	if e.IsDir {
		path += "/"
	}
	if e.IsSymlink {
		if e.SymlinkTarget == "" {
			path += " -> [error]"
		} else {
			target := e.SymlinkTarget
			if targetInfo, err := os.Stat(filepath.Join(filepath.Dir(e.Path), target)); err == nil && targetInfo.IsDir() {
				target += "/"
			}
			path += " -> " + target
		}
	}
	return path
}

func (l *Lister) printEntry(e Entry, widths common.ColumnWidths) {
	if l.config.Format == "name" {
		fmt.Println(e.Name)
		return
	}

	path := l.formatPathWithMarker(e)

	if l.config.Format == "path" && !l.shouldShowSize() && !l.shouldShowMtime() && !l.shouldShowCtime() {
		fmt.Println(path)
		return
	}

	var parts []string

	if l.shouldShowSize() {
		parts = append(parts, common.FormatFixedWidth(common.FormatSize(e.Size), widths.Size, true))
	}
	if l.shouldShowMtime() {
		parts = append(parts, common.FormatFixedWidth(common.FormatTime(e.ModTime), widths.Time, false))
	}
	if l.shouldShowCtime() {
		parts = append(parts, common.FormatFixedWidth(common.FormatTime(e.ChangeTime), widths.Time, false))
	}
	parts = append(parts, common.FormatFixedWidth(path, widths.Path, false))

	fmt.Println(strings.Join(parts, " "))
}

func formatMode(mode fs.FileMode, isDir, isSymlink bool) string {
	s := mode.String()
	if len(s) > 10 {
		s = s[len(s)-10:]
	}
	if isDir && len(s) >= 1 {
		s = "d" + s[1:]
	}
	if isSymlink && len(s) >= 1 {
		s = "l" + s[1:]
	}
	for len(s) < 10 {
		s = s + "-"
	}
	return s[:10]
}
