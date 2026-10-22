package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

type TasaCambioRepositorySQLite struct {
	db *DB
}

func NewTasaCambioRepository(db *DB) ports.TasaCambioRepository {
	return &TasaCambioRepositorySQLite{db: db}
}

var _ ports.TasaCambioRepository = (*TasaCambioRepositorySQLite)(nil)

func (r *TasaCambioRepositorySQLite) GetTasa(ctx context.Context, moneda string) (*domain.TasaCambio, error) {
	query := `SELECT moneda, tasa, fuente, fecha_actualizacion FROM tasas_cambio WHERE moneda = ?`
	var t domain.TasaCambio
	var fechaActualizacion time.Time

	err := r.db.QueryRowContext(ctx, query, moneda).Scan(
		&t.Moneda,
		&t.Tasa,
		&t.Fuente,
		&fechaActualizacion,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrTasaNoEncontrada
		}
		return nil, fmt.Errorf("error al consultar tasa de cambio para %s: %w", moneda, err)
	}
	t.FechaActualizacion = fechaActualizacion
	return &t, nil
}

func (r *TasaCambioRepositorySQLite) GetAllTasas(ctx context.Context) ([]*domain.TasaCambio, error) {
	query := `SELECT moneda, tasa, fuente, fecha_actualizacion FROM tasas_cambio ORDER BY moneda DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error al listar tasas de cambio: %w", err)
	}
	defer rows.Close()

	var tasas []*domain.TasaCambio
	for rows.Next() {
		var t domain.TasaCambio
		var fechaActualizacion time.Time
		if err := rows.Scan(&t.Moneda, &t.Tasa, &t.Fuente, &fechaActualizacion); err != nil {
			return nil, fmt.Errorf("error al leer fila de tasa de cambio: %w", err)
		}
		t.FechaActualizacion = fechaActualizacion
		tasas = append(tasas, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de tasas: %w", err)
	}
	return tasas, nil
}

func (r *TasaCambioRepositorySQLite) GuardarTasa(ctx context.Context, tasa *domain.TasaCambio) error {
	query := `INSERT INTO tasas_cambio (moneda, tasa, fuente, fecha_actualizacion)
	          VALUES (?, ?, ?, ?)
	          ON CONFLICT(moneda) DO UPDATE SET
	              tasa = excluded.tasa,
	              fuente = excluded.fuente,
	              fecha_actualizacion = excluded.fecha_actualizacion`

	now := time.Now()
	if !tasa.FechaActualizacion.IsZero() {
		now = tasa.FechaActualizacion
	}

	_, err := r.db.ExecContext(ctx, query, tasa.Moneda, tasa.Tasa, tasa.Fuente, now)
	if err != nil {
		return fmt.Errorf("error al guardar tasa de cambio: %w", err)
	}
	tasa.FechaActualizacion = now
	return nil
}
