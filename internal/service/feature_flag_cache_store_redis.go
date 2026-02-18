package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisFeatureFlagEvaluationCacheStore struct {
	client redis.UniversalClient
	prefix string
}

func NewRedisFeatureFlagEvaluationCacheStore(client redis.UniversalClient, prefix string) *RedisFeatureFlagEvaluationCacheStore {
	if prefix == "" {
		prefix = "feature_flag_eval_cache"
	}
	return &RedisFeatureFlagEvaluationCacheStore{client: client, prefix: prefix}
}

func (s *RedisFeatureFlagEvaluationCacheStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if s.client == nil {
		return nil, false, nil
	}
	val, err := s.client.Get(ctx, s.dataKey(key)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

func (s *RedisFeatureFlagEvaluationCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s.client == nil || ttl <= 0 {
		return nil
	}
	dataKey := s.dataKey(key)
	allIndex := s.allIndexKey()
	userIndex := s.userIndexKeyFromCacheKey(key)
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, dataKey, value, ttl)
	pipe.SAdd(ctx, allIndex, dataKey)
	pipe.Expire(ctx, allIndex, ttl+time.Minute)
	if userIndex != "" {
		pipe.SAdd(ctx, userIndex, dataKey)
		pipe.Expire(ctx, userIndex, ttl+time.Minute)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisFeatureFlagEvaluationCacheStore) InvalidateUser(ctx context.Context, userID uint) error {
	if s.client == nil {
		return nil
	}
	indexKey := s.userIndexKey(userID)
	keys, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	pipe := s.client.TxPipeline()
	if len(keys) > 0 {
		pipe.Del(ctx, keys...)
	}
	pipe.Del(ctx, indexKey)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisFeatureFlagEvaluationCacheStore) InvalidateAll(ctx context.Context) error {
	if s.client == nil {
		return nil
	}
	allIndex := s.allIndexKey()
	keys, err := s.client.SMembers(ctx, allIndex).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	pipe := s.client.TxPipeline()
	if len(keys) > 0 {
		pipe.Del(ctx, keys...)
	}
	pipe.Del(ctx, allIndex)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisFeatureFlagEvaluationCacheStore) dataKey(cacheKey string) string {
	return fmt.Sprintf("%s:data:%s", s.prefix, hashToken(cacheKey))
}

func (s *RedisFeatureFlagEvaluationCacheStore) allIndexKey() string {
	return fmt.Sprintf("%s:index:all", s.prefix)
}

func (s *RedisFeatureFlagEvaluationCacheStore) userIndexKey(userID uint) string {
	return fmt.Sprintf("%s:index:user:%s", s.prefix, strconv.FormatUint(uint64(userID), 10))
}

func (s *RedisFeatureFlagEvaluationCacheStore) userIndexKeyFromCacheKey(cacheKey string) string {
	prefix := "u:"
	if !strings.HasPrefix(cacheKey, prefix) {
		return ""
	}
	rest := cacheKey[len(prefix):]
	sep := strings.Index(rest, "|")
	if sep <= 0 {
		return ""
	}
	uid := rest[:sep]
	return fmt.Sprintf("%s:index:user:%s", s.prefix, uid)
}
