package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// InsumoRepositorySQLite implementa ports.InsumoRepository con database/sql nativo.
type InsumoRepositorySQLite struct {
	db *DB
}

// NewInsumoRepository construye un nuevo adaptador de persistencia para insumos.
func NewInsumoRepository(db *DB) ports.InsumoRepository {
	return &InsumoRepositorySQLite{db: db}
}

// GetByID obtiene un insumo por su identificador primario.
func (r *InsumoRepositorySQLite) GetByID(ctx context.Context, id int) (*domain.Insumo, error) {
	query := `SELECT id, nombre, unidad_base, costo_promedio, stock_minimo, stock, precio_venta_usd,
	                   presentacion_nombre, cantidad_base_por_presentacion,
	                   empaque_tipo, empaque_nombre, unidades_por_empaque, contenido_ml_por_unidad, es_retornable
	          FROM insumos WHERE id = ?`

	var i domain.Insumo
	var esRetornable int
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&i.ID,
		&i.Nombre,
		&i.UnidadBase,
		&i.CostoPromedio,
		&i.StockMinimo,
		&i.Stock,
		&i.PrecioVentaUSD,
		&i.PresentacionNombre,
		&i.CantidadBasePorPresentacion,
		&i.EmpaqueTipo,
		&i.EmpaqueNombre,
		&i.UnidadesPorEmpaque,
		&i.ContenidoMlPorUnidad,
		&esRetornable,
	)
	i.NormalizarPresentacion()
	i.EsRetornable = esRetornable == 1

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInsumoNoEncontrado
		}
		return nil, fmt.Errorf("error al consultar insumo por ID: %w", err)
	}

	return &i, nil
}

// GetByNombre busca un insumo por su nombre único.
func (r *InsumoRepositorySQLite) GetByNombre(ctx context.Context, nombre string) (*domain.Insumo, error) {
	query := `SELECT id, nombre, unidad_base, costo_promedio, stock_minimo, stock, precio_venta_usd,
	                   presentacion_nombre, cantidad_base_por_presentacion,
	                   empaque_tipo, empaque_nombre, unidades_por_empaque, contenido_ml_por_unidad, es_retornable
	          FROM insumos WHERE nombre = ?`

	var i domain.Insumo
	var esRetornable int
	err := r.db.QueryRowContext(ctx, query, nombre).Scan(
		&i.ID,
		&i.Nombre,
		&i.UnidadBase,
		&i.CostoPromedio,
		&i.StockMinimo,
		&i.Stock,
		&i.PrecioVentaUSD,
		&i.PresentacionNombre,
		&i.CantidadBasePorPresentacion,
		&i.EmpaqueTipo,
		&i.EmpaqueNombre,
		&i.UnidadesPorEmpaque,
		&i.ContenidoMlPorUnidad,
		&esRetornable,
	)
	i.NormalizarPresentacion()
	i.EsRetornable = esRetornable == 1

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInsumoNoEncontrado
		}
		return nil, fmt.Errorf("error al consultar insumo por nombre: %w", err)
	}

	return &i, nil
}

// GetAll lista todos los insumos almacenados.
func (r *InsumoRepositorySQLite) GetAll(ctx context.Context) ([]*domain.Insumo, error) {
	query := `SELECT id, nombre, unidad_base, costo_promedio, stock_minimo, stock, precio_venta_usd,
	                   presentacion_nombre, cantidad_base_por_presentacion,
	                   empaque_tipo, empaque_nombre, unidades_por_empaque, contenido_ml_por_unidad, es_retornable
	          FROM insumos ORDER BY nombre ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error al listar insumos: %w", err)
	}
	defer rows.Close()

	var insumos []*domain.Insumo
	for rows.Next() {
		var i domain.Insumo
		var esRetornable int
		if err := rows.Scan(
			&i.ID,
			&i.Nombre,
			&i.UnidadBase,
			&i.CostoPromedio,
			&i.StockMinimo,
			&i.Stock,
			&i.PrecioVentaUSD,
			&i.PresentacionNombre,
			&i.CantidadBasePorPresentacion,
			&i.EmpaqueTipo,
			&i.EmpaqueNombre,
			&i.UnidadesPorEmpaque,
			&i.ContenidoMlPorUnidad,
			&esRetornable,
		); err != nil {
			return nil, fmt.Errorf("error al leer fila de insumo: %w", err)
		}
		i.NormalizarPresentacion()
		i.EsRetornable = esRetornable == 1
		insumos = append(insumos, &i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de insumos: %w", err)
	}

	return insumos, nil
}

// Create inserta un nuevo insumo en la base de datos.
func (r *InsumoRepositorySQLite) Create(ctx context.Context, insumo *domain.Insumo) error {
	insumo.NormalizarEmpaque()
	insumo.NormalizarPresentacion()
	query := `INSERT INTO insumos (nombre, unidad_base, costo_promedio, stock_minimo, stock, precio_venta_usd,
	          presentacion_nombre, cantidad_base_por_presentacion,
	          empaque_tipo, empaque_nombre, unidades_por_empaque, contenido_ml_por_unidad, es_retornable)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var esRetornable int
	if insumo.EsRetornable {
		esRetornable = 1
	}

	result, err := r.db.ExecContext(ctx, query,
		insumo.Nombre,
		insumo.UnidadBase,
		insumo.CostoPromedio,
		insumo.StockMinimo,
		insumo.Stock,
		insumo.PrecioVentaUSD,
		insumo.PresentacionNombre,
		insumo.CantidadBasePorPresentacion,
		string(insumo.EmpaqueTipo),
		insumo.EmpaqueNombre,
		insumo.UnidadesPorEmpaque,
		insumo.ContenidoMlPorUnidad,
		esRetornable,
	)
	if err != nil {
		return fmt.Errorf("error al insertar insumo: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error al recuperar ID de insumo: %w", err)
	}
	insumo.ID = int(id)

	return nil
}

// Update actualiza los datos generales de un insumo.
func (r *InsumoRepositorySQLite) Update(ctx context.Context, insumo *domain.Insumo) error {
	insumo.NormalizarEmpaque()
	insumo.NormalizarPresentacion()
	log.Printf("Updating insumo %v: nombre=%s, unidad_base=%s, costo_promedio=%v, stock_minimo=%v, stock=%v, precio_venta_usd=%v, presentacion=%s, base_x_presentacion=%v, empaque_tipo=%s", insumo.ID, insumo.Nombre, insumo.UnidadBase, insumo.CostoPromedio, insumo.StockMinimo, insumo.Stock, insumo.PrecioVentaUSD, insumo.PresentacionNombre, insumo.CantidadBasePorPresentacion, insumo.EmpaqueTipo)
	query := `UPDATE insumos 
	          SET nombre = ?, unidad_base = ?, costo_promedio = ?, stock_minimo = ?, stock = ?, precio_venta_usd = ?,
	              presentacion_nombre = ?, cantidad_base_por_presentacion = ?,
	              empaque_tipo = ?, empaque_nombre = ?, unidades_por_empaque = ?, contenido_ml_por_unidad = ?, es_retornable = ?,
	              updated_at = CURRENT_TIMESTAMP 
	          WHERE id = ?`

	var esRetornable int
	if insumo.EsRetornable {
		esRetornable = 1
	}

	res, err := r.db.ExecContext(ctx, query,
		insumo.Nombre,
		insumo.UnidadBase,
		insumo.CostoPromedio,
		insumo.StockMinimo,
		insumo.Stock,
		insumo.PrecioVentaUSD,
		insumo.PresentacionNombre,
		insumo.CantidadBasePorPresentacion,
		string(insumo.EmpaqueTipo),
		insumo.EmpaqueNombre,
		insumo.UnidadesPorEmpaque,
		insumo.ContenidoMlPorUnidad,
		esRetornable,
		insumo.ID,
	)
	if err != nil {
		return fmt.Errorf("error al actualizar insumo: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrInsumoNoEncontrado
	}

	return nil
}

// Delete elimina un insumo de la base de datos junto con sus dependencias
// referenciales (recetas, movimientos y unidades de compra) de forma atómica.
// Mantiene la integridad referencial sin requerir ON DELETE CASCADE en el esquema.
func (r *InsumoRepositorySQLite) Delete(ctx context.Context, id int) error {
	log.Printf("Deleting insumo with ID: %d", id)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción de eliminación: %w", err)
	}
	defer tx.Rollback()

	// Limpiar dependencias con FK en RESTRICT para evitar errores de integridad referencial.
	if _, err := tx.ExecContext(ctx, `DELETE FROM recetas_escandallo WHERE insumo_id = ?`, id); err != nil {
		return fmt.Errorf("error al eliminar recetas asociadas: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM movimientos_inventario WHERE insumo_id = ?`, id); err != nil {
		return fmt.Errorf("error al eliminar movimientos de inventario: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM insumo_unidades_compra WHERE insumo_id = ?`, id); err != nil {
		return fmt.Errorf("error al eliminar unidades de compra: %w", err)
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM insumos WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("error al eliminar insumo: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrInsumoNoEncontrado
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error al confirmar eliminación: %w", err)
	}

	log.Printf("Deleted insumo with ID: %d and dependencies, rows affected: %d", id, rows)
	return nil
}

// GetAllEnriquecido retorna todos los insumos con precio de venta y merma acumulada.
// Usa LEFT JOIN con movimientos_inventario para calcular la suma de mermas por insumo.
func (r *InsumoRepositorySQLite) GetAllEnriquecido(ctx context.Context) ([]*ports.InsumoEnriquecido, error) {
	query := `
		SELECT
		    i.id,
		    i.nombre,
		    i.unidad_base,
		    i.costo_promedio,
		    COALESCE(i.precio_venta_usd, 0.0)                                        AS precio_venta_usd,
		    i.stock_minimo,
		    i.stock,
		    COALESCE(SUM(CASE WHEN m.tipo = 'SALIDA_MERMA'
		                      THEN m.cantidad ELSE 0 END), 0.0)                        AS merma_acumulada,
		    i.presentacion_nombre,
		    i.cantidad_base_por_presentacion,
		    i.empaque_tipo,
		    i.empaque_nombre,
		    i.unidades_por_empaque,
		    i.contenido_ml_por_unidad,
		    i.es_retornable,
		    um.abreviatura                                                            AS unidad_compra_abrev,
		    COALESCE(i.stock_minimo / (iuc.factor_n * 1.0 / iuc.factor_d), i.stock_minimo) AS stock_minimo_en_compra
		FROM insumos i
		LEFT JOIN movimientos_inventario m ON m.insumo_id = i.id
		LEFT JOIN insumo_unidades_compra iuc ON iuc.insumo_id = i.id AND iuc.es_principal = 1
		LEFT JOIN unidades_medida um ON um.id = iuc.unidad_id
		GROUP BY i.id, i.presentacion_nombre, i.cantidad_base_por_presentacion, i.empaque_tipo, i.empaque_nombre, i.unidades_por_empaque, i.contenido_ml_por_unidad, i.es_retornable,
		         um.abreviatura, iuc.factor_n, iuc.factor_d
		ORDER BY i.nombre ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error al listar insumos enriquecidos: %w", err)
	}
	defer rows.Close()

	var insumos []*ports.InsumoEnriquecido
	for rows.Next() {
		var ie ports.InsumoEnriquecido
		var esRetornable int
		var unidadCompraAbrev sql.NullString
		if err := rows.Scan(
			&ie.ID,
			&ie.Nombre,
			&ie.UnidadBase,
			&ie.CostoPromedio,
			&ie.PrecioVentaUSD,
			&ie.StockMinimo,
			&ie.Stock,
			&ie.MermaAcumulada,
			&ie.PresentacionNombre,
			&ie.CantidadBasePorPresentacion,
			&ie.EmpaqueTipo,
			&ie.EmpaqueNombre,
			&ie.UnidadesPorEmpaque,
			&ie.ContenidoMlPorUnidad,
			&esRetornable,
			&unidadCompraAbrev,
			&ie.StockMinimoEnCompra,
		); err != nil {
			return nil, fmt.Errorf("error al leer fila de insumo enriquecido: %w", err)
		}
		ie.NormalizarPresentacion()
		ie.EsRetornable = esRetornable == 1
		ie.UnidadCompraAbrev = unidadCompraAbrev.String
		insumos = append(insumos, &ie)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de insumos enriquecidos: %w", err)
	}

	return insumos, nil
}

// UpdateStockAndCosto actualiza atómicamente el stock y el costo promedio de un insumo.
func (r *InsumoRepositorySQLite) UpdateStockAndCosto(ctx context.Context, id int, nuevoStock float64, nuevoCostoPromedio float64) error {
	query := `UPDATE insumos 
	          SET stock = ?, costo_promedio = ?, updated_at = CURRENT_TIMESTAMP 
	          WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query, nuevoStock, nuevoCostoPromedio, id)
	if err != nil {
		return fmt.Errorf("error al actualizar stock y costo: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrInsumoNoEncontrado
	}

	return nil
}
