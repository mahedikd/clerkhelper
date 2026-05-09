package clerkhelper

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	clerkSDK "github.com/clerk/clerk-sdk-go/v2"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T, ttl time.Duration) *UserCache {
	cache, err := ristretto.NewCache(&ristretto.Config[string, *ClerkUserData]{
		NumCounters: 1e3,
		MaxCost:     100,
		BufferItems: 64,
	})
	require.NoError(t, err)
	return &UserCache{
		cache: cache,
		ttl:   ttl,
	}
}

func TestUserCache_GetUserData(t *testing.T) {
	origUserGet := userGet
	t.Cleanup(func() { userGet = origUserGet })

	t.Run("Cache hit returns cached data without API call", func(t *testing.T) {
		uc := newTestCache(t, 5*time.Minute)
		expected := &ClerkUserData{UserID: "user_cached", Role: "ADMIN"}

		uc.cache.Set("user_cached", expected, 1)
		time.Sleep(10 * time.Millisecond)

		apiCalled := false
		userGet = func(ctx context.Context, userID string) (*clerkSDK.User, error) {
			apiCalled = true
			return nil, errors.New("should not be called")
		}

		result, err := uc.GetUserData(context.Background(), "user_cached")
		require.NoError(t, err)
		assert.Equal(t, expected, result)
		assert.False(t, apiCalled)
	})

	t.Run("Cache miss fetches from API with extra metadata", func(t *testing.T) {
		uc := newTestCache(t, 5*time.Minute)
		Init(Config{MetadataKeys: []string{"t_id", "org_id"}})

		metadata := map[string]interface{}{
			"role":   "OWNER",
			"t_id":   "tenant_1",
			"org_id": "org_1",
			"other":  "ignored",
		}
		metaBytes, _ := json.Marshal(metadata)

		userGet = func(ctx context.Context, userID string) (*clerkSDK.User, error) {
			return &clerkSDK.User{
				ID:             "user_new",
				PublicMetadata: json.RawMessage(metaBytes),
			}, nil
		}

		result, err := uc.GetUserData(context.Background(), "user_new")
		require.NoError(t, err)
		assert.Equal(t, "user_new", result.UserID)
		assert.Equal(t, "OWNER", result.Role)
		assert.Equal(t, "tenant_1", result.Extra["t_id"])
		assert.Equal(t, "org_1", result.Extra["org_id"])
		_, ok := result.Extra["other"]
		assert.False(t, ok)
	})
}

func TestUserCache_ConcurrentAccess(t *testing.T) {
	origUserGet := userGet
	t.Cleanup(func() { userGet = origUserGet })

	uc := newTestCache(t, 5*time.Minute)

	metadata := map[string]interface{}{"role": "ADMIN"}
	metaBytes, _ := json.Marshal(metadata)

	var callCount int
	var mu sync.Mutex

	userGet = func(ctx context.Context, userID string) (*clerkSDK.User, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		return &clerkSDK.User{
			ID:             userID,
			PublicMetadata: json.RawMessage(metaBytes),
		}, nil
	}

	var wg sync.WaitGroup
	numReqs := 10
	for i := 0; i < numReqs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = uc.GetUserData(context.Background(), "user_concurrent")
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, callCount)
}

func TestCacheConfiguration(t *testing.T) {
	// Reset global state for this test
	userCache = &UserCache{
		ttl: 5 * time.Minute,
	}

	Init(Config{
		CacheTTL:   10 * time.Minute,
		CacheLimit: 2000,
	})

	userCache.ensureCache()

	assert.Equal(t, 10*time.Minute, userCache.ttl)
}
