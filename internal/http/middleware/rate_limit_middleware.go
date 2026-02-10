package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/http/response"
	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/security"
)

type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
	Remaining  int
	ResetAt    time.Time
}

type RateLimitPolicy struct {
	SustainedLimit    int
	SustainedWindow   time.Duration
	BurstCapacity     int
	BurstRefillPerSec float64
}

type Limiter interface {
	Allow(ctx context.Context, key string, policy RateLimitPolicy) (Decision, error)
}

type FailureMode string

const (
	FailOpen   FailureMode = "fail_open"
	FailClosed FailureMode = "fail_closed"
)

type localFixedWindowLimiter struct {
	mu      sync.Mutex
	store   map[string]*localHybridState
	cleanup time.Time
}

type localHybridState struct {
	tokens     float64
	lastRefill time.Time
	hits       []time.Time
}

type RateLimiter struct {
	limiter Limiter
	policy  RateLimitPolicy
	mode    FailureMode
	scope   string
	keyFunc func(r *http.Request) string
}

func NewLocalFixedWindowLimiter() Limiter {
	return &localFixedWindowLimiter{
		store:   make(map[string]*localHybridState),
		cleanup: time.Now().Add(time.Minute),
	}
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return NewDistributedRateLimiterWithKeyAndPolicy(
		NewLocalFixedWindowLimiter(),
		newRateLimitPolicy(limit, window, 1.0),
		FailClosed,
		"local",
		nil,
	)
}

func NewDistributedRateLimiter(limiter Limiter, limit int, window time.Duration, mode FailureMode, scope string) *RateLimiter {
	return NewDistributedRateLimiterWithKeyAndPolicy(
		limiter,
		newRateLimitPolicy(limit, window, 1.0),
		mode,
		scope,
		nil,
	)
}

func NewRateLimiterWithKey(limit int, window time.Duration, keyFunc func(r *http.Request) string) *RateLimiter {
	return NewDistributedRateLimiterWithKeyAndPolicy(
		NewLocalFixedWindowLimiter(),
		newRateLimitPolicy(limit, window, 1.0),
		FailClosed,
		"local",
		keyFunc,
	)
}

func NewDistributedRateLimiterWithKey(
	limiter Limiter,
	limit int,
	window time.Duration,
	mode FailureMode,
	scope string,
	keyFunc func(r *http.Request) string,
) *RateLimiter {
	return NewDistributedRateLimiterWithKeyAndPolicy(
		limiter,
		newRateLimitPolicy(limit, window, 1.0),
		mode,
		scope,
		keyFunc,
	)
}

func NewRateLimiterWithPolicy(policy RateLimitPolicy, keyFunc func(r *http.Request) string) *RateLimiter {
	return NewDistributedRateLimiterWithKeyAndPolicy(NewLocalFixedWindowLimiter(), policy, FailClosed, "local", keyFunc)
}

func NewDistributedRateLimiterWithKeyAndPolicy(
	limiter Limiter,
	policy RateLimitPolicy,
	mode FailureMode,
	scope string,
	keyFunc func(r *http.Request) string,
) *RateLimiter {
	if scope == "" {
		scope = "api"
	}
	if keyFunc == nil {
		keyFunc = clientIPKey
	}
	policy = normalizePolicy(policy)
	return &RateLimiter{
		limiter: limiter,
		policy:  policy,
		mode:    mode,
		scope:   scope,
		keyFunc: keyFunc,
	}
}

func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rl.keyFunc(r)
			if key == "" {
				key = clientIPKey(r)
			}
			decision, err := rl.limiter.Allow(r.Context(), key, rl.policy)
			if err != nil {
				if rl.mode == FailOpen {
					slog.Warn("rate limiter backend unavailable, allowing request",
						"scope", rl.scope,
						"mode", string(rl.mode),
						"error", err.Error(),
					)
					next.ServeHTTP(w, r)
					return
				}
				writeRateLimitHeaders(w.Header(), rl.policy.SustainedLimit, 0, time.Now().Add(rl.policy.SustainedWindow))
				w.Header().Set("Retry-After", retryAfterHeader(rl.policy.SustainedWindow))
				response.Error(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests", nil)
				return
			}
			writeRateLimitHeaders(w.Header(), rl.policy.SustainedLimit, decision.Remaining, decision.ResetAt)
			if !decision.Allowed {
				w.Header().Set("Retry-After", retryAfterHeader(decision.RetryAfter))
				response.Error(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func SubjectOrIPKeyFunc(jwtMgr *security.JWTManager) func(r *http.Request) string {
	return func(r *http.Request) string {
		if jwtMgr == nil {
			return clientIPKey(r)
		}

		raw := security.GetCookie(r, "access_token")
		if raw == "" {
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			if len(auth) >= len("bearer ")+1 && strings.EqualFold(auth[:len("bearer ")], "bearer ") {
				raw = strings.TrimSpace(auth[len("bearer "):])
			}
		}
		if raw == "" {
			return clientIPKey(r)
		}

		claims, err := jwtMgr.ParseAccessToken(raw)
		if err != nil || claims == nil {
			return clientIPKey(r)
		}
		subject := strings.TrimSpace(claims.Subject)
		if subject == "" {
			return clientIPKey(r)
		}
		return "sub:" + subject
	}
}

func (rl *localFixedWindowLimiter) Allow(_ context.Context, key string, policy RateLimitPolicy) (Decision, error) {
	policy = normalizePolicy(policy)
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if now.After(rl.cleanup) {
		for k, v := range rl.store {
			if len(v.hits) == 0 && now.Sub(v.lastRefill) > 2*policy.SustainedWindow {
				delete(rl.store, k)
			}
		}
		rl.cleanup = now.Add(policy.SustainedWindow)
	}

	state, ok := rl.store[key]
	if !ok {
		state = &localHybridState{
			tokens:     float64(policy.BurstCapacity),
			lastRefill: now,
			hits:       nil,
		}
		rl.store[key] = state
	}
	if now.After(state.lastRefill) {
		elapsed := now.Sub(state.lastRefill).Seconds()
		state.tokens = min(float64(policy.BurstCapacity), state.tokens+(elapsed*policy.BurstRefillPerSec))
		state.lastRefill = now
	}

	cutoff := now.Add(-policy.SustainedWindow)
	pruned := state.hits[:0]
	for _, hit := range state.hits {
		if hit.After(cutoff) {
			pruned = append(pruned, hit)
		}
	}
	state.hits = pruned

	sustainedRemaining := policy.SustainedLimit - len(state.hits)
	bucketRetry := time.Duration(0)
	if state.tokens < 1 {
		need := 1 - state.tokens
		bucketRetry = time.Duration(math.Ceil((need / policy.BurstRefillPerSec) * float64(time.Second)))
	}
	sustainedRetry := time.Duration(0)
	if sustainedRemaining <= 0 {
		sustainedRetry = state.hits[0].Add(policy.SustainedWindow).Sub(now)
		if sustainedRetry < 0 {
			sustainedRetry = 0
		}
	}

	allowed := bucketRetry <= 0 && sustainedRetry <= 0
	if allowed {
		state.tokens = max(state.tokens-1, 0)
		state.hits = append(state.hits, now)
		sustainedRemaining = policy.SustainedLimit - len(state.hits)
	}

	bucketRemaining := int(math.Floor(state.tokens))
	if bucketRemaining < 0 {
		bucketRemaining = 0
	}
	if sustainedRemaining < 0 {
		sustainedRemaining = 0
	}
	remaining := min(bucketRemaining, sustainedRemaining)
	retryAfter := max(bucketRetry, sustainedRetry)
	if !allowed && retryAfter <= 0 {
		retryAfter = time.Second
	}

	resetAt := now.Add(policy.SustainedWindow)
	if len(state.hits) > 0 {
		resetAt = state.hits[0].Add(policy.SustainedWindow)
	}
	if !allowed {
		resetAt = now.Add(retryAfter)
	}

	return Decision{
		Allowed:    allowed,
		RetryAfter: retryAfter,
		Remaining:  remaining,
		ResetAt:    resetAt,
	}, nil
}

func clientIPKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func retryAfterHeader(d time.Duration) string {
	if d <= 0 {
		return "1"
	}
	seconds := int(d.Round(time.Second).Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return fmt.Sprintf("%d", seconds)
}

func writeRateLimitHeaders(h http.Header, limit int, remaining int, resetAt time.Time) {
	h.Set("X-RateLimit-Limit", fmt.Sprintf("%d", max(limit, 0)))
	h.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(remaining, 0)))
	if resetAt.IsZero() {
		resetAt = time.Now().Add(time.Second)
	}
	h.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))
}

func newRateLimitPolicy(limit int, window time.Duration, burstMultiplier float64) RateLimitPolicy {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	if burstMultiplier < 1 {
		burstMultiplier = 1
	}
	burst := int(math.Ceil(float64(limit) * burstMultiplier))
	if burst < limit {
		burst = limit
	}
	refill := float64(limit) / window.Seconds()
	if refill <= 0 {
		refill = 1
	}
	return RateLimitPolicy{
		SustainedLimit:    limit,
		SustainedWindow:   window,
		BurstCapacity:     burst,
		BurstRefillPerSec: refill,
	}
}

func normalizePolicy(policy RateLimitPolicy) RateLimitPolicy {
	if policy.SustainedLimit <= 0 {
		policy.SustainedLimit = 1
	}
	if policy.SustainedWindow <= 0 {
		policy.SustainedWindow = time.Minute
	}
	if policy.BurstCapacity <= 0 {
		policy.BurstCapacity = policy.SustainedLimit
	}
	if policy.BurstCapacity < policy.SustainedLimit {
		policy.BurstCapacity = policy.SustainedLimit
	}
	if policy.BurstRefillPerSec <= 0 {
		policy.BurstRefillPerSec = float64(policy.SustainedLimit) / policy.SustainedWindow.Seconds()
	}
	if policy.BurstRefillPerSec <= 0 {
		policy.BurstRefillPerSec = 1
	}
	return policy
}
