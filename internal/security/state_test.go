package security

import "testing"

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
