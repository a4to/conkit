package utils

import (
	"time"

	"github.com/jellydator/ttlcache/v3"

	"github.com/a4to/protocol/conkit"
)

const (
	iceConfigTTLMin = 5 * time.Minute
)

type IceConfigCache[T comparable] struct {
	c *ttlcache.Cache[T, *conkit.ICEConfig]
}

func NewIceConfigCache[T comparable](ttl time.Duration) *IceConfigCache[T] {
	cache := ttlcache.New(
		ttlcache.WithTTL[T, *conkit.ICEConfig](max(ttl, iceConfigTTLMin)),
		ttlcache.WithDisableTouchOnHit[T, *conkit.ICEConfig](),
	)
	go cache.Start()

	return &IceConfigCache[T]{cache}
}

func (icc *IceConfigCache[T]) Stop() {
	icc.c.Stop()
}

func (icc *IceConfigCache[T]) Put(key T, iceConfig *conkit.ICEConfig) {
	icc.c.Set(key, iceConfig, ttlcache.DefaultTTL)
}

func (icc *IceConfigCache[T]) Get(key T) *conkit.ICEConfig {
	if it := icc.c.Get(key); it != nil {
		return it.Value()
	}
	return &conkit.ICEConfig{}
}
