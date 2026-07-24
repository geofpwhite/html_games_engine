// Package rediscache provides a Redis-backed implementation of the Cache interface.
package rediscache

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/geofpwhite/html_games_engine/accounts/cache"

	"github.com/redis/go-redis/v9"
)

const sessionTTL = 20 * time.Minute

const (
	sessionKeyPrefix  = "session:"
	userSessionPrefix = "usersession:"
	onlineUsersKey    = "online_users"
	onlineActivityKey = "online_activity"
)

type cacher struct {
	client redis.Client
}

func NewCache() cache.Cache {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "redis:6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &cacher{
		client: *client,
	}
}

func (c *cacher) SetSession(sessionKey string, userID int32, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userIDStr := strconv.Itoa(int(userID))

	if err := c.client.Set(ctx, sessionKeyPrefix+sessionKey, userIDStr, sessionTTL).Err(); err != nil {
		return err
	}
	if err := c.client.Set(ctx, userSessionPrefix+userIDStr, sessionKey, sessionTTL).Err(); err != nil {
		return err
	}
	if err := c.client.HSet(ctx, onlineUsersKey, userIDStr, username).Err(); err != nil {
		return err
	}
	return c.client.ZAdd(ctx, onlineActivityKey, redis.Z{Score: float64(time.Now().Unix()), Member: userIDStr}).Err()
}

func (c *cacher) GetSession(sessionKey string) (int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userIDStr, err := c.client.Get(ctx, sessionKeyPrefix+sessionKey).Result()
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseInt(userIDStr, 10, 32)
	if err != nil {
		return 0, err
	}

	if _, err := c.client.Expire(ctx, sessionKeyPrefix+sessionKey, sessionTTL).Result(); err != nil {
		return 0, err
	}
	if _, err := c.client.Expire(ctx, userSessionPrefix+userIDStr, sessionTTL).Result(); err != nil {
		return 0, err
	}
	if err := c.client.ZAdd(ctx, onlineActivityKey, redis.Z{Score: float64(time.Now().Unix()), Member: userIDStr}).Err(); err != nil {
		return 0, err
	}

	return int32(val), nil
}

func (c *cacher) DeleteSession(sessionKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userIDStr, err := c.client.Get(ctx, sessionKeyPrefix+sessionKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	if err := c.client.Del(ctx, sessionKeyPrefix+sessionKey).Err(); err != nil {
		return err
	}
	if userIDStr == "" {
		return nil
	}
	return c.logout(ctx, userIDStr)
}

// logout removes every trace of a user's presence given their string user ID.
func (c *cacher) logout(ctx context.Context, userIDStr string) error {
	if err := c.client.Del(ctx, userSessionPrefix+userIDStr).Err(); err != nil {
		return err
	}
	if err := c.client.HDel(ctx, onlineUsersKey, userIDStr).Err(); err != nil {
		return err
	}
	return c.client.ZRem(ctx, onlineActivityKey, userIDStr).Err()
}

func (c *cacher) ListOnline() ([]cache.OnlineUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	entries, err := c.client.HGetAll(ctx, onlineUsersKey).Result()
	if err != nil {
		return nil, err
	}
	users := make([]cache.OnlineUser, 0, len(entries))
	for idStr, username := range entries {
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			continue
		}
		users = append(users, cache.OnlineUser{UserID: int32(id), Username: username})
	}
	return users, nil
}

func (c *cacher) IsOnline(userID int32) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.HExists(ctx, onlineUsersKey, strconv.Itoa(int(userID))).Result()
}

func (c *cacher) PurgeInactive(maxAge time.Duration) ([]int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-maxAge).Unix()
	staleIDs, err := c.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     onlineActivityKey,
		Start:   "-inf",
		Stop:    strconv.FormatInt(cutoff, 10),
		ByScore: true,
	}).Result()
	if err != nil {
		return nil, err
	}

	purged := make([]int32, 0, len(staleIDs))
	for _, idStr := range staleIDs {
		sessionKey, err := c.client.Get(ctx, userSessionPrefix+idStr).Result()
		if err == nil {
			if delErr := c.client.Del(ctx, sessionKeyPrefix+sessionKey).Err(); delErr != nil {
				return purged, delErr
			}
		} else if !errors.Is(err, redis.Nil) {
			return purged, err
		}

		if logoutErr := c.logout(ctx, idStr); logoutErr != nil {
			return purged, logoutErr
		}

		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			continue
		}
		purged = append(purged, int32(id))
	}
	return purged, nil
}
