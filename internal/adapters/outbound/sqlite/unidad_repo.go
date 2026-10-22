package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// UnidadMedidaRepositorySQLite implementa ports.UnidadMedidaRepository con database/sql nativo.
type UnidadMedidaRepositorySQLite struct {
	db *DB
}

// NewUnidadMedidaRepositorySQLite construye un nuevo adaptador de persistencia para unidades de medida.
func NewUnidadMedidaRepositorySQLite(db *DB) ports.UnidadMedidaRepository {
	return &UnidadMedidaRepositorySQLite{db: db}
}

// ObtenerTodas lista todas las unidades de medida ordenadas por categoría y es_base.
func (r *UnidadMedidaRepositorySQLite) ObtenerTodas(ctx context.Context) ([]domain.UnidadMedida, error) {
	query := `SELECT id, nombre, abreviatura, categoria, es_base, factor_n, factor_d
	          FROM unidades_medida ORDER BY categoria ASC, es_base DESC, nombre ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error al listar unidades de medida: %w", err)
	}
	defer rows.Close()

	var unidades []domain.UnidadMedida
	for rows.Next() {
		var u domain.UnidadMedida
		var esBase int
		if err := rows.Scan(
			&u.ID,
			&u.Nombre,
			&u.Abreviatura,
			&u.Categoria,
			&esBase,
			&u.FactorN,
			&u.FactorD,
		); err != nil {
			return nil, fmt.Errorf("error al leer fila de unidad de medida: %w", err)
		}
		u.EsBase = esBase == 1
		unidades = append(unidades, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de unidades de medida: %w", err)
	}

	return unidades, nil
}

// ObtenerPorID obtiene una unidad de medida por su identificador primario.
func (r *UnidadMedidaRepositorySQLite) ObtenerPorID(ctx context.Context, id int64) (domain.UnidadMedida, error) {
	query := `SELECT id, nombre, abreviatura, categoria, es_base, factor_n, factor_d
	          FROM unidades_medida WHERE id = ?`

	var u domain.UnidadMedida
	var esBase int
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Nombre,
		&u.Abreviatura,
		&u.Categoria,
		&esBase,
		&u.FactorN,
		&u.FactorD,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return domain.UnidadMedida{}, domain.ErrNotFound
		}
		return domain.UnidadMedida{}, fmt.Errorf("error al consultar unidad de medida por ID: %w", err)
	}

	u.EsBase = esBase == 1
	return u, nil
}

// ObtenerPorAbreviatura busca una unidad de medida por su abreviatura única (índice único).
func (r *UnidadMedidaRepositorySQLite) ObtenerPorAbreviatura(ctx context.Context, abrev string) (domain.UnidadMedida, error) {
	query := `SELECT id, nombre, abreviatura, categoria, es_base, factor_n, factor_d
	          FROM unidades_medida WHERE abreviatura = ?`

	var u domain.UnidadMedida
	var esBase int
	err := r.db.QueryRowContext(ctx, query, abrev).Scan(
		&u.ID,
		&u.Nombre,
		&u.Abreviatura,
		&u.Categoria,
		&esBase,
		&u.FactorN,
		&u.FactorD,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return domain.UnidadMedida{}, domain.ErrNotFound
		}
		return domain.UnidadMedida{}, fmt.Errorf("error al consultar unidad de medida por abreviatura: %w", err)
	}

	u.EsBase = esBase == 1
	return u, nil
}

// ObtenerPorCategoria lista unidades de medida filtradas por categoría.
func (r *UnidadMedidaRepositorySQLite) ObtenerPorCategoria(ctx context.Context, cat domain.Categoria) ([]domain.UnidadMedida, error) {
	query := `SELECT id, nombre, abreviatura, categoria, es_base, factor_n, factor_d
	          FROM unidades_medida WHERE categoria = ? ORDER BY es_base DESC, nombre ASC`

	rows, err := r.db.QueryContext(ctx, query, string(cat))
	if err != nil {
		return nil, fmt.Errorf("error al listar unidades de medida por categoría: %w", err)
	}
	defer rows.Close()

	var unidades []domain.UnidadMedida
	for rows.Next() {
		var u domain.UnidadMedida
		var esBase int
		if err := rows.Scan(
			&u.ID,
			&u.Nombre,
			&u.Abreviatura,
			&u.Categoria,
			&esBase,
			&u.FactorN,
			&u.FactorD,
		); err != nil {
			return nil, fmt.Errorf("error al leer fila de unidad de medida: %w", err)
		}
		u.EsBase = esBase == 1
		unidades = append(unidades, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de unidades de medida: %w", err)
	}

	return unidades, nil
}

// Crear inserta una nueva unidad de medida con validación de denominador > 0.
func (r *UnidadMedidaRepositorySQLite) Crear(ctx context.Context, u domain.UnidadMedida) (int64, error) {
	if u.FactorD <= 0 {
		return 0, domain.ErrDenominadorCero
	}
	if u.FactorN <= 0 {
		return 0, fmt.Errorf("factor_n debe ser mayor a cero")
	}

	var esBase int
	if u.EsBase {
		esBase = 1
	}

	query := `INSERT INTO unidades_medida (nombre, abreviatura, categoria, es_base, factor_n, factor_d)
	          VALUES (?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query,
		u.Nombre,
		u.Abreviatura,
		string(u.Categoria),
		esBase,
		u.FactorN,
		u.FactorD,
	)
	if err != nil {
		return 0, fmt.Errorf("error al insertar unidad de medida: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error al recuperar ID de unidad de medida: %w", err)
	}

	return id, nil
}

// ObtenerUnidadesCompraInsumo lista las equivalencias de compra registradas para un insumo.
func (r *UnidadMedidaRepositorySQLite) ObtenerUnidadesCompraInsumo(ctx context.Context, insumoID int64) ([]domain.InsumoUnidadCompra, error) {
	query := `SELECT iuc.id, iuc.insumo_id, iuc.unidad_id, iuc.factor_n, iuc.factor_d, iuc.es_principal, iuc.descripcion,
	                   um.nombre, um.abreviatura, um.categoria, um.es_base, um.factor_n, um.factor_d
	          FROM insumo_unidades_compra iuc
	          JOIN unidades_medida um ON iuc.unidad_id = um.id
	          WHERE iuc.insumo_id = ?
	          ORDER BY iuc.es_principal DESC, um.nombre ASC`

	rows, err := r.db.QueryContext(ctx, query, insumoID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar unidades de compra del insumo: %w", err)
	}
	defer rows.Close()

	var unidades []domain.InsumoUnidadCompra
	for rows.Next() {
		var iuc domain.InsumoUnidadCompra
		var esPrincipal int
		var esBase int
		var um domain.UnidadMedida
		if err := rows.Scan(
			&iuc.ID,
			&iuc.InsumoID,
			&iuc.UnidadID,
			&iuc.FactorN,
			&iuc.FactorD,
			&esPrincipal,
			&iuc.Description,
			&um.Nombre,
			&um.Abreviatura,
			&um.Categoria,
			&esBase,
			&um.FactorN,
			&um.FactorD,
		); err != nil {
			return nil, fmt.Errorf("error al leer fila de unidad de compra: %w", err)
		}
		iuc.EsPrincipal = esPrincipal == 1
		um.EsBase = esBase == 1
		unidades = append(unidades, iuc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de unidades de compra: %w", err)
	}

	return unidades, nil
}

// GuardarUnidadCompraInsumo inserta o reemplaza una equivalencia de compra por insumo.
// Maneja el conflicto de es_principal único mediante el índice parcial idx_iuc_principal.
func (r *UnidadMedidaRepositorySQLite) GuardarUnidadCompraInsumo(ctx context.Context, iuc domain.InsumoUnidadCompra) error {
	if iuc.FactorD <= 0 {
		return domain.ErrDenominadorCero
	}
	if iuc.FactorN <= 0 {
		return fmt.Errorf("factor_n debe ser mayor a cero")
	}

	var esPrincipal int
	if iuc.EsPrincipal {
		esPrincipal = 1
	}

	query := `INSERT INTO insumo_unidades_compra (insumo_id, unidad_id, factor_n, factor_d, es_principal, descripcion)
	          VALUES (?, ?, ?, ?, ?, ?)
	          ON CONFLICT(insumo_id, unidad_id) DO UPDATE SET
	            factor_n = excluded.factor_n,
	            factor_d = excluded.factor_d,
	            es_principal = excluded.es_principal,
	            descripcion = excluded.descripcion`

	_, err := r.db.ExecContext(ctx, query,
		iuc.InsumoID,
		iuc.UnidadID,
		iuc.FactorN,
		iuc.FactorD,
		esPrincipal,
		iuc.Description,
	)
	if err != nil {
		return fmt.Errorf("error al guardar unidad de compra del insumo: %w", err)
	}

	return nil
}
