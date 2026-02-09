package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisIdempotencyBeginScript = redis.NewScript(`
local key = KEYS[1]
local fingerprint = ARGV[1]
local ttl_ms = ARGV[2]

if redis.call("EXISTS", key) == 0 then
  redis.call("HSET", key, "fingerprint", fingerprint, "status", "new")
  redis.call("PEXPIRE", key, ttl_ms)
  return {"new"}
end

local existing_fp = redis.call("HGET", key, "fingerprint")
if existing_fp ~= fingerprint then
  return {"conflict"}
end

local status = redis.call("HGET", key, "status")
if status == "completed" then
  return {"replay", redis.call("HGET", key, "response_status") or "", redis.call("HGET", key, "content_type") or "", redis.call("HGET", key, "response_body") or ""}
end

return {"in_progress"}
`)

var redisIdempotencyCompleteScript = redis.NewScript(`
local key = KEYS[1]
local fingerprint = ARGV[1]
local ttl_ms = ARGV[2]
local status_code = ARGV[3]
local content_type = ARGV[4]
local response_body = ARGV[5]

if redis.call("EXISTS", key) == 0 then
  return 0
end
if redis.call("HGET", key, "fingerprint") ~= fingerprint then
  return -1
end

redis.call("HSET", key, "status", "completed", "response_status", status_code, "content_type", content_type, "response_body", response_body)
redis.call("PEXPIRE", key, ttl_ms)
return 1
`)

type RedisIdempotencyStore struct {
	client redis.UniversalClient
	prefix string
}

func NewRedisIdempotencyStore(client redis.UniversalClient, prefix string) *RedisIdempotencyStore {
	if prefix == "" {
		prefix = "idem"
	}
	return &RedisIdempotencyStore{client: client, prefix: prefix}
}

func (s *RedisIdempotencyStore) redisKey(scope, key string) string {
	return fmt.Sprintf("%s:%s:%s", s.prefix, scope, key)
}

func (s *RedisIdempotencyStore) Begin(ctx context.Context, scope, key, fingerprint string, ttl time.Duration) (IdempotencyBeginResult, error) {
	raw, err := redisIdempotencyBeginScript.Run(
		ctx,
		s.client,
		[]string{s.redisKey(scope, key)},
		fingerprint,
		int(ttl/time.Millisecond),
	).Result()
	if err != nil {
		return IdempotencyBeginResult{}, err
	}
	values, ok := raw.([]interface{})
	if !ok || len(values) == 0 {
		return IdempotencyBeginResult{}, fmt.Errorf("unexpected redis begin result type")
	}
	state := asString(values[0])
	switch IdempotencyState(state) {
	case IdempotencyStateNew:
		return IdempotencyBeginResult{State: IdempotencyStateNew}, nil
	case IdempotencyStateConflict:
		return IdempotencyBeginResult{State: IdempotencyStateConflict}, nil
	case IdempotencyStateInProgress:
		return IdempotencyBeginResult{State: IdempotencyStateInProgress}, nil
	case IdempotencyStateReplay:
		if len(values) < 4 {
			return IdempotencyBeginResult{}, fmt.Errorf("unexpected replay payload")
		}
		status, parseErr := strconv.Atoi(asString(values[1]))
		if parseErr != nil {
			return IdempotencyBeginResult{}, fmt.Errorf("parse replay status: %w", parseErr)
		}
		decoded, decodeErr := base64.StdEncoding.DecodeString(asString(values[3]))
		if decodeErr != nil {
			return IdempotencyBeginResult{}, fmt.Errorf("decode replay body: %w", decodeErr)
		}
		return IdempotencyBeginResult{
			State: IdempotencyStateReplay,
			Cached: &CachedHTTPResponse{
				StatusCode:  status,
				ContentType: asString(values[2]),
				Body:        decoded,
			},
		}, nil
	default:
		return IdempotencyBeginResult{}, fmt.Errorf("unknown idempotency state %q", state)
	}
}

func (s *RedisIdempotencyStore) Complete(ctx context.Context, scope, key, fingerprint string, response CachedHTTPResponse, ttl time.Duration) error {
	_, err := redisIdempotencyCompleteScript.Run(
		ctx,
		s.client,
		[]string{s.redisKey(scope, key)},
		fingerprint,
		int(ttl/time.Millisecond),
		response.StatusCode,
		response.ContentType,
		base64.StdEncoding.EncodeToString(response.Body),
	).Result()
	return err
}

func asString(v interface{}) string {
	switch typed := v.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(v)
	}
}
