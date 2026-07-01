//go:build linux

package list

import (
	"os"
	"syscall"
	"time"
)

func getChangeTimeFromSys(info os.FileInfo, fallback time.Time) time.Time {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		sec := stat.Ctim.Sec
		nsec := stat.Ctim.Nsec
		t := time.Unix(sec, int64(nsec))
		if !t.IsZero() && t.Year() > 1970 {
			return t
		}
	}
	return fallback
}
