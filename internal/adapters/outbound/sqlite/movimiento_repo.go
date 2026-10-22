package sqlite

import (
	"context"
	"fmt"
	"time"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// MovimientoRepositorySQLite implementa ports.MovimientoRepository.
type MovimientoRepositorySQLite struct {
	db *DB
}

// NewMovimientoRepository crea una nueva instancia del repositorio del libro diario de inventario.
func NewMovimientoRepository(db *DB) ports.MovimientoRepository {
	return &MovimientoRepositorySQLite{db: db}
}

// Create inserta un registro inmutable en el libro diario de auditoría.
func (r *MovimientoRepositorySQLite) Create(ctx context.Context, mov *domain.MovimientoInventario) error {
	query := `INSERT INTO movimientos_inventario (insumo_id, tipo, cantidad, costo_unitario, referencia) 
	          VALUES (?, ?, ?, ?, ?)`

	res, err := r.db.ExecContext(ctx, query,
		mov.InsumoID,
		string(mov.Tipo),
		mov.Cantidad,
		mov.CostoUnitario,
		mov.Referencia,
	)
	if err != nil {
		return fmt.Errorf("error al insertar movimiento de auditoría: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("error al obtener ID de movimiento: %w", err)
	}
	mov.ID = int(id)
	mov.CreatedAt = time.Now()

	return nil
}

// GetByInsumoID obtiene el historial de movimientos de un insumo ordenados cronológicamente.
func (r *MovimientoRepositorySQLite) GetByInsumoID(ctx context.Context, insumoID int) ([]*domain.MovimientoInventario, error) {
	query := `SELECT id, insumo_id, tipo, cantidad, costo_unitario, referencia, created_at 
	          FROM movimientos_inventario 
	          WHERE insumo_id = ? 
	          ORDER BY created_at ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, query, insumoID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar movimientos del insumo: %w", err)
	}
	defer rows.Close()

	var movimientos []*domain.MovimientoInventario
	for rows.Next() {
		var m domain.MovimientoInventario
		var tipoStr string
		var createdAtStr string

		if err := rows.Scan(
			&m.ID,
			&m.InsumoID,
			&tipoStr,
			&m.Cantidad,
			&m.CostoUnitario,
			&m.Referencia,
			&createdAtStr,
		); err != nil {
			return nil, fmt.Errorf("error al leer fila de movimiento: %w", err)
		}
		m.Tipo = domain.TipoMovimiento(tipoStr)
		if m.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr); err != nil {
			return nil, fmt.Errorf("error al parsear created_at del movimiento: %w", err)
		}
		movimientos = append(movimientos, &m)
	}

	return movimientos, nil
}

// ListarHistorialMermas devuelve movimientos SALIDA_MERMA excluyendo los de uso personal.
func (r *MovimientoRepositorySQLite) ListarHistorialMermas(ctx context.Context, limite int) ([]*ports.HistorialSalidaDTO, error) {
	if limite <= 0 {
		limite = 100
	}
	query := `SELECT m.id, m.insumo_id, i.nombre, m.cantidad, i.unidad_base,
	                 (m.cantidad * m.costo_unitario) AS costo_total, m.referencia, m.created_at
	          FROM movimientos_inventario m
	          JOIN insumos i ON i.id = m.insumo_id
	          WHERE m.tipo = 'SALIDA_MERMA' AND m.referencia NOT LIKE 'USO_PERSONAL:%'
	          ORDER BY m.created_at DESC, m.id DESC
	          LIMIT ?`

	return r.listarHistorialSalida(ctx, query, limite)
}

// ListarHistorialUsoPersonal devuelve movimientos SALIDA_MERMA marcados como uso personal.
func (r *MovimientoRepositorySQLite) ListarHistorialUsoPersonal(ctx context.Context, limite int) ([]*ports.HistorialSalidaDTO, error) {
	if limite <= 0 {
		limite = 100
	}
	query := `SELECT m.id, m.insumo_id, i.nombre, m.cantidad, i.unidad_base,
	                 (m.cantidad * m.costo_unitario) AS costo_total, m.referencia, m.created_at
	          FROM movimientos_inventario m
	          JOIN insumos i ON i.id = m.insumo_id
	          WHERE m.tipo = 'SALIDA_MERMA' AND m.referencia LIKE 'USO_PERSONAL:%'
	          ORDER BY m.created_at DESC, m.id DESC
	          LIMIT ?`

	return r.listarHistorialSalida(ctx, query, limite)
}

func (r *MovimientoRepositorySQLite) listarHistorialSalida(ctx context.Context, query string, limite int) ([]*ports.HistorialSalidaDTO, error) {
	rows, err := r.db.QueryContext(ctx, query, limite)
	if err != nil {
		return nil, fmt.Errorf("error al consultar historial de salidas: %w", err)
	}
	defer rows.Close()

	var resultados []*ports.HistorialSalidaDTO
	for rows.Next() {
		var dto ports.HistorialSalidaDTO
		var createdAtStr string
		if err := rows.Scan(
			&dto.ID,
			&dto.InsumoID,
			&dto.InsumoNombre,
			&dto.Cantidad,
			&dto.UnidadBase,
			&dto.CostoTotal,
			&dto.Referencia,
			&createdAtStr,
		); err != nil {
			return nil, fmt.Errorf("error al leer fila de historial: %w", err)
		}
		if dto.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr); err != nil {
			return nil, fmt.Errorf("error al parsear created_at del historial: %w", err)
		}
		resultados = append(resultados, &dto)
	}

	return resultados, nil
}