package domain

import "time"

// Sesion representa una sesión autenticada activa en el sistema.
type Sesion struct {
	Token     string    // 64 hex chars — 32 bytes de crypto/rand
	UsuarioID int
	IPOrigen  string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// EstaVigente comprueba si la sesión no ha expirado.
func (s *Sesion) EstaVigente() bool {
	return time.Now().Before(s.ExpiresAt)
}