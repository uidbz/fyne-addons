// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"fmt"
	"math"
	"time"
)

// formatSize mimics the strings KIO shows in the details view.
func formatSize(n int64, isDir bool, count int) string {
	if isDir {
		if count == 1 {
			return "1 item"
		}
		return fmt.Sprintf("%d items", count)
	}
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	v := float64(n)
	i := -1
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if v < 10 {
		return fmt.Sprintf("%.1f %s", v, units[i])
	}
	return fmt.Sprintf("%.0f %s", v, units[i])
}

// formatDate matches the en_US short date representation used by Plasma on the
// reference system, for example "10/22/26 at 7:24 PM".
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("1/2/06") + " at " + t.Local().Format("3:04 PM")
}

func clamp(v, lo, hi float32) float32 {
	return float32(math.Min(float64(hi), math.Max(float64(lo), float64(v))))
}
