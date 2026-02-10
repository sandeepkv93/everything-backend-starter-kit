package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/security"
)

type BypassEvaluator func(r *http.Request) (bool, string)

type RequestBypassConfig struct {
	EnableInternalProbeBypass bool
	EnableTrustedActorBypass  bool
	TrustedActorCIDRs         []string
	TrustedActorSubjects      []string
}

type requestBypassMatcher struct {
	enableProbeBypass   bool
	enableTrustedBypass bool
	trustedCIDRs        []*net.IPNet
	trustedSubjects     map[string]struct{}
	jwtMgr              *security.JWTManager
}

func NewRequestBypassEvaluator(cfg RequestBypassConfig, jwtMgr *security.JWTManager) BypassEvaluator {
	m := &requestBypassMatcher{
		enableProbeBypass:   cfg.EnableInternalProbeBypass,
		enableTrustedBypass: cfg.EnableTrustedActorBypass,
		trustedCIDRs:        make([]*net.IPNet, 0, len(cfg.TrustedActorCIDRs)),
		trustedSubjects:     make(map[string]struct{}, len(cfg.TrustedActorSubjects)),
		jwtMgr:              jwtMgr,
	}

	for _, cidr := range cfg.TrustedActorCIDRs {
		v := strings.TrimSpace(cidr)
		if v == "" {
			continue
		}
		_, network, err := net.ParseCIDR(v)
		if err != nil {
			continue
		}
		m.trustedCIDRs = append(m.trustedCIDRs, network)
	}
	for _, subject := range cfg.TrustedActorSubjects {
		v := strings.TrimSpace(subject)
		if v == "" {
			continue
		}
		m.trustedSubjects[v] = struct{}{}
	}

	if !m.enableProbeBypass && (!m.enableTrustedBypass || (len(m.trustedCIDRs) == 0 && len(m.trustedSubjects) == 0)) {
		return nil
	}
	return m.Match
}

func (m *requestBypassMatcher) Match(r *http.Request) (bool, string) {
	if r == nil {
		return false, ""
	}
	if m.enableProbeBypass {
		switch strings.TrimSpace(strings.ToLower(r.URL.Path)) {
		case "/health/live", "/health/ready":
			return true, "internal_probe_path"
		}
	}
	if !m.enableTrustedBypass {
		return false, ""
	}

	if ip := parseRequestIP(r); ip != nil {
		for _, network := range m.trustedCIDRs {
			if network.Contains(ip) {
				return true, "trusted_actor_cidr"
			}
		}
	}

	if len(m.trustedSubjects) > 0 {
		subject := requestSubject(r, m.jwtMgr)
		if _, ok := m.trustedSubjects[subject]; ok {
			return true, "trusted_actor_subject"
		}
	}
	return false, ""
}

func requestSubject(r *http.Request, jwtMgr *security.JWTManager) string {
	if jwtMgr == nil || r == nil {
		return ""
	}

	raw := security.GetCookie(r, "access_token")
	if raw == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if len(auth) >= len("bearer ")+1 && strings.EqualFold(auth[:len("bearer ")], "bearer ") {
			raw = strings.TrimSpace(auth[len("bearer "):])
		}
	}
	if raw == "" {
		return ""
	}

	claims, err := jwtMgr.ParseAccessToken(raw)
	if err != nil || claims == nil {
		return ""
	}
	return strings.TrimSpace(claims.Subject)
}

func parseRequestIP(r *http.Request) net.IP {
	if r == nil {
		return nil
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil || host == "" {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	if host == "" {
		return nil
	}
	return net.ParseIP(host)
}
