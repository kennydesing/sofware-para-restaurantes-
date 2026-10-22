package services_test

import (
	"context"
	"testing"
	"time"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
	"pos-inventory/internal/core/services"
)

// MockUsuarioRepository implementa ports.UsuarioRepository en memoria para pruebas
type mockUsuarioRepo struct {
	usuarios []*domain.Usuario
	lastID   int
}

func (m *mockUsuarioRepo) GetByID(ctx context.Context, id int) (*domain.Usuario, error) {
	for _, u := range m.usuarios {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUsuarioRepo) GetByEmail(ctx context.Context, email string) (*domain.Usuario, error) {
	for _, u := range m.usuarios {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUsuarioRepo) GetAll(ctx context.Context) ([]*domain.Usuario, error) {
	return m.usuarios, nil
}

func (m *mockUsuarioRepo) Create(ctx context.Context, u *domain.Usuario) error {
	for _, existing := range m.usuarios {
		if existing.Email == u.Email {
			return domain.ErrEmailDuplicado
		}
	}
	m.lastID++
	u.ID = m.lastID
	copyU := *u
	m.usuarios = append(m.usuarios, &copyU)
	return nil
}

func (m *mockUsuarioRepo) Update(ctx context.Context, u *domain.Usuario) error {
	for i, existing := range m.usuarios {
		if existing.ID == u.ID {
			m.usuarios[i] = u
			return nil
		}
	}
	return domain.ErrNotFound
}

// MockSesionRepository implementa ports.SesionRepository en memoria
type mockSesionRepo struct {
	sesiones map[string]*domain.Sesion
}

func (m *mockSesionRepo) Create(ctx context.Context, s *domain.Sesion) error {
	m.sesiones[s.Token] = s
	return nil
}

func (m *mockSesionRepo) GetByToken(ctx context.Context, token string) (*domain.Sesion, error) {
	if s, exists := m.sesiones[token]; exists {
		return s, nil
	}
	return nil, domain.ErrSesionInvalida
}

func (m *mockSesionRepo) DeleteByToken(ctx context.Context, token string) error {
	delete(m.sesiones, token)
	return nil
}

func (m *mockSesionRepo) DeleteByUsuarioID(ctx context.Context, usuarioID int) error {
	for token, s := range m.sesiones {
		if s.UsuarioID == usuarioID {
			delete(m.sesiones, token)
		}
	}
	return nil
}

func (m *mockSesionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	var count int64
	now := time.Now()
	for token, s := range m.sesiones {
		if s.ExpiresAt.Before(now) {
			delete(m.sesiones, token)
			count++
		}
	}
	return count, nil
}

// MockAuditoriaRepository implementa ports.AuditoriaRepository
type mockAuditoriaRepo struct {
	eventos []*domain.EventoAuditoria
}

func (m *mockAuditoriaRepo) Create(ctx context.Context, e *domain.EventoAuditoria) error {
	m.eventos = append(m.eventos, e)
	return nil
}

func (m *mockAuditoriaRepo) GetFiltered(ctx context.Context, filtro ports.FiltroAuditoria) ([]*domain.EventoAuditoria, error) {
	return m.eventos, nil
}

func TestAuthService_FlujoCompleto(t *testing.T) {
	ctx := context.Background()
	uRepo := &mockUsuarioRepo{usuarios: make([]*domain.Usuario, 0)}
	sRepo := &mockSesionRepo{sesiones: make(map[string]*domain.Sesion)}
	aRepo := &mockAuditoriaRepo{eventos: make([]*domain.EventoAuditoria, 0)}

	authSvc := services.NewAuthService(uRepo, sRepo, aRepo)

	// 1. Crear usuario con contraseña insegura debe fallar
	_, err := authSvc.CrearUsuario(ctx, ports.CrearUsuarioRequest{
		Nombre:   "Admin Test",
		Email:    "admin@test.com",
		Password: "corta", // Menos de 8 caracteres
		Rol:      domain.RolAdmin,
	})
	if err != domain.ErrPasswordInsegura {
		t.Fatalf("Esperaba ErrPasswordInsegura para pass corta, obtuve: %v", err)
	}

	// 2. Crear usuario con contraseña válida
	usuario, err := authSvc.CrearUsuario(ctx, ports.CrearUsuarioRequest{
		Nombre:   "Admin Test",
		Email:    "admin@test.com",
		Password: "Password123",
		Rol:      domain.RolAdmin,
	})
	if err != nil {
		t.Fatalf("Error creando usuario válido: %v", err)
	}
	if usuario.ID == 0 || usuario.Email != "admin@test.com" {
		t.Errorf("Usuario creado incorrectamente: %+v", usuario)
	}

	// 3. Intento de email duplicado debe fallar
	_, err = authSvc.CrearUsuario(ctx, ports.CrearUsuarioRequest{
		Nombre:   "Duplicado",
		Email:    "admin@test.com",
		Password: "Password123",
		Rol:      domain.RolAdmin,
	})
	if err != domain.ErrEmailDuplicado {
		t.Fatalf("Esperaba ErrEmailDuplicado, obtuve: %v", err)
	}

	// 4. Login con contraseña incorrecta debe fallar
	_, err = authSvc.Login(ctx, ports.LoginRequest{
		Email:    "admin@test.com",
		Password: "PasswordIncorrecta99",
		IPOrigen: "127.0.0.1",
	})
	if err != domain.ErrCredencialesInvalidas {
		t.Fatalf("Esperaba ErrCredencialesInvalidas por pass incorrecta, obtuve: %v", err)
	}

	// 5. Login con email no existente debe fallar
	_, err = authSvc.Login(ctx, ports.LoginRequest{
		Email:    "noexiste@test.com",
		Password: "Password123",
		IPOrigen: "127.0.0.1",
	})
	if err != domain.ErrCredencialesInvalidas {
		t.Fatalf("Esperaba ErrCredencialesInvalidas por email inexistente, obtuve: %v", err)
	}

	// 6. Login exitoso
	sesion, err := authSvc.Login(ctx, ports.LoginRequest{
		Email:    "admin@test.com",
		Password: "Password123",
		IPOrigen: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Error en login exitoso: %v", err)
	}
	if len(sesion.Token) != 64 {
		t.Errorf("Token de sesión debe tener 64 caracteres hex, tiene %d", len(sesion.Token))
	}

	// 7. ValidarSesion activa
	userEnSesion, err := authSvc.ValidarSesion(ctx, sesion.Token)
	if err != nil {
		t.Fatalf("Error validando sesión activa: %v", err)
	}
	if userEnSesion.ID != usuario.ID {
		t.Errorf("Usuario de sesión no coincide: got %d, want %d", userEnSesion.ID, usuario.ID)
	}

	// 8. Logout invalida la sesión
	if err := authSvc.Logout(ctx, sesion.Token); err != nil {
		t.Fatalf("Error en Logout: %v", err)
	}

	// 9. ValidarSesion tras logout debe fallar
	_, err = authSvc.ValidarSesion(ctx, sesion.Token)
	if err != domain.ErrSesionInvalida {
		t.Fatalf("Esperaba ErrSesionInvalida tras logout, obtuve: %v", err)
	}

	// 10. Desactivar usuario y verificar que no puede iniciar sesión
	inactivo := false
	_, err = authSvc.ModificarUsuario(ctx, usuario.ID, ports.ModificarUsuarioRequest{Activo: &inactivo})
	if err != nil {
		t.Fatalf("Error desactivando usuario: %v", err)
	}

	_, err = authSvc.Login(ctx, ports.LoginRequest{
		Email:    "admin@test.com",
		Password: "Password123",
		IPOrigen: "127.0.0.1",
	})
	if err != domain.ErrCuentaInactiva {
		t.Fatalf("Esperaba ErrCuentaInactiva para usuario desactivado, obtuve: %v", err)
	}
}
