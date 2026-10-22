package middleware

import "net/http"

// SecurityHeaders añade cabeceras HTTP de endurecimiento de seguridad en cada respuesta.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' https:; img-src 'self' data: https:; font-src 'self' data: https:;")
		next.ServeHTTP(w, r)
	})
}
