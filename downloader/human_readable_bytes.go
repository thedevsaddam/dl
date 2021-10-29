package downloader

import (
	"fmt"
	"math"
)

// humanaReadableBytes convert Bytes to human readable form
func humanaReadableBytes(s float64) string {
	units := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	base := 1024.0
	if s < 10 {
		return fmt.Sprintf("%2.0f B", s)
	}
	e := math.Floor(math.Log(s) / math.Log(base))
	suffix := units[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f"
	if val < 10 {
		f = "%.1f"
	}

	return fmt.Sprintf("%s %s", fmt.Sprintf(f, val), suffix)
}
