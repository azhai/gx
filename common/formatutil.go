package common

import (
	"fmt"
	"io/fs"
	"strings"
	"time"
)

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

func FormatFixedWidth(value string, width int, alignRight bool) string {
	if len(value) > width {
		if width > 3 {
			return value[:width-3] + "..."
		}
		return value[:width]
	}

	if alignRight {
		return fmt.Sprintf("%*s", width, value)
	}
	return fmt.Sprintf("%-*s", width, value)
}

func FormatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1fK", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1fM", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1fG", float64(size)/(1024*1024*1024))
}

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return strings.Repeat(" ", 19)
	}
	return t.Format("2006-01-02T15:04:05")
}

func FormatMode(mode fs.FileMode, isDir, isSymlink bool) string {
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

type ColumnWidths struct {
	Mode int
	Size int
	Time int
	Path int
}

func DefaultColumnWidths() ColumnWidths {
	return ColumnWidths{
		Mode: 10,
		Size: 10,
		Time: 19,
		Path: 40,
	}
}

func CalculateColumnWidths(entries []Entry, format string) ColumnWidths {
	widths := DefaultColumnWidths()

	maxPathLen := 0
	for _, e := range entries {
		pathLen := len(e.Path)
		if e.IsDir {
			pathLen++
		}
		if e.IsSymlink && e.SymlinkTarget != "" {
			pathLen += 4 + len(e.SymlinkTarget)
		}
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

func FormatEntryLong(e Entry, widths ColumnWidths) string {
	sizeStr := FormatFixedWidth(FormatSize(e.Size), widths.Size, true)
	timeStr := FormatFixedWidth(FormatTime(e.ModTime), widths.Time, false)
	pathStr := FormatFixedWidth(e.Path, widths.Path, false)
	return fmt.Sprintf("%s %s %s", sizeStr, timeStr, pathStr)
}

func FormatEntryShow(e Entry, widths ColumnWidths) string {
	sizeStr := FormatFixedWidth(FormatSize(e.Size), widths.Size, true)
	mtimeStr := FormatFixedWidth(FormatTime(e.ModTime), widths.Time, false)
	ctimeStr := FormatFixedWidth(FormatTime(e.ChangeTime), widths.Time, false)
	pathStr := FormatFixedWidth(e.Path, widths.Path, false)
	return fmt.Sprintf("%s %s %s %s", sizeStr, mtimeStr, ctimeStr, pathStr)
}
