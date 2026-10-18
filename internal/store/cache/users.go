package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/redis/go-redis/v9"
)

const userCacheTTL = 5 * time.Minute

type RedisUserCache struct {
	rdb *redis.Client
}

func userKey(id int64) string {
	return fmt.Sprintf("user:%d", id)
}

func (c *RedisUserCache) Get(ctx context.Context, id int64) (*store.User, error) {
	status, err := c.rdb.Get(ctx, userKey(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var user store.User
	if err := json.Unmarshal([]byte(status), &user); err != nil {
		return nil, err
	}
	
	return &user, nil
}

func (c *RedisUserCache) Set(ctx context.Context, user *store.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return c.rdb.Set(ctx, userKey(user.ID), data, userCacheTTL).Err()
}

func (c *RedisUserCache) Delete(ctx context.Context, id int64) error {
	return c.rdb.Del(ctx, userKey(id)).Err()
}