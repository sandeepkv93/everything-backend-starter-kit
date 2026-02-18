package service

import (
	"context"
	"sync"
	"time"
)

type FeatureFlagEvaluationCacheStore interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	InvalidateUser(ctx context.Context, userID uint) error
	InvalidateAll(ctx context.Context) error
}

type NoopFeatureFlagEvaluationCacheStore struct{}

func NewNoopFeatureFlagEvaluationCacheStore() *NoopFeatureFlagEvaluationCacheStore {
	return &NoopFeatureFlagEvaluationCacheStore{}
}

func (s *NoopFeatureFlagEvaluationCacheStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func (s *NoopFeatureFlagEvaluationCacheStore) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}

func (s *NoopFeatureFlagEvaluationCacheStore) InvalidateUser(context.Context, uint) error {
	return nil
}

func (s *NoopFeatureFlagEvaluationCacheStore) InvalidateAll(context.Context) error {
	return nil
}

type featureFlagCacheEntry struct {
	payload   []byte
	expiresAt time.Time
}

type InMemoryFeatureFlagEvaluationCacheStore struct {
	mu      sync.RWMutex
	entries map[string]featureFlagCacheEntry
}

func NewInMemoryFeatureFlagEvaluationCacheStore() *InMemoryFeatureFlagEvaluationCacheStore {
	return &InMemoryFeatureFlagEvaluationCacheStore{entries: map[string]featureFlagCacheEntry{}}
}

func (s *InMemoryFeatureFlagEvaluationCacheStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	now := time.Now().UTC()
	s.mu.RLock()
	entry, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if now.After(entry.expiresAt) {
		s.mu.Lock()
		delete(s.entries, key)
		s.mu.Unlock()
		return nil, false, nil
	}
	return append([]byte(nil), entry.payload...), true, nil
}

func (s *InMemoryFeatureFlagEvaluationCacheStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	s.mu.Lock()
	s.entries[key] = featureFlagCacheEntry{payload: append([]byte(nil), value...), expiresAt: time.Now().UTC().Add(ttl)}
	s.mu.Unlock()
	return nil
}

func (s *InMemoryFeatureFlagEvaluationCacheStore) InvalidateUser(_ context.Context, userID uint) error {
	prefix := "u:" + uintString(userID) + "|"
	s.mu.Lock()
	for key := range s.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(s.entries, key)
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *InMemoryFeatureFlagEvaluationCacheStore) InvalidateAll(_ context.Context) error {
	s.mu.Lock()
	s.entries = map[string]featureFlagCacheEntry{}
	s.mu.Unlock()
	return nil
}

func uintString(v uint) string {
	if v == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
