package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// UsuarioRepositorySQLite implementa ports.UsuarioRepository usando SQLite.
type UsuarioRepositorySQLite struct {
	db *DB
}

// NewUsuarioRepository construye un nuevo repositorio SQLite para usuarios.
func NewUsuarioRepository(db *DB) ports.UsuarioRepository {
	return &UsuarioRepositorySQLite{db: db}
}

// Verificación en tiempo de compilación
var _ ports.UsuarioRepository = (*UsuarioRepositorySQLite)(nil)

// GetByID obtiene un usuario por su identificador primario.
func (r *UsuarioRepositorySQLite) GetByID(ctx context.Context, id int) (*domain.Usuario, error) {
	query := `SELECT id, nombre, email, password_hash, rol, activo, created_at, updated_at
	          FROM usuarios WHERE id = ?`

	var u domain.Usuario
	var activoInt int
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Nombre,
		&u.Email,
		&u.PasswordHash,
		&u.Rol,
		&activoInt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al consultar usuario por ID: %w", err)
	}

	u.Activo = activoInt == 1
	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt

	return &u, nil
}

// GetByEmail busca un usuario por su email único.
func (r *UsuarioRepositorySQLite) GetByEmail(ctx context.Context, email string) (*domain.Usuario, error) {
	query := `SELECT id, nombre, email, password_hash, rol, activo, created_at, updated_at
	          FROM usuarios WHERE email = ?`

	var u domain.Usuario
	var activoInt int
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID,
		&u.Nombre,
		&u.Email,
		&u.PasswordHash,
		&u.Rol,
		&activoInt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al consultar usuario por email: %w", err)
	}

	u.Activo = activoInt == 1
	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt

	return &u, nil
}

// GetAll obtiene todos los usuarios ordenados por fecha de creación descendente.
// NUNCA recupera ni expone el hash de contraseñas.
func (r *UsuarioRepositorySQLite) GetAll(ctx context.Context) ([]*domain.Usuario, error) {
	query := `SELECT id, nombre, email, rol, activo, created_at, updated_at
	          FROM usuarios ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error al listar usuarios: %w", err)
	}
	defer rows.Close()

	var usuarios []*domain.Usuario
	for rows.Next() {
		var u domain.Usuario
		var activoInt int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(
			&u.ID,
			&u.Nombre,
			&u.Email,
			&u.Rol,
			&activoInt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("error al escanear usuario: %w", err)
		}

		u.Activo = activoInt == 1
		u.CreatedAt = createdAt
		u.UpdatedAt = updatedAt
		u.PasswordHash = "" // Proteger contra cualquier filtración

		usuarios = append(usuarios, &u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando usuarios: %w", err)
	}

	return usuarios, nil
}

// Create inserta un nuevo usuario en la base de datos.
func (r *UsuarioRepositorySQLite) Create(ctx context.Context, u *domain.Usuario) error {
	query := `INSERT INTO usuarios (nombre, email, password_hash, rol, activo, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))`

	activoInt := 0
	if u.Activo {
		activoInt = 1
	}

	res, err := r.db.ExecContext(ctx, query, u.Nombre, u.Email, u.PasswordHash, string(u.Rol), activoInt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "PRIMARY KEY") {
			return domain.ErrEmailDuplicado
		}
		return fmt.Errorf("error al crear usuario: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		u.ID = int(id)
	}

	return nil
}

// Update modifica únicamente rol y activo de un usuario existente.
// NUNCA modifica password_hash.
func (r *UsuarioRepositorySQLite) Update(ctx context.Context, u *domain.Usuario) error {
	query := `UPDATE usuarios 
	          SET rol = ?, activo = ?, updated_at = datetime('now')
	          WHERE id = ?`

	activoInt := 0
	if u.Activo {
		activoInt = 1
	}

	res, err := r.db.ExecContext(ctx, query, string(u.Rol), activoInt, u.ID)
	if err != nil {
		return fmt.Errorf("error al actualizar usuario: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error comprobando filas actualizadas: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}
