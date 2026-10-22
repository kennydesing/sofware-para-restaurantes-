package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// RecetaRepositorySQLite implementa ports.RecetaRepository.
type RecetaRepositorySQLite struct {
	db *DB
}

// NewRecetaRepository crea una nueva instancia del repositorio de recetas/escandallos.
func NewRecetaRepository(db *DB) ports.RecetaRepository {
	return &RecetaRepositorySQLite{db: db}
}

// GetByPlatoID obtiene la ficha técnica de un plato uniendo datos del insumo (nombre y unidad base).
func (r *RecetaRepositorySQLite) GetByPlatoID(ctx context.Context, platoID int) ([]domain.RecetaItem, error) {
	query := `SELECT r.id, r.plato_id, r.insumo_id, r.cantidad_requerida, r.porcentaje_merma, r.unidad_receta_id,
	                 i.nombre, i.unidad_base, um.abreviatura
	          FROM recetas_escandallo r
	          INNER JOIN insumos i ON r.insumo_id = i.id
	          LEFT JOIN unidades_medida um ON r.unidad_receta_id = um.id
	          WHERE r.plato_id = ?`

	rows, err := r.db.QueryContext(ctx, query, platoID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar recetas del plato %d: %w", platoID, err)
	}
	defer rows.Close()

	var items []domain.RecetaItem
	for rows.Next() {
		var item domain.RecetaItem
		var unidadRecetaAbrev sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.PlatoID,
			&item.InsumoID,
			&item.CantidadRequerida,
			&item.PorcentajeMerma,
			&item.UnidadRecetaID,
			&item.NombreInsumo,
			&item.UnidadBase,
			&unidadRecetaAbrev,
		); err != nil {
			return nil, fmt.Errorf("error al leer ingrediente de receta: %w", err)
		}
		if unidadRecetaAbrev.Valid {
			item.UnidadRecetaAbrev = unidadRecetaAbrev.String
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de receta: %w", err)
	}

	return items, nil
}

// Create inserta un ingrediente con su merma en la ficha técnica de un plato.
func (r *RecetaRepositorySQLite) Create(ctx context.Context, item *domain.RecetaItem) error {
	query := `INSERT INTO recetas_escandallo (plato_id, insumo_id, cantidad_requerida, porcentaje_merma, unidad_receta_id) 
	          VALUES (?, ?, ?, ?, ?)`

	res, err := r.db.ExecContext(ctx, query, item.PlatoID, item.InsumoID, item.CantidadRequerida, item.PorcentajeMerma, item.UnidadRecetaID)
	if err != nil {
		return fmt.Errorf("error al registrar ingrediente en receta: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("error al obtener ID de receta: %w", err)
	}
	item.ID = int(id)

	return nil
}