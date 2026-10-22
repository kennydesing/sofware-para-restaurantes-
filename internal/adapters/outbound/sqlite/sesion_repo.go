package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// SesionRepositorySQLite implementa ports.SesionRepository usando SQLite.
type SesionRepositorySQLite struct {
	db *DB
}

// NewSesionRepository crea un nuevo repositorio de sesiones SQLite.
func NewSesionRepository(db *DB) ports.SesionRepository {
	return &SesionRepositorySQLite{db: db}
}

var _ ports.SesionRepository = (*SesionRepositorySQLite)(nil)

// Create inserta una nueva sesión en SQLite.
func (r *SesionRepositorySQLite) Create(ctx context.Context, s *domain.Sesion) error {
	query := `INSERT INTO sesiones (token, usuario_id, ip_origen, user_agent, created_at, expires_at)
	          VALUES (?, ?, ?, ?, ?, ?)`

	createdAt := s.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, query,
		s.Token,
		s.UsuarioID,
		s.IPOrigen,
		s.UserAgent,
		createdAt,
		s.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("error al insertar sesión: %w", err)
	}

	return nil
}

// GetByToken consulta una sesión por su token opaco.
func (r *SesionRepositorySQLite) GetByToken(ctx context.Context, token string) (*domain.Sesion, error) {
	query := `SELECT token, usuario_id, ip_origen, user_agent, created_at, expires_at
	          FROM sesiones WHERE token = ?`

	var s domain.Sesion
	var createdAt, expiresAt time.Time

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&s.Token,
		&s.UsuarioID,
		&s.IPOrigen,
		&s.UserAgent,
		&createdAt,
		&expiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSesionInvalida
		}
		return nil, fmt.Errorf("error al consultar sesión por token: %w", err)
	}

	s.CreatedAt = createdAt
	s.ExpiresAt = expiresAt

	return &s, nil
}

// DeleteByToken invalida una sesión específica al cerrar sesión.
func (r *SesionRepositorySQLite) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM sesiones WHERE token = ?`
	_, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("error al eliminar sesión: %w", err)
	}
	return nil
}

// DeleteByUsuarioID elimina todas las sesiones de un usuario (para SINGLE_SESSION).
func (r *SesionRepositorySQLite) DeleteByUsuarioID(ctx context.Context, usuarioID int) error {
	query := `DELETE FROM sesiones WHERE usuario_id = ?`
	_, err := r.db.ExecContext(ctx, query, usuarioID)
	if err != nil {
		return fmt.Errorf("error al eliminar sesiones por usuario: %w", err)
	}
	return nil
}

// DeleteExpired elimina las sesiones vencidas y retorna la cantidad de registros borrados.
func (r *SesionRepositorySQLite) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sesiones WHERE expires_at < datetime('now')`
	res, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("error al limpiar sesiones expiradas: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error obteniendo filas afectadas al limpiar sesiones: %w", err)
	}

	return rows, nil
}
