package auth

import (
	"embed"
	"errors"
	"html/template"
	"net/http"

	"pos-inventory/internal/adapters/inbound/shared/middleware"
	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

//go:embed templates/*
var templateFS embed.FS

// Handler encapsula los endpoints de autenticación web y gestión de sesiones.
type Handler struct {
	authSvc     ports.AuthService
	rateLimiter *middleware.RateLimiter
	tmpl        *template.Template
}

// NewHandler construye el adaptador de entrada para autenticación.
func NewHandler(authSvc ports.AuthService, rateLimiter *middleware.RateLimiter) *Handler {
	tmpl, err := template.ParseFS(templateFS, "templates/login.html")
	if err != nil {
		panic("error cargando plantillas de autenticación: " + err.Error())
	}

	return &Handler{
		authSvc:     authSvc,
		rateLimiter: rateLimiter,
		tmpl:        tmpl,
	}
}

type loginPageData struct {
	Email     string
	Error     string
	Expired   bool
	CSRFToken string
}

// HandleGetLogin maneja GET /login. Si el usuario ya cuenta con sesión válida, redirige a su panel.
func (h *Handler) HandleGetLogin(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("pos_session"); err == nil && cookie.Value != "" {
		if usuario, err := h.authSvc.ValidarSesion(r.Context(), cookie.Value); err == nil && usuario != nil {
			h.redirectByRole(w, r, usuario.Rol)
			return
		}
	}

	data := loginPageData{
		Expired:   r.URL.Query().Get("expired") == "1",
		CSRFToken: middleware.CSRFToken(r),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.Execute(w, data)
}

// HandlePostLogin procesa POST /login con verificación de tasa, autenticación Argon2id y creación de cookie.
func (h *Handler) HandlePostLogin(w http.ResponseWriter, r *http.Request) {
	ip := middleware.ExtractIP(r)

	if !h.rateLimiter.Check(ip) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = h.tmpl.Execute(w, loginPageData{
			Error:     "Demasiados intentos de acceso fallidos. Por favor espere antes de reintentar.",
			CSRFToken: middleware.CSRFToken(r),
		})
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Datos de formulario inválidos", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	sesion, err := h.authSvc.Login(r.Context(), ports.LoginRequest{
		Email:    email,
		Password: password,
		IPOrigen: ip,
	})

	if err != nil {
		h.rateLimiter.RecordFailedAttempt(ip)

		errorMsg := "Credenciales inválidas"
		if errors.Is(err, domain.ErrCuentaInactiva) {
			errorMsg = "Cuenta desactivada. Contacte al administrador."
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_ = h.tmpl.Execute(w, loginPageData{
			Email:     email,
			Error:     errorMsg,
			CSRFToken: middleware.CSRFToken(r),
		})
		return
	}

	// Login exitoso: restablecer intentos para la IP
	h.rateLimiter.Reset(ip)

	// Establecer cookie HttpOnly con SameSite=Strict
	http.SetCookie(w, &http.Cookie{
		Name:     "pos_session",
		Value:    sesion.Token,
		Path:     "/",
		Expires:  sesion.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// Redirigir según el rol del usuario
	usuario, err := h.authSvc.ValidarSesion(r.Context(), sesion.Token)
	if err != nil {
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
		return
	}

	h.redirectByRole(w, r, usuario.Rol)
}

// HandleLogout invalida la sesión activa y limpia la cookie de cliente.
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("pos_session"); err == nil && cookie.Value != "" {
		_ = h.authSvc.Logout(r.Context(), cookie.Value)
	}

	middleware.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) redirectByRole(w http.ResponseWriter, r *http.Request, rol domain.Rol) {
	switch rol {
	case domain.RolCocinero:
		http.Redirect(w, r, "/kds/", http.StatusSeeOther)
	case domain.RolMesero:
		http.Redirect(w, r, "/?mesa=1", http.StatusSeeOther)
	case domain.RolAdmin, domain.RolGerente, domain.RolCajero:
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
