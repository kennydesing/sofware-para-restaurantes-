package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// AtomicDebitorSQLite implementa ports.AtomicDebitor garantizando atomicidad total en SQLite.
type AtomicDebitorSQLite struct {
	db *DB
}

// NewAtomicDebitor crea una nueva instancia del ejecutor de transacciones atómicas.
func NewAtomicDebitor(db *DB) ports.AtomicDebitor {
	return &AtomicDebitorSQLite{db: db}
}

// DebitarStockPedidoAtomic ejecuta el débito atómico de insumos dentro de una transacción SQL explícita.
//
// Protocolo de Seguridad ACID:
// 1. Inicia BEGIN TRANSACTION con aislamiento serializable.
// 2. FASE DE VERIFICACIÓN: Consulta el stock físico actual de cada insumo. Si cualquiera de ellos
//    posee stock menor al requerido por el escandallo, aborta de inmediato con Rollback total
//    y retorna domain.ErrStockInsuficiente con el detalle exacto.
// 3. FASE DE DÉBITO: Descuenta la cantidad exacta (gramos/ml/u calculados con merma) en la tabla 'insumos'.
// 4. FASE DE AUDITORÍA: Inserta un registro 'SALIDA_VENTA' en 'movimientos_inventario' por cada insumo consumido.
// 5. FASE DE ESTADO: Actualiza el pedido a 'EN_COCINA'.
// 6. FASE DE CONFIRMACIÓN: Ejecuta COMMIT. Si el sistema sufre un corte de energía en cualquier microsegundo,
//    SQLite WAL revierte todo al reiniciar, garantizando integridad física y contable del 100%.
func (a *AtomicDebitorSQLite) DebitarStockPedidoAtomic(ctx context.Context, pedidoID int, consumos []ports.ConsumoInsumoDTO) error {
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("error al iniciar transacción de débito atómico: %w", err)
	}
	defer tx.Rollback() // Rollback seguro en caso de error o pánico

	// =========================================================================
	// FASE 1: VERIFICACIÓN PREVIA DE STOCK CON BLOQUEO DENTRO DE LA TX
	// =========================================================================
	queryCheck := `SELECT nombre, stock, unidad_base FROM insumos WHERE id = ?`
	stmtCheck, err := tx.PrepareContext(ctx, queryCheck)
	if err != nil {
		return fmt.Errorf("error al preparar verificación de stock: %w", err)
	}
	defer stmtCheck.Close()

	for _, c := range consumos {
		var nombre string
		var stockActual float64
		var unidadBase string

		err := stmtCheck.QueryRowContext(ctx, c.InsumoID).Scan(&nombre, &stockActual, &unidadBase)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("%w: insumo ID %d", domain.ErrInsumoNoEncontrado, c.InsumoID)
			}
			return fmt.Errorf("error al consultar stock del insumo ID %d: %w", c.InsumoID, err)
		}

		if stockActual < c.Cantidad {
			// ROLLBACK AUTOMÁTICO: Stock insuficiente
			return fmt.Errorf("%w: insumo '%s' (ID %d) tiene %.2f %s disponibles pero se requieren %.2f %s",
				domain.ErrStockInsuficiente,
				nombre,
				c.InsumoID,
				stockActual,
				unidadBase,
				c.Cantidad,
				unidadBase,
			)
		}
	}

	// =========================================================================
	// FASE 2: DESCUENTO DE STOCK EN INSUMOS
	// =========================================================================
	queryUpdateStock := `UPDATE insumos 
	                     SET stock = stock - ?, updated_at = CURRENT_TIMESTAMP 
	                     WHERE id = ?`
	stmtUpdateStock, err := tx.PrepareContext(ctx, queryUpdateStock)
	if err != nil {
		return fmt.Errorf("error al preparar actualización de stock: %w", err)
	}
	defer stmtUpdateStock.Close()

	for _, c := range consumos {
		res, err := stmtUpdateStock.ExecContext(ctx, c.Cantidad, c.InsumoID)
		if err != nil {
			return fmt.Errorf("error al descontar stock del insumo %d: %w", c.InsumoID, err)
		}
		rows, err := res.RowsAffected()
		if err != nil || rows == 0 {
			return fmt.Errorf("no se pudo actualizar stock del insumo ID %d", c.InsumoID)
		}
	}

	// =========================================================================
	// FASE 3: AUDITORÍA EN MOVIMIENTOS_INVENTARIO (SALIDA_VENTA)
	// =========================================================================
	queryMovimiento := `INSERT INTO movimientos_inventario (insumo_id, tipo, cantidad, costo_unitario, referencia) 
	                    VALUES (?, 'SALIDA_VENTA', ?, ?, ?)`
	stmtMovimiento, err := tx.PrepareContext(ctx, queryMovimiento)
	if err != nil {
		return fmt.Errorf("error al preparar inserción de auditoría: %w", err)
	}
	defer stmtMovimiento.Close()

	for _, c := range consumos {
		_, err := stmtMovimiento.ExecContext(ctx, c.InsumoID, c.Cantidad, c.CostoUnitario, c.Referencia)
		if err != nil {
			return fmt.Errorf("error al registrar movimiento de auditoría para insumo %d: %w", c.InsumoID, err)
		}
	}

	// =========================================================================
	// FASE 4: ACTUALIZACIÓN DE ESTADO DEL PEDIDO A 'EN_COCINA'
	// =========================================================================
	queryEstadoPedido := `UPDATE pedidos 
	                      SET estado = 'EN_COCINA', updated_at = CURRENT_TIMESTAMP 
	                      WHERE id = ?`
	resPed, err := tx.ExecContext(ctx, queryEstadoPedido, pedidoID)
	if err != nil {
		return fmt.Errorf("error al actualizar estado del pedido: %w", err)
	}
	rowsPed, err := resPed.RowsAffected()
	if err != nil || rowsPed == 0 {
		return fmt.Errorf("pedido ID %d no encontrado para actualizar estado", pedidoID)
	}

	// =========================================================================
	// FASE 5: COMMIT FINAL DE LA TRANSACCIÓN
	// =========================================================================
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error al confirmar commit de la transacción: %w", err)
	}

	return nil
}
