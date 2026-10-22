package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// PlatoRepositorySQLite implementa ports.PlatoRepository.
type PlatoRepositorySQLite struct {
	db *DB
}

// NewPlatoRepository crea una nueva instancia del repositorio de platos.
func NewPlatoRepository(db *DB) ports.PlatoRepository {
	return &PlatoRepositorySQLite{db: db}
}

// GetByID obtiene un plato por su ID.
func (r *PlatoRepositorySQLite) GetByID(ctx context.Context, id int) (*domain.Plato, error) {
	query := `SELECT id, nombre, precio_venta, activo FROM platos WHERE id = ?`

	var p domain.Plato
	var activoInt int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.Nombre, &p.PrecioVenta, &activoInt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrPlatoNoEncontrado
		}
		return nil, fmt.Errorf("error al consultar plato por ID: %w", err)
	}
	p.Activo = activoInt == 1

	return &p, nil
}

// GetAll obtiene todos los platos del menú.
func (r *PlatoRepositorySQLite) GetAll(ctx context.Context) ([]*domain.Plato, error) {
	query := `SELECT id, nombre, precio_venta, activo FROM platos ORDER BY nombre ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error al listar platos: %w", err)
	}
	defer rows.Close()

	var platos []*domain.Plato
	for rows.Next() {
		var p domain.Plato
		var activoInt int
		if err := rows.Scan(&p.ID, &p.Nombre, &p.PrecioVenta, &activoInt); err != nil {
			return nil, fmt.Errorf("error al leer plato: %w", err)
		}
		p.Activo = activoInt == 1
		platos = append(platos, &p)
	}

	return platos, nil
}

// Create inserta un nuevo plato en el catálogo.
func (r *PlatoRepositorySQLite) Create(ctx context.Context, plato *domain.Plato) error {
	query := `INSERT INTO platos (nombre, precio_venta, activo) VALUES (?, ?, ?)`

	activoInt := 0
	if plato.Activo {
		activoInt = 1
	}

	res, err := r.db.ExecContext(ctx, query, plato.Nombre, plato.PrecioVenta, activoInt)
	if err != nil {
		return fmt.Errorf("error al insertar plato: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("error al obtener ID de plato: %w", err)
	}
	plato.ID = int(id)

	return nil
}
