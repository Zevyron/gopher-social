package cache

import (
	"context"

	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/redis/go-redis/v9"
)

type UserCache interface {
		Get(context.Context, int64) (*store.User, error)
		Set(context.Context, *store.User) error
		Delete(context.Context, int64) error
}

type Cache struct {
	Users UserCache
}

func NewRedisCache(rdb *redis.Client) Cache {
	return Cache{
		Users: &RedisUserCache{rdb: rdb},
	}
}