package redis

import (
	"context"
	"time"

	"github.com/go-redis/cache/v8"
	log "github.com/sirupsen/logrus"
)

// CachedEval will return the value at cacheKey if it exists.
// If it does not exist or is nil, CachedEval calls `fn()` to evaluate the result, and stores it at the cache key.
func CachedEval[T any](ctx context.Context, redis *Client, cacheKey string, lockTimeout, cacheExpiration time.Duration, fn func() (*T, error)) (value *T, err error) {
	if err = redis.Cache.Get(ctx, cacheKey, &value); err != nil {
		// if we do not have a cache hit, take a lock to avoid running fn() more than once
		if acquired := redis.AcquireLock(ctx, cacheKey+"-lock", lockTimeout); acquired {
			defer func() {
				if err := redis.ReleaseLock(ctx, cacheKey+"-lock"); err != nil {
					log.WithContext(ctx).WithError(err).Error("failed to release lock")
				}
			}()
		}
		if value, err = fn(); value == nil || err != nil {
			return
		}
		if err = redis.Cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   cacheKey,
			Value: &value,
			TTL:   cacheExpiration,
		}); err != nil {
			return
		}
	}

	return
}