//go:build !darwin && !linux && !windows

package list

import (
	"os"
	"time"
)

func getChangeTimeFromSys(info os.FileInfo, fallback time.Time) time.Time {
	return fallback
}
