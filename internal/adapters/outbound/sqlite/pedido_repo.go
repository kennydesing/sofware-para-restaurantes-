package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// PedidoRepositorySQLite implementa ports.PedidoRepository.
type PedidoRepositorySQLite struct {
	db *DB
}

// NewPedidoRepository crea una nueva instancia del repositorio de pedidos.
func NewPedidoRepository(db *DB) ports.PedidoRepository {
	return &PedidoRepositorySQLite{db: db}
}

// Create inserta el pedido y sus líneas de detalle en una transacción SQLite.
func (r *PedidoRepositorySQLite) Create(ctx context.Context, pedido *domain.Pedido) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción para pedido: %w", err)
	}
	defer tx.Rollback()

	// 1. Insertar cabecera del pedido
	queryPedido := `INSERT INTO pedidos (mesa_id, estado, total) VALUES (?, ?, ?)`
	res, err := tx.ExecContext(ctx, queryPedido, pedido.MesaID, string(pedido.Estado), pedido.Total)
	if err != nil {
		return fmt.Errorf("error al insertar cabecera de pedido: %w", err)
	}

	pedidoID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("error al obtener ID de pedido: %w", err)
	}
	pedido.ID = int(pedidoID)
	pedido.CreatedAt = time.Now()
	pedido.UpdatedAt = time.Now()

	// 2. Insertar cada línea de detalle
	queryDetalle := `INSERT INTO detalles_pedido (pedido_id, plato_id, cantidad, precio_unitario, subtotal) 
	                 VALUES (?, ?, ?, ?, ?)`
	stmtDetalle, err := tx.PrepareContext(ctx, queryDetalle)
	if err != nil {
		return fmt.Errorf("error al preparar inserción de detalles: %w", err)
	}
	defer stmtDetalle.Close()

	for i := range pedido.Detalles {
		d := &pedido.Detalles[i]
		d.PedidoID = pedido.ID
		resDet, err := stmtDetalle.ExecContext(ctx, d.PedidoID, d.PlatoID, d.Cantidad, d.PrecioUnitario, d.Subtotal)
		if err != nil {
			return fmt.Errorf("error al insertar detalle de pedido (plato %d): %w", d.PlatoID, err)
		}
		detID, err := resDet.LastInsertId()
		if err != nil {
			return err
		}
		d.ID = int(detID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error al confirmar transacción de pedido: %w", err)
	}

	return nil
}

// GetByID obtiene un pedido completo con sus líneas de detalle y nombres de plato.
func (r *PedidoRepositorySQLite) GetByID(ctx context.Context, id int) (*domain.Pedido, error) {
	queryPedido := `SELECT id, mesa_id, estado, total, created_at, updated_at 
	                FROM pedidos WHERE id = ?`

	var p domain.Pedido
	var estadoStr string
	var createdAtStr, updatedAtStr string

	err := r.db.QueryRowContext(ctx, queryPedido, id).Scan(
		&p.ID,
		&p.MesaID,
		&estadoStr,
		&p.Total,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrPedidoNoEncontrado
		}
		return nil, fmt.Errorf("error al consultar pedido: %w", err)
	}
	p.Estado = domain.EstadoPedido(estadoStr)
	if p.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr); err != nil {
		return nil, fmt.Errorf("error al parsear created_at: %w", err)
	}
	if p.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr); err != nil {
		return nil, fmt.Errorf("error al parsear updated_at: %w", err)
	}

	// Consultar líneas de detalle
	queryDetalles := `SELECT d.id, d.pedido_id, d.plato_id, d.cantidad, d.precio_unitario, d.subtotal, pl.nombre
	                  FROM detalles_pedido d
	                  INNER JOIN platos pl ON d.plato_id = pl.id
	                  WHERE d.pedido_id = ?`

	rows, err := r.db.QueryContext(ctx, queryDetalles, id)
	if err != nil {
		return nil, fmt.Errorf("error al consultar detalles de pedido: %w", err)
	}
	defer rows.Close()

	var detalles []domain.DetallePedido
	for rows.Next() {
		var d domain.DetallePedido
		if err := rows.Scan(
			&d.ID,
			&d.PedidoID,
			&d.PlatoID,
			&d.Cantidad,
			&d.PrecioUnitario,
			&d.Subtotal,
			&d.NombrePlato,
		); err != nil {
			return nil, fmt.Errorf("error al leer detalle de pedido: %w", err)
		}
		detalles = append(detalles, d)
	}
	p.Detalles = detalles

	return &p, nil
}

// UpdateEstado actualiza el estado de un pedido (ej: PENDIENTE -> EN_COCINA -> ENTREGADO).
func (r *PedidoRepositorySQLite) UpdateEstado(ctx context.Context, id int, estado domain.EstadoPedido) error {
	query := `UPDATE pedidos SET estado = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, string(estado), id)
	if err != nil {
		return fmt.Errorf("error al actualizar estado de pedido: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrPedidoNoEncontrado
	}

	return nil
}

// GetAll obtiene todos los pedidos con sus líneas de detalle para KDS.
func (r *PedidoRepositorySQLite) GetAll(ctx context.Context) ([]*domain.Pedido, error) {
	queryPedido := `SELECT id, mesa_id, estado, total, created_at, updated_at 
	                FROM pedidos ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, queryPedido)
	if err != nil {
		return nil, fmt.Errorf("error al consultar pedidos: %w", err)
	}
	defer rows.Close()

	var pedidos []*domain.Pedido
	for rows.Next() {
		var p domain.Pedido
		var estadoStr string
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&p.ID,
			&p.MesaID,
			&estadoStr,
			&p.Total,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("error al leer pedido: %w", err)
		}
		p.Estado = domain.EstadoPedido(estadoStr)
		if p.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr); err != nil {
			return nil, fmt.Errorf("error al parsear created_at: %w", err)
		}
		if p.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr); err != nil {
			return nil, fmt.Errorf("error al parsear updated_at: %w", err)
		}

		pedidos = append(pedidos, &p)
	}

	for _, p := range pedidos {
		queryDetalles := `SELECT d.id, d.pedido_id, d.plato_id, d.cantidad, d.precio_unitario, d.subtotal, pl.nombre
		                  FROM detalles_pedido d
		                  INNER JOIN platos pl ON d.plato_id = pl.id
		                  WHERE d.pedido_id = ?`

		rowsDet, err := r.db.QueryContext(ctx, queryDetalles, p.ID)
		if err != nil {
			return nil, fmt.Errorf("error al consultar detalles de pedido %d: %w", p.ID, err)
		}
		defer rowsDet.Close()

		var detalles []domain.DetallePedido
		for rowsDet.Next() {
			var d domain.DetallePedido
			if err := rowsDet.Scan(
				&d.ID,
				&d.PedidoID,
				&d.PlatoID,
				&d.Cantidad,
				&d.PrecioUnitario,
				&d.Subtotal,
				&d.NombrePlato,
			); err != nil {
				return nil, fmt.Errorf("error al leer detalle de pedido: %w", err)
			}
			detalles = append(detalles, d)
		}
		p.Detalles = detalles
	}

	return pedidos, nil
}
