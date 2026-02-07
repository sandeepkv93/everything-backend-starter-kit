package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

func NewRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SignState(state, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(state))
	sig := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return state + "." + sig
}

func VerifySignedState(raw, secret string) (string, bool) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return "", false
	}
	expected := SignState(parts[0], secret)
	if !hmac.Equal([]byte(expected), []byte(raw)) {
		return "", false
	}
	return parts[0], true
}

func NewCSRFToken() (string, error) {
	return NewRandomString(24)
}

func RequireCSRFFromHeader(r *http.Request) error {
	cookie := GetCookie(r, "csrf_token")
	head := r.Header.Get("X-CSRF-Token")
	if cookie == "" || head == "" || head != cookie {
		return fmt.Errorf("invalid csrf token")
	}
	return nil
}
