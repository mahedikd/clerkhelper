package clerkhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/dgraph-io/ristretto/v2"
	"golang.org/x/sync/singleflight"
)

var userGet = user.Get

type UserCache struct {
	cache *ristretto.Cache[string, *ClerkUserData]
	ttl   time.Duration
	group singleflight.Group
}

var userCache = &UserCache{
	ttl: 5 * time.Minute,
}

func (uc *UserCache) ensureCache() {
	if config.CacheTTL > 0 {
		uc.ttl = config.CacheTTL
	}

	if uc.cache != nil {
		return
	}

	limit := config.CacheLimit
	if limit <= 0 {
		limit = 100_000
	}

	c, err := ristretto.NewCache(&ristretto.Config[string, *ClerkUserData]{
		NumCounters: 1e7,
		MaxCost:     limit,
		BufferItems: 64,
	})
	if err != nil {
		panic(fmt.Sprintf("clerkhelper: cache init failed: %v", err))
	}

	uc.cache = c
}

func (uc *UserCache) GetUserData(ctx context.Context, userID string) (*ClerkUserData, error) {
	uc.ensureCache()

	if val, found := uc.cache.Get(userID); found {
		return val, nil
	}

	value, err, _ := uc.group.Do(userID, func() (any, error) {
		if val, found := uc.cache.Get(userID); found {
			return val, nil
		}

		u, err := userGet(ctx, userID)
		if err != nil {
			return nil, err
		}

		var meta map[string]any
		if err := json.Unmarshal(u.PublicMetadata, &meta); err != nil {
			return nil, err
		}

		data := &ClerkUserData{
			UserID: u.ID,
			Extra:  make(map[string]string),
		}

		if r, ok := meta["role"].(string); ok {
			data.Role = r
		}

		for _, k := range config.MetadataKeys {
			if v, ok := meta[k].(string); ok {
				data.Extra[k] = v
			}
		}

		uc.cache.SetWithTTL(userID, data, 1, uc.ttl)
		return data, nil
	})

	if err != nil {
		return nil, err
	}

	return value.(*ClerkUserData), nil
}

func StartCacheCleanup() func() {
	return func() {
		if userCache.cache != nil {
			userCache.cache.Close()
		}
	}
}
