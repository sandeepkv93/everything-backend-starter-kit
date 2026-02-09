package service

import (
	"context"
	"time"
)

type IdempotencyState string

const (
	IdempotencyStateNew        IdempotencyState = "new"
	IdempotencyStateReplay     IdempotencyState = "replay"
	IdempotencyStateConflict   IdempotencyState = "conflict"
	IdempotencyStateInProgress IdempotencyState = "in_progress"
)

type CachedHTTPResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

type IdempotencyBeginResult struct {
	State  IdempotencyState
	Cached *CachedHTTPResponse
}

type IdempotencyStore interface {
	Begin(ctx context.Context, scope, key, fingerprint string, ttl time.Duration) (IdempotencyBeginResult, error)
	Complete(ctx context.Context, scope, key, fingerprint string, response CachedHTTPResponse, ttl time.Duration) error
}
