package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"net/http"
	"strings"
)

var csrfSecret = []byte("pos-inventory-secret-csrf-key-2026")

type csrfContextKey struct{}

var ctxKeyCSRF = csrfContextKey{}

// GenerateCSRFToken deriva un token CSRF criptográfico a partir del token de sesión.
func GenerateCSRFToken(sessionToken string) string {
	if sessionToken == "" {
		return ""
	}
	h := hmac.New(sha256.New, csrfSecret)
	h.Write([]byte(sessionToken))
	return hex.EncodeToString(h.Sum(nil))
}

// CSRFTokenFromContext extrae el token CSRF disponible en el contexto.
func CSRFTokenFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(ctxKeyCSRF).(string); ok {
		return val
	}
	return ""
}

// CSRFToken extrae o deriva el token CSRF correspondiente a la sesión de la petición.
func CSRFToken(r *http.Request) string {
	if token := CSRFTokenFromContext(r.Context()); token != "" {
		return token
	}
	if cookie, err := r.Cookie("pos_session"); err == nil && cookie.Value != "" {
		return GenerateCSRFToken(cookie.Value)
	}
	return ""
}

// CSRFField genera un campo HTML input hidden con el token CSRF para formularios.
func CSRFField(r *http.Request) template.HTML {
	tok := CSRFToken(r)
	if tok == "" {
		return ""
	}
	return template.HTML(`<input type="hidden" name="_csrf" value="` + tok + `">`)
}

// CSRFMiddleware valida el token CSRF en peticiones mutables (POST, PUT, DELETE) cuando hay sesión activa.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sessionToken string
		if cookie, err := r.Cookie("pos_session"); err == nil {
			sessionToken = cookie.Value
		}

		csrfTok := ""
		if sessionToken != "" {
			csrfTok = GenerateCSRFToken(sessionToken)
			r = r.WithContext(context.WithValue(r.Context(), ctxKeyCSRF, csrfTok))
		}

		// Métodos seguros (GET, HEAD, OPTIONS) no requieren verificación CSRF
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Si no hay cookie de sesión activa (ej: login inicial), se permite el paso
		if sessionToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		// En peticiones mutadas autenticadas, validar token CSRF
		providedToken := r.Header.Get("X-CSRF-Token")
		if providedToken == "" {
			providedToken = r.FormValue("_csrf")
		}

		if providedToken == "" || !hmac.Equal([]byte(providedToken), []byte(csrfTok)) {
			if strings.Contains(r.Header.Get("Accept"), "application/json") || strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"Token CSRF inválido o ausente"}`))
				return
			}
			http.Error(w, "403 Forbidden: Token CSRF inválido o ausente", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
