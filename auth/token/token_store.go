package auth

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type TokenStore struct {
	RedisClient *redis.Client
}

func NewTokenStore(redisAddr string) *TokenStore {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	return &TokenStore{RedisClient: rdb}
}

// Refresh Token'ı Redis'te saklama (kullanıcı ID ile birlikte)
func (store *TokenStore) StoreRefreshToken(userID uint, token string, duration time.Duration) error {
	key := createRedisKey(userID, token)
	return store.RedisClient.Set(ctx, key, userID, duration).Err()
}

// Refresh Token'ı Redis'ten çekme
func (store *TokenStore) FetchRefreshToken(userID uint, token string) (uint, error) {
	var key string
	key = createRedisKey(userID, token)
	storedUserID, err := store.RedisClient.Get(ctx, key).Uint64()
	return uint(storedUserID), err
}

// Refresh Token'ı Redis'ten silme
func (store *TokenStore) DeleteRefreshToken(userID uint, token string) error {
	key := createRedisKey(userID, token)
	return store.RedisClient.Del(ctx, key).Err()
}

func createRedisKey(userID uint, token string) string {
	return "refresh_token:" + token
}
