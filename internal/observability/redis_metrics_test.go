package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestClassifyKeyspaceOutcomeGet(t *testing.T) {
	getMiss := redis.NewStringCmd(context.Background(), "get", "k")
	getMiss.SetErr(redis.Nil)
	hits, misses, ok := classifyKeyspaceOutcome(getMiss)
	if !ok || hits != 0 || misses != 1 {
		t.Fatalf("expected miss classification, got ok=%v hits=%d misses=%d", ok, hits, misses)
	}

	getHit := redis.NewStringCmd(context.Background(), "get", "k")
	getHit.SetVal("value")
	hits, misses, ok = classifyKeyspaceOutcome(getHit)
	if !ok || hits != 1 || misses != 0 {
		t.Fatalf("expected hit classification, got ok=%v hits=%d misses=%d", ok, hits, misses)
	}
}

func TestClassifyKeyspaceOutcomeMGet(t *testing.T) {
	cmd := redis.NewSliceCmd(context.Background(), "mget", "a", "b", "c", "d")
	cmd.SetVal([]interface{}{"a", nil, "b", nil})
	hits, misses, ok := classifyKeyspaceOutcome(cmd)
	if !ok || hits != 2 || misses != 2 {
		t.Fatalf("expected mget 2 hits 2 misses, got ok=%v hits=%d misses=%d", ok, hits, misses)
	}
}

func TestClassifyRedisError(t *testing.T) {
	if got := classifyRedisError(errors.New("dial timeout")); got != "timeout" {
		t.Fatalf("expected timeout, got %s", got)
	}
	if got := classifyRedisError(errors.New("connection reset by peer")); got != "connection" {
		t.Fatalf("expected connection, got %s", got)
	}
	if got := classifyRedisError(errors.New("unknown error")); got != "other" {
		t.Fatalf("expected other, got %s", got)
	}
}

func TestClampRatio(t *testing.T) {
	if clampRatio(-1) != 0 {
		t.Fatal("expected lower clamp to 0")
	}
	if clampRatio(2) != 1 {
		t.Fatal("expected upper clamp to 1")
	}
	if clampRatio(0.5) != 0.5 {
		t.Fatal("expected in-range value unchanged")
	}
}
