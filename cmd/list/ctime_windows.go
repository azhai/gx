//go:build windows

package list

import (
	"os"
	"syscall"
	"time"
)

func getChangeTimeFromSys(info os.FileInfo, fallback time.Time) time.Time {
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.CreationTime.Nanoseconds())
	}
	return fallback
}
