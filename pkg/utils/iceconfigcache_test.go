package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a4to/protocol/conkit"
)

func TestIceConfigCache(t *testing.T) {
	cache := NewIceConfigCache[string](10 * time.Second)
	t.Cleanup(cache.Stop)

	cache.Put("test", &conkit.ICEConfig{})
	require.NotNil(t, cache)
}
