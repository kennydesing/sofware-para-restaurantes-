package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"pos-inventory/internal/adapters/inbound/auth"
	"pos-inventory/internal/adapters/inbound/shared/middleware"
	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
	"pos-inventory/internal/core/services"
)

// Mock repos para testing de handler
type mockUserRepo struct {
	usuarios map[string]*domain.Usuario
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int) (*domain.Usuario, error) {
	for _, u := range m.usuarios {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.Usuario, error) {
	if u, exists := m.usuarios[email]; exists {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetAll(ctx context.Context) ([]*domain.Usuario, error) {
	var list []*domain.Usuario
	for _, u := range m.usuarios {
		list = append(list, u)
	}
	return list, nil
}

func (m *mockUserRepo) Create(ctx context.Context, u *domain.Usuario) error {
	u.ID = len(m.usuarios) + 1
	copyU := *u
	m.usuarios[u.Email] = &copyU
	return nil
}

func (m *mockUserRepo) Update(ctx context.Context, u *domain.Usuario) error {
	m.usuarios[u.Email] = u
	return nil
}

type mockSessRepo struct {
	sesiones map[string]*domain.Sesion
}

func (m *mockSessRepo) Create(ctx context.Context, s *domain.Sesion) error {
	m.sesiones[s.Token] = s
	return nil
}

func (m *mockSessRepo) GetByToken(ctx context.Context, token string) (*domain.Sesion, error) {
	if s, exists := m.sesiones[token]; exists {
		return s, nil
	}
	return nil, domain.ErrSesionInvalida
}

func (m *mockSessRepo) DeleteByToken(ctx context.Context, token string) error {
	delete(m.sesiones, token)
	return nil
}

func (m *mockSessRepo) DeleteByUsuarioID(ctx context.Context, usuarioID int) error {
	for t, s := range m.sesiones {
		if s.UsuarioID == usuarioID {
			delete(m.sesiones, t)
		}
	}
	return nil
}

func (m *mockSessRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

type mockAuditRepo struct{}

func (m *mockAuditRepo) Create(ctx context.Context, e *domain.EventoAuditoria) error {
	return nil
}

func (m *mockAuditRepo) GetFiltered(ctx context.Context, f ports.FiltroAuditoria) ([]*domain.EventoAuditoria, error) {
	return nil, nil
}

func setupTestAuth(t *testing.T) (*auth.Handler, ports.AuthService) {
	uRepo := &mockUserRepo{usuarios: make(map[string]*domain.Usuario)}
	sRepo := &mockSessRepo{sesiones: make(map[string]*domain.Sesion)}
	aRepo := &mockAuditRepo{}

	authSvc := services.NewAuthService(uRepo, sRepo, aRepo)

	// Crear usuario admin de prueba
	_, err := authSvc.CrearUsuario(context.Background(), ports.CrearUsuarioRequest{
		Nombre:   "Admin Test",
		Email:    "admin@test.com",
		Password: "Password123",
		Rol:      domain.RolAdmin,
	})
	if err != nil {
		t.Fatalf("Error configurando usuario test: %v", err)
	}

	rateLimiter := middleware.NewRateLimiter()
	handler := auth.NewHandler(authSvc, rateLimiter)

	return handler, authSvc
}

func TestHandler_HandleGetLogin(t *testing.T) {
	handler, _ := setupTestAuth(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	handler.HandleGetLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET /login esperó status 200, obtuvo %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "POS RESTAURANTE") {
		t.Errorf("GET /login cuerpo no contiene título esperado")
	}
}

func TestHandler_HandlePostLogin_Exito(t *testing.T) {
	handler, _ := setupTestAuth(t)

	formData := url.Values{}
	formData.Set("email", "admin@test.com")
	formData.Set("password", "Password123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.HandlePostLogin(rr, req)

	// Espera redirección (303 See Other)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("POST /login exitoso esperó 303, obtuvo %d", rr.Code)
	}

	// Verificar cookie
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "pos_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("POST /login exitoso no estableció la cookie pos_session")
	}
	if !sessionCookie.HttpOnly {
		t.Error("Cookie pos_session debe ser HttpOnly")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Error("Cookie pos_session debe ser SameSite=Strict")
	}
}

func TestHandler_HandlePostLogin_CredencialesInvalidas(t *testing.T) {
	handler, _ := setupTestAuth(t)

	formData := url.Values{}
	formData.Set("email", "admin@test.com")
	formData.Set("password", "PassErronea999")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.HandlePostLogin(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("POST /login fallido esperó 401, obtuvo %d", rr.Code)
	}
}

func TestHandler_HandleLogout(t *testing.T) {
	handler, _ := setupTestAuth(t)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "pos_session", Value: "token-de-prueba"})
	rr := httptest.NewRecorder()

	handler.HandleLogout(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("POST /logout esperó 303, obtuvo %d", rr.Code)
	}

	// Verificar cookie borrada (MaxAge = -1)
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "pos_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil || sessionCookie.MaxAge != -1 {
		t.Error("POST /logout debe limpiar la cookie pos_session con MaxAge=-1")
	}
}
