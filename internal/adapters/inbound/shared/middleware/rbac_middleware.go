package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"pos-inventory/internal/core/domain"
)

// RequireRole verifica que el usuario autenticado posea al menos uno de los roles autorizados.
func RequireRole(rolesPermitidos ...domain.Rol) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			usuario := UsuarioFromContext(r.Context())

			if usuario == nil || !usuario.PuedeAcceder(rolesPermitidos) {
				isJSON := strings.Contains(r.Header.Get("Accept"), "application/json") || strings.HasPrefix(r.URL.Path, "/api/")

				if isJSON {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "Acceso denegado: rol insuficiente para esta operación",
					})
					return
				}

				http.Error(w, "403 Forbidden: No tiene permisos suficientes para acceder a este recurso.", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
