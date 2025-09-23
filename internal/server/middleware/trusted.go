// internal/middleware/trusted.go
package middleware

import (
	"net"
	"net/http"
)

func TrustedCIDR(cidrs string) func(http.Handler) http.Handler {
	var ipnet *net.IPNet
	if cidrs != "" {
		_, n, err := net.ParseCIDR(cidrs)
		if err != nil {
			// если конфиг кривой — логни и запрети всё, либо отключи проверку.
			// Я бы фейлил старт сервера:
			panic("invalid trusted_subnet: " + err.Error())
		}
		ipnet = n
	}

	return func(next http.Handler) http.Handler {
		// пусто — без ограничений
		if ipnet == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			xrip := r.Header.Get("X-Real-IP")
			ip := net.ParseIP(xrip)
			if ip == nil || !ipnet.Contains(ip) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
