package middleware

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newRedisLimiterForTest(t *testing.T) (*miniredis.Miniredis, *redis.Client, *RedisFixedWindowLimiter) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		m.Close()
	})
	return m, client, NewRedisFixedWindowLimiter(client, "rl_test")
}

func TestRedisFixedWindowLimiterAllowDenyAndFallbackKey(t *testing.T) {
	_, _, limiter := newRedisLimiterForTest(t)
	ctx := context.Background()
	policy := RateLimitPolicy{SustainedLimit: 1, SustainedWindow: time.Second, BurstCapacity: 1, BurstRefillPerSec: 1}

	d1, err := limiter.Allow(ctx, "", policy)
	if err != nil {
		t.Fatalf("allow first request: %v", err)
	}
	if !d1.Allowed {
		t.Fatalf("expected first request to be allowed: %+v", d1)
	}

	d2, err := limiter.Allow(ctx, "", policy)
	if err != nil {
		t.Fatalf("allow second request: %v", err)
	}
	if d2.Allowed {
		t.Fatalf("expected second request denied: %+v", d2)
	}
	if d2.RetryAfter <= 0 {
		t.Fatalf("expected positive retry-after, got %v", d2.RetryAfter)
	}
}

func TestRedisFixedWindowLimiterBackendAndNilClientErrors(t *testing.T) {
	limiter := NewRedisFixedWindowLimiter(nil, "")
	if _, err := limiter.Allow(context.Background(), "k", RateLimitPolicy{}); err == nil {
		t.Fatal("expected nil client error")
	}

	badClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 20 * time.Millisecond, ReadTimeout: 20 * time.Millisecond, WriteTimeout: 20 * time.Millisecond})
	t.Cleanup(func() { _ = badClient.Close() })
	limiter = NewRedisFixedWindowLimiter(badClient, "")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := limiter.Allow(ctx, "k", RateLimitPolicy{}); err == nil {
		t.Fatal("expected backend error")
	}
}

func TestParseRedisInt64Branches(t *testing.T) {
	if v, err := parseRedisInt64(int64(4)); err != nil || v != 4 {
		t.Fatalf("int64 parse mismatch v=%d err=%v", v, err)
	}
	if v, err := parseRedisInt64(int(3)); err != nil || v != 3 {
		t.Fatalf("int parse mismatch v=%d err=%v", v, err)
	}
	if _, err := parseRedisInt64(uint64(math.MaxUint64)); err == nil {
		t.Fatal("expected overflow error for uint64")
	}
	if _, err := parseRedisInt64("1"); err == nil {
		t.Fatal("expected string type error")
	}
	if _, err := parseRedisInt64(errors.New("x")); err == nil {
		t.Fatal("expected unexpected type error")
	}
}

func FuzzParseRedisInt64Robustness(f *testing.F) {
	f.Add(uint8(0), int64(42), uint64(42), "42")
	f.Add(uint8(1), int64(-1), uint64(math.MaxUint64), "bad")
	f.Add(uint8(2), int64(0), uint64(0), "")

	f.Fuzz(func(t *testing.T, kind uint8, si int64, ui uint64, s string) {
		if len(s) > 256 {
			s = s[:256]
		}
		var (
			v       any
			wantErr bool
			wantVal int64
		)
		switch kind % 7 {
		case 0:
			v = si
			wantVal = si
		case 1:
			v = int(si)
			wantVal = int64(int(si))
		case 2:
			v = ui
			if ui > math.MaxInt64 {
				wantErr = true
			} else {
				wantVal = int64(ui)
			}
		case 3:
			v = s
			wantErr = true
		case 4:
			v = errors.New(s)
			wantErr = true
		case 5:
			v = nil
			wantErr = true
		default:
			v = struct{ X string }{X: s}
			wantErr = true
		}

		got, err := parseRedisInt64(v)
		if wantErr {
			if err == nil {
				t.Fatalf("expected error for type %T value %v, got value=%d", v, v, got)
			}
			return
		}
		if err != nil {
			t.Fatalf("unexpected error for type %T value %v: %v", v, v, err)
		}
		if got != wantVal {
			t.Fatalf("value mismatch: got=%d want=%d (type=%T value=%v)", got, wantVal, v, v)
		}
	})
}

func FuzzRedisFixedWindowLimiterAllowKeyFallback(f *testing.F) {
	f.Add("", uint16(1), uint16(1), uint16(1), uint16(1000))
	f.Add("unknown", uint16(2), uint16(2), uint16(3), uint16(500))
	f.Add("🔥-key", uint16(5), uint16(3), uint16(10), uint16(1200))

	f.Fuzz(func(t *testing.T, key string, sustainedLimit, burstCapacity, refillPerSec, windowMS uint16) {
		if len(key) > 256 {
			key = key[:256]
		}
		key = strings.TrimSpace(key)

		m := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: m.Addr()})
		t.Cleanup(func() {
			_ = client.Close()
			m.Close()
		})

		limiter := NewRedisFixedWindowLimiter(client, "fuzz_rl")
		policy := RateLimitPolicy{
			SustainedLimit:    int(max(int64(sustainedLimit%20), 1)),
			SustainedWindow:   time.Duration(max(int64(windowMS), 1)) * time.Millisecond,
			BurstCapacity:     int(max(int64(burstCapacity%20), 1)),
			BurstRefillPerSec: max(float64(refillPerSec), 1),
		}

		ctx := context.Background()
		d1, err := limiter.Allow(ctx, key, policy)
		if err != nil {
			t.Fatalf("first allow failed: %v", err)
		}
		if d1.RetryAfter <= 0 {
			t.Fatalf("retry-after must be positive: %+v", d1)
		}
		if d1.Remaining < 0 {
			t.Fatalf("remaining must be non-negative: %+v", d1)
		}

		d2, err := limiter.Allow(ctx, key, policy)
		if err != nil {
			t.Fatalf("second allow failed: %v", err)
		}
		if d2.RetryAfter <= 0 {
			t.Fatalf("retry-after must be positive on second decision: %+v", d2)
		}
		if d2.Remaining < 0 {
			t.Fatalf("remaining must be non-negative on second decision: %+v", d2)
		}

		if key == "" {
			if err := client.FlushAll(ctx).Err(); err != nil {
				t.Fatalf("flush before empty-key check: %v", err)
			}
			dEmpty, err := limiter.Allow(ctx, "", policy)
			if err != nil {
				t.Fatalf("empty key allow failed: %v", err)
			}
			if err := client.FlushAll(ctx).Err(); err != nil {
				t.Fatalf("flush before unknown-key check: %v", err)
			}
			dUnknown, err := limiter.Allow(ctx, "unknown", policy)
			if err != nil {
				t.Fatalf("unknown key allow failed: %v", err)
			}
			if dEmpty.Allowed != dUnknown.Allowed || dEmpty.Remaining != dUnknown.Remaining {
				t.Fatalf("fallback mismatch empty vs unknown: empty=%+v unknown=%+v", dEmpty, dUnknown)
			}
		}
	})
}
