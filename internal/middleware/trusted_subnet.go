package middleware

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

// TrustedSubnetMiddleware — HTTP мидлварь, пропускающая запрос только если
// IP-адрес агента, переданный в заголовке X-Real-IP, входит в доверенную
// подсеть trustedSubnet. Иначе возвращает 403 Forbidden.
func TrustedSubnetMiddleware(h http.HandlerFunc, trustedSubnet *net.IPNet, logger *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realIP := net.ParseIP(r.Header.Get("X-Real-IP"))
		if realIP == nil || !trustedSubnet.Contains(realIP) {
			logger.Warnw("Rejected request from untrusted subnet", "x-real-ip", r.Header.Get("X-Real-IP"))
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	}
}
