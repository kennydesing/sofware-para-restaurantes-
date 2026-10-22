package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

type usuarioContextKey struct{}

var ctxKeyUsuario = usuarioContextKey{}

// UsuarioFromContext obtiene el usuario autenticado desde el contexto HTTP.
func UsuarioFromContext(ctx context.Context) *domain.Usuario {
	if val, ok := ctx.Value(ctxKeyUsuario).(*domain.Usuario); ok {
		return val
	}
	return nil
}

// ClearSessionCookie invalida la cookie de sesión en el cliente.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "pos_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// AuthMiddleware valida la existencia y vigencia de la sesión mediante la cookie pos_session.
func AuthMiddleware(authSvc ports.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Permitir libre acceso a recursos estáticos (CSS, JS, imágenes, iconos)
			if strings.Contains(r.URL.Path, "/static/") || r.URL.Path == "/favicon.ico" {
				next.ServeHTTP(w, r)
				return
			}

			isJSON := strings.Contains(r.Header.Get("Accept"), "application/json") || 
		strings.HasPrefix(r.URL.Path, "/api/") || 
		strings.HasPrefix(r.URL.Path, "/admin/api/") ||
		strings.HasPrefix(r.URL.Path, "/kds/api/")

			cookie, err := r.Cookie("pos_session")
			if err != nil || cookie.Value == "" {
				if isJSON {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "No autenticado. Por favor inicie sesión.",
					})
					return
				}
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			usuario, err := authSvc.ValidarSesion(r.Context(), cookie.Value)
			if err != nil {
				ClearSessionCookie(w)

				if errors.Is(err, domain.ErrSesionExpirada) {
					if isJSON {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						_ = json.NewEncoder(w).Encode(map[string]string{
							"error": "Sesión expirada. Inicie sesión nuevamente.",
						})
						return
					}
					http.Redirect(w, r, "/login?expired=1", http.StatusSeeOther)
					return
				}

				if isJSON {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "Sesión inválida.",
					})
					return
				}
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Inyectar usuario en el contexto para middlewares downstream y handlers
			ctx := context.WithValue(r.Context(), ctxKeyUsuario, usuario)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
