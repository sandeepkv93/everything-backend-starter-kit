package security

import (
	"net/http"
	"strings"
	"time"
)

type CookieManager struct {
	Domain   string
	Secure   bool
	SameSite http.SameSite
}

func NewCookieManager(domain string, secure bool, sameSite string) *CookieManager {
	ss := http.SameSiteLaxMode
	switch strings.ToLower(sameSite) {
	case "none":
		ss = http.SameSiteNoneMode
	case "strict":
		ss = http.SameSiteStrictMode
	}
	return &CookieManager{Domain: domain, Secure: secure, SameSite: ss}
}

func (c *CookieManager) SetTokenCookies(w http.ResponseWriter, accessToken, refreshToken, csrf string, refreshTTL time.Duration) {
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: accessToken, Path: "/", HttpOnly: true, Secure: c.Secure, SameSite: c.SameSite, Domain: c.Domain, MaxAge: 900})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: refreshToken, Path: "/api/v1/auth", HttpOnly: true, Secure: c.Secure, SameSite: c.SameSite, Domain: c.Domain, MaxAge: int(refreshTTL.Seconds())})
	http.SetCookie(w, &http.Cookie{Name: "csrf_token", Value: csrf, Path: "/", HttpOnly: false, Secure: c.Secure, SameSite: c.SameSite, Domain: c.Domain, MaxAge: int(refreshTTL.Seconds())})
}

func (c *CookieManager) ClearTokenCookies(w http.ResponseWriter) {
	clear := func(name, path string, httpOnly bool) {
		http.SetCookie(w, &http.Cookie{Name: name, Path: path, Value: "", MaxAge: -1, HttpOnly: httpOnly, Secure: c.Secure, SameSite: c.SameSite, Domain: c.Domain})
	}
	clear("access_token", "/", true)
	clear("refresh_token", "/api/v1/auth", true)
	clear("csrf_token", "/", false)
	clear("oauth_state", "/api/v1/auth/google", true)
}

func GetCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
