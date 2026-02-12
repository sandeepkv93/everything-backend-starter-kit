package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestModelUpdateAndViewStates(t *testing.T) {
	m := model{title: "Test", action: func(context.Context) ([]string, error) { return nil, nil }}
	if view := m.View(); !strings.Contains(view, "Running") {
		t.Fatalf("expected running view, got %q", view)
	}

	updated, _ := m.Update(actionMsg{details: []string{"x"}, err: nil})
	mu := updated.(model)
	if !mu.done || mu.err != nil || len(mu.details) != 1 {
		t.Fatalf("unexpected success update state: %+v", mu)
	}
	if view := mu.View(); !strings.Contains(view, "OK") {
		t.Fatalf("expected ok view, got %q", view)
	}

	updated, _ = m.Update(actionMsg{details: nil, err: errors.New("boom")})
	me := updated.(model)
	if !me.done || me.err == nil {
		t.Fatalf("unexpected error update state: %+v", me)
	}
	if view := me.View(); !strings.Contains(view, "FAILED") {
		t.Fatalf("expected failed view, got %q", view)
	}
}
