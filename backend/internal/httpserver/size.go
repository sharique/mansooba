package httpserver

import (
	"fmt"
	"strconv"
	"strings"
)

// parseSize converts a size such as "4M", "25M", "1K" or "1.5GB" to bytes,
// using powers of 1024 and case-insensitive suffixes (K, KB, M, MB, G, GB, T,
// TB). A bare number is bytes. It mirrors the units Echo v4's BodyLimit
// accepted, so BODY_SIZE_LIMIT keeps working unchanged under Echo v5, whose
// BodyLimit takes a byte count.
func parseSize(s string) (int64, error) {
	t := strings.ToUpper(strings.TrimSpace(s))
	if t == "" {
		return 0, fmt.Errorf("empty size")
	}
	mult := int64(1)
	for _, u := range []struct {
		suffix string
		mult   int64
	}{
		{"TB", 1 << 40}, {"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10},
		{"T", 1 << 40}, {"G", 1 << 30}, {"M", 1 << 20}, {"K", 1 << 10}, {"B", 1},
	} {
		if strings.HasSuffix(t, u.suffix) {
			mult = u.mult
			t = strings.TrimSpace(strings.TrimSuffix(t, u.suffix))
			break
		}
	}
	n, err := strconv.ParseFloat(t, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return int64(n * float64(mult)), nil
}
