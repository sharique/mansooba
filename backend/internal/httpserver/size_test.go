package httpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"4M": 4 << 20, "25M": 25 << 20, "1K": 1024, "1KB": 1024, "2mb": 2 << 20,
		"1G": 1 << 30, "1.5K": 1536, "512": 512, "512B": 512, " 4M ": 4 << 20,
	}
	for in, want := range cases {
		got, err := parseSize(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"", "M", "abc", "-1M", "4X"} {
		_, err := parseSize(bad)
		assert.Error(t, err, bad)
	}
}
