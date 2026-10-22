package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

const (
	// Parámetros Argon2id recomendados (OWASP 2023)
	argon2Memory      = 64 * 1024 // 64 MB
	argon2Iterations  = 3
	argon2Parallelism = 2
	argon2KeyLen      = 32 // 256 bits
	argon2SaltLen     = 16
)

type authService struct {
	usuarioRepo   ports.UsuarioRepository
	sesionRepo    ports.SesionRepository
	auditoriaRepo ports.AuditoriaRepository
}

func NewAuthService(
	usuarioRepo ports.UsuarioRepository,
	sesionRepo ports.SesionRepository,
	auditoriaRepo ports.AuditoriaRepository,
) ports.AuthService {
	return &authService{
		usuarioRepo:   usuarioRepo,
		sesionRepo:    sesionRepo,
		auditoriaRepo: auditoriaRepo,
	}
}

// Verificación en tiempo de compilación
var _ ports.AuthService = (*authService)(nil)

func (s *authService) Login(ctx context.Context, req ports.LoginRequest) (*domain.Sesion, error) {
	usuario, err := s.usuarioRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.registrarLoginFallido(ctx, req.Email, req.IPOrigen)
			time.Sleep(300 * time.Millisecond) // Mitigación timing attack
			return nil, domain.ErrCredencialesInvalidas
		}
		return nil, err
	}

	if !usuario.EsActivo() {
		s.registrarLoginFallido(ctx, req.Email, req.IPOrigen)
		return nil, domain.ErrCuentaInactiva
	}

	if err := verifyArgon2id(usuario.PasswordHash, req.Password); err != nil {
		s.registrarLoginFallido(ctx, req.Email, req.IPOrigen)
		time.Sleep(300 * time.Millisecond)
		return nil, domain.ErrCredencialesInvalidas
	}

	// Generar token de sesión
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("error generando token de sesión: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// TTL de sesión desde variable de entorno
	ttlHours := getEnvInt("SESSION_TTL_HOURS", 8)
	expiresAt := time.Now().Add(time.Duration(ttlHours) * time.Hour)

	sesion := &domain.Sesion{
		Token:     token,
		UsuarioID: usuario.ID,
		IPOrigen:  req.IPOrigen,
		UserAgent: "",
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	if err := s.sesionRepo.Create(ctx, sesion); err != nil {
		return nil, fmt.Errorf("error creando sesión: %w", err)
	}

	// SINGLE_SESSION: invalidar sesiones anteriores si está habilitado
	if getEnvBool("SINGLE_SESSION", false) {
		if err := s.sesionRepo.DeleteByUsuarioID(ctx, usuario.ID); err != nil {
			log.Printf("⚠️ Error limpiando sesiones previas: %v", err)
		}
	}

	// Auditoría: LOGIN exitoso
	if err := s.auditoriaRepo.Create(ctx, &domain.EventoAuditoria{
		UsuarioID:   &usuario.ID,
		Accion:      domain.AccionLogin,
		EntidadTipo: "usuario",
		EntidadID:   &usuario.ID,
		Detalle:     "Inicio de sesión exitoso",
		IPOrigen:    req.IPOrigen,
		CreatedAt:   time.Now(),
	}); err != nil {
		log.Printf("⚠️ Error registrando auditoría de login: %v", err)
	}

	return sesion, nil
}

func (s *authService) Logout(ctx context.Context, token string) error {
	sesion, err := s.sesionRepo.GetByToken(ctx, token)
	if err != nil {
		return err
	}

	if err := s.sesionRepo.DeleteByToken(ctx, token); err != nil {
		return err
	}

	// Auditoría: LOGOUT
	if err := s.auditoriaRepo.Create(ctx, &domain.EventoAuditoria{
		UsuarioID:   &sesion.UsuarioID,
		Accion:      domain.AccionLogout,
		EntidadTipo: "usuario",
		EntidadID:   &sesion.UsuarioID,
		Detalle:     "Cierre de sesión",
		IPOrigen:    sesion.IPOrigen,
		CreatedAt:   time.Now(),
	}); err != nil {
		log.Printf("⚠️ Error registrando auditoría de logout: %v", err)
	}

	return nil
}

func (s *authService) ValidarSesion(ctx context.Context, token string) (*domain.Usuario, error) {
	sesion, err := s.sesionRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, domain.ErrSesionInvalida
	}

	if !sesion.EstaVigente() {
		_ = s.sesionRepo.DeleteByToken(ctx, token)
		return nil, domain.ErrSesionExpirada
	}

	usuario, err := s.usuarioRepo.GetByID(ctx, sesion.UsuarioID)
	if err != nil {
		return nil, domain.ErrSesionInvalida
	}

	if !usuario.EsActivo() {
		return nil, domain.ErrCuentaInactiva
	}

	return usuario, nil
}

func (s *authService) CrearUsuario(ctx context.Context, req ports.CrearUsuarioRequest) (*domain.Usuario, error) {
	if !domain.RolesValidos[req.Rol] {
		return nil, domain.ErrRolInvalido
	}

	if err := validarPassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := hashArgon2id(req.Password)
	if err != nil {
		return nil, fmt.Errorf("error hasheando contraseña: %w", err)
	}

	usuario := &domain.Usuario{
		Nombre:       req.Nombre,
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: hash,
		Rol:          req.Rol,
		Activo:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.usuarioRepo.Create(ctx, usuario); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			return nil, domain.ErrEmailDuplicado
		}
		return nil, err
	}

	// Auditoría: CREAR_USUARIO
	if err := s.auditoriaRepo.Create(ctx, &domain.EventoAuditoria{
		UsuarioID:   &usuario.ID,
		Accion:      domain.AccionCrearUsuario,
		EntidadTipo: "usuario",
		EntidadID:   &usuario.ID,
		Detalle:     fmt.Sprintf("Usuario creado con rol %s", req.Rol),
		IPOrigen:    "",
		CreatedAt:   time.Now(),
	}); err != nil {
		log.Printf("⚠️ Error registrando auditoría de creación de usuario: %v", err)
	}

	// Retornar copia con hash limpio
	res := *usuario
	res.PasswordHash = ""
	return &res, nil
}

func (s *authService) ModificarUsuario(ctx context.Context, usuarioID int, req ports.ModificarUsuarioRequest) (*domain.Usuario, error) {
	usuario, err := s.usuarioRepo.GetByID(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	var accion domain.AccionAuditoria
	detalle := ""

	if req.Rol != nil {
		if !domain.RolesValidos[*req.Rol] {
			return nil, domain.ErrRolInvalido
		}
		usuario.Rol = *req.Rol
		accion = domain.AccionModificarRol
		detalle = fmt.Sprintf("Rol cambiado a %s", *req.Rol)
	}

	if req.Activo != nil {
		usuario.Activo = *req.Activo
		if accion == domain.AccionModificarRol {
			accion = domain.AccionDesactivarUsuario
			detalle = fmt.Sprintf("Estado activo=%v, rol=%s", *req.Activo, usuario.Rol)
		} else {
			accion = domain.AccionDesactivarUsuario
			detalle = fmt.Sprintf("Estado activo=%v", *req.Activo)
		}
	}

	usuario.UpdatedAt = time.Now()

	if err := s.usuarioRepo.Update(ctx, usuario); err != nil {
		return nil, err
	}

	// Auditoría
	if err := s.auditoriaRepo.Create(ctx, &domain.EventoAuditoria{
		UsuarioID:   &usuario.ID,
		Accion:      accion,
		EntidadTipo: "usuario",
		EntidadID:   &usuario.ID,
		Detalle:     detalle,
		IPOrigen:    "",
		CreatedAt:   time.Now(),
	}); err != nil {
		log.Printf("⚠️ Error registrando auditoría de modificación: %v", err)
	}

	res := *usuario
	res.PasswordHash = ""
	return &res, nil
}

func (s *authService) ListarUsuarios(ctx context.Context) ([]*domain.Usuario, error) {
	usuarios, err := s.usuarioRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	// Asegurar que no se expone el hash
	for _, u := range usuarios {
		u.PasswordHash = ""
	}
	return usuarios, nil
}

func (s *authService) LimpiarSesionesExpiradas(ctx context.Context) (int64, error) {
	eliminados, err := s.sesionRepo.DeleteExpired(ctx)
	if err != nil {
		return 0, err
	}
	if eliminados > 0 {
		log.Printf("🧹 Limpieza de sesiones: %d tokens expirados eliminados", eliminados)
	}
	return eliminados, nil
}

func (s *authService) registrarLoginFallido(ctx context.Context, email, ip string) {
	if err := s.auditoriaRepo.Create(ctx, &domain.EventoAuditoria{
		UsuarioID:   nil,
		Accion:      domain.AccionLoginFallido,
		EntidadTipo: "usuario",
		EntidadID:   nil,
		Detalle:     fmt.Sprintf("Intento fallido para %s", email),
		IPOrigen:    ip,
		CreatedAt:   time.Now(),
	}); err != nil {
		log.Printf("⚠️ Error registrando login fallido: %v", err)
	}
}

func hashArgon2id(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, uint8(argon2Parallelism), argon2KeyLen)

	// Formato PHC: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	b64Salt := hex.EncodeToString(salt)
	b64Hash := hex.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argon2Memory, argon2Iterations, argon2Parallelism, b64Salt, b64Hash), nil
}

func verifyArgon2id(phcHash, password string) error {
	// Parse PHC string: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	parts := strings.Split(phcHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return errors.New("formato de hash inválido")
	}

	// Extraer parámetros: m=65536,t=3,p=2
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return errors.New("parámetros de hash inválidos")
	}

	memory := parseParam(params[0], "m=")
	iterations := parseParam(params[1], "t=")
	parallelism := parseParam(params[2], "p=")

	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return err
	}
	expectedHash, err := hex.DecodeString(parts[5])
	if err != nil {
		return err
	}

	computedHash := argon2.IDKey([]byte(password), salt, uint32(iterations), uint32(memory), uint8(parallelism), uint32(len(expectedHash)))

	// Comparación en tiempo constante
	if len(computedHash) != len(expectedHash) {
		return errors.New("hash no coincide")
	}
	var diff byte
	for i := range computedHash {
		diff |= computedHash[i] ^ expectedHash[i]
	}
	if diff != 0 {
		return errors.New("hash no coincide")
	}

	return nil
}

func parseParam(param, prefix string) int {
	val := strings.TrimPrefix(param, prefix)
	n, _ := strconv.Atoi(val)
	return n
}

func validarPassword(password string) error {
	if len(password) < 8 {
		return domain.ErrPasswordInsegura
	}
	hasUpper := false
	hasNumber := false
	for _, c := range password {
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		}
		if c >= '0' && c <= '9' {
			hasNumber = true
		}
	}
	if !hasUpper || !hasNumber {
		return domain.ErrPasswordInsegura
	}
	return nil
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1"
	}
	return defaultVal
}