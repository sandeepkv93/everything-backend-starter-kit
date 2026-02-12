package security

import (
	"strings"
	"testing"
)

func TestStateSignAndVerify(t *testing.T) {
	raw, err := NewRandomString(16)
	if err != nil {
		t.Fatal(err)
	}
	signed := SignState(raw, "state-secret-123456")
	parsed, ok := VerifySignedState(signed, "state-secret-123456")
	if !ok || parsed != raw {
		t.Fatalf("verify failed: %v %s", ok, parsed)
	}
	if _, ok := VerifySignedState(signed, "wrong-secret"); ok {
		t.Fatal("expected verification failure with wrong secret")
	}
}

func FuzzVerifySignedStateRobustness(f *testing.F) {
	f.Add("simple-state", "state-secret-123456", "")
	f.Add("unicode-\u2603-\U0001f680", "unicode-secret-🔥", "")
	f.Add(strings.Repeat("s", 2048), strings.Repeat("k", 128), "")
	f.Add("", "empty-state-secret", "malformed.signed.value")

	f.Fuzz(func(t *testing.T, state, secret, arbitraryRaw string) {
		if len(state) > 4096 {
			state = state[:4096]
		}
		if len(secret) > 512 {
			secret = secret[:512]
		}
		if len(arbitraryRaw) > 8192 {
			arbitraryRaw = arbitraryRaw[:8192]
		}

		signed := SignState(state, secret)
		parsed, ok := VerifySignedState(signed, secret)
		if strings.Contains(state, ".") {
			if ok {
				t.Fatalf("expected dotted state verification to fail, parsed=%q", parsed)
			}
		} else if !ok || parsed != state {
			t.Fatalf("signed state verify failed: ok=%v parsed=%q want=%q", ok, parsed, state)
		}

		if secret != "" {
			if _, okWrong := VerifySignedState(signed, secret+"-wrong"); okWrong {
				t.Fatal("expected wrong secret verification to fail")
			}
		}

		_, _ = VerifySignedState(arbitraryRaw, secret)
	})
}
