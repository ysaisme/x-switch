package security

import (
	"net/http"
	"strings"
)

type AuthMiddleware struct {
	token      string
	allowedIPs map[string]bool
}

func NewAuthMiddleware(token string, allowedIPs []string) *AuthMiddleware {
	ips := make(map[string]bool)
	for _, ip := range allowedIPs {
		ips[ip] = true
	}

	return &AuthMiddleware{
		token:      token,
		allowedIPs: ips,
	}
}

func (a *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.checkIP(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if a.token != "" && !a.checkToken(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *AuthMiddleware) checkIP(r *http.Request) bool {
	if len(a.allowedIPs) == 0 {
		return true
	}

	ip := extractIP(r)
	return a.allowedIPs[ip]
}

func (a *AuthMiddleware) checkToken(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return false
	}

	token := parts[1]
	return token == a.token
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}
