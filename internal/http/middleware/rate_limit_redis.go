package middleware

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisFixedWindowScript = redis.NewScript(`
local now_ms = tonumber(ARGV[1])
local burst_capacity = tonumber(ARGV[2])
local refill_per_ms = tonumber(ARGV[3])
local sustained_limit = tonumber(ARGV[4])
local sustained_window_ms = tonumber(ARGV[5])

if refill_per_ms <= 0 then
  refill_per_ms = 0.001
end

local bucket_key = KEYS[1]
local window_key = KEYS[2]
local seq_key = KEYS[3]

local stored_tokens = redis.call("HGET", bucket_key, "tokens")
local stored_last = redis.call("HGET", bucket_key, "last_ms")
local tokens = burst_capacity
local last_ms = now_ms

if stored_tokens then
  tokens = tonumber(stored_tokens)
end
if stored_last then
  last_ms = tonumber(stored_last)
end
if now_ms < last_ms then
  last_ms = now_ms
end

local elapsed_ms = now_ms - last_ms
tokens = math.min(burst_capacity, tokens + (elapsed_ms * refill_per_ms))

local window_start = now_ms - sustained_window_ms
redis.call("ZREMRANGEBYSCORE", window_key, "-inf", window_start)
local current_count = redis.call("ZCARD", window_key)

local bucket_ok = tokens >= 1
local sustained_ok = current_count < sustained_limit
local allowed = 0
if bucket_ok and sustained_ok then
  allowed = 1
  tokens = tokens - 1
  local seq = redis.call("INCR", seq_key)
  local member = tostring(now_ms) .. "-" .. tostring(seq)
  redis.call("ZADD", window_key, now_ms, member)
  current_count = current_count + 1
end

local retry_bucket_ms = 0
if tokens < 1 then
  retry_bucket_ms = math.ceil((1 - tokens) / refill_per_ms)
end

local retry_sustained_ms = 0
if current_count >= sustained_limit then
  local oldest = redis.call("ZRANGE", window_key, 0, 0, "WITHSCORES")
  if oldest and oldest[2] then
    retry_sustained_ms = math.ceil((tonumber(oldest[2]) + sustained_window_ms) - now_ms)
    if retry_sustained_ms < 0 then
      retry_sustained_ms = 0
    end
  end
end

local retry_ms = retry_bucket_ms
if retry_sustained_ms > retry_ms then
  retry_ms = retry_sustained_ms
end
if retry_ms <= 0 then
  retry_ms = 1
end

local remaining_bucket = math.floor(tokens)
if remaining_bucket < 0 then
  remaining_bucket = 0
end
local remaining_sustained = sustained_limit - current_count
if remaining_sustained < 0 then
  remaining_sustained = 0
end
local remaining = remaining_bucket
if remaining_sustained < remaining then
  remaining = remaining_sustained
end

redis.call("HSET", bucket_key, "tokens", tostring(tokens), "last_ms", tostring(now_ms))
local bucket_ttl_ms = sustained_window_ms
if refill_per_ms > 0 then
  bucket_ttl_ms = math.ceil((burst_capacity / refill_per_ms))
end
if bucket_ttl_ms < sustained_window_ms then
  bucket_ttl_ms = sustained_window_ms
end
redis.call("PEXPIRE", bucket_key, bucket_ttl_ms)
redis.call("PEXPIRE", window_key, sustained_window_ms)
redis.call("PEXPIRE", seq_key, sustained_window_ms)

local reset_ms = now_ms + sustained_window_ms
if not allowed and retry_ms > 0 then
  reset_ms = now_ms + retry_ms
end
return {allowed, retry_ms, remaining, reset_ms}
`)

type RedisFixedWindowLimiter struct {
	client redis.UniversalClient
	prefix string
}

func NewRedisFixedWindowLimiter(client redis.UniversalClient, prefix string) *RedisFixedWindowLimiter {
	if prefix == "" {
		prefix = "rl"
	}
	return &RedisFixedWindowLimiter{
		client: client,
		prefix: prefix,
	}
}

func (l *RedisFixedWindowLimiter) Allow(ctx context.Context, key string, policy RateLimitPolicy) (Decision, error) {
	policy = normalizePolicy(policy)
	now := time.Now()
	nowMS := now.UnixMilli()
	if l.client == nil {
		return Decision{}, fmt.Errorf("redis client is nil")
	}
	if key == "" {
		key = "unknown"
	}
	windowMS := int(policy.SustainedWindow / time.Millisecond)
	if windowMS <= 0 {
		windowMS = 1000
	}
	refillPerMS := policy.BurstRefillPerSec / 1000.0
	storeKey := fmt.Sprintf("%s:%s", l.prefix, key)
	windowKey := storeKey + ":sw"
	seqKey := storeKey + ":seq"
	raw, err := redisFixedWindowScript.Run(
		ctx,
		l.client,
		[]string{storeKey, windowKey, seqKey},
		nowMS,
		policy.BurstCapacity,
		refillPerMS,
		policy.SustainedLimit,
		windowMS,
	).Result()
	if err != nil {
		return Decision{}, err
	}
	values, ok := raw.([]interface{})
	if !ok || len(values) != 4 {
		return Decision{}, fmt.Errorf("unexpected redis script response type")
	}

	allowedInt, err := parseRedisInt64(values[0])
	if err != nil {
		return Decision{}, err
	}
	retryMS, err := parseRedisInt64(values[1])
	if err != nil {
		return Decision{}, err
	}
	remainingInt, err := parseRedisInt64(values[2])
	if err != nil {
		return Decision{}, err
	}
	resetMS, err := parseRedisInt64(values[3])
	if err != nil {
		return Decision{}, err
	}
	if retryMS <= 0 {
		retryMS = 1
	}
	if resetMS <= nowMS {
		resetMS = nowMS + retryMS
	}
	retryAfter := time.Duration(retryMS) * time.Millisecond
	return Decision{
		Allowed:    allowedInt == 1,
		RetryAfter: retryAfter,
		Remaining:  int(max(remainingInt, 0)),
		ResetAt:    time.UnixMilli(resetMS),
	}, nil
}

func parseRedisInt64(v interface{}) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case uint64:
		if n > math.MaxInt64 {
			return 0, fmt.Errorf("redis response overflows int64")
		}
		return int64(n), nil
	case int:
		return int64(n), nil
	case string:
		return 0, fmt.Errorf("unexpected string redis response: %s", n)
	default:
		return 0, fmt.Errorf("unexpected redis response type %T", v)
	}
}
