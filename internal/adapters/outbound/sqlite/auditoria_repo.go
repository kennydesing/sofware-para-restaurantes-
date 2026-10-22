package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// AuditoriaRepositorySQLite implementa ports.AuditoriaRepository con SQLite.
// Es estrictamente de solo inserción (INSERT-only) en cumplimiento con RNF-05.
type AuditoriaRepositorySQLite struct {
	db *DB
}

// NewAuditoriaRepository crea un nuevo repositorio de auditoría SQLite.
func NewAuditoriaRepository(db *DB) ports.AuditoriaRepository {
	return &AuditoriaRepositorySQLite{db: db}
}

var _ ports.AuditoriaRepository = (*AuditoriaRepositorySQLite)(nil)

// Create registra de forma inmutable un nuevo evento de auditoría en SQLite.
func (r *AuditoriaRepositorySQLite) Create(ctx context.Context, e *domain.EventoAuditoria) error {
	query := `INSERT INTO auditoria_acciones 
	          (usuario_id, accion, entidad_tipo, entidad_id, detalle, ip_origen, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`

	res, err := r.db.ExecContext(ctx, query,
		e.UsuarioID,
		string(e.Accion),
		e.EntidadTipo,
		e.EntidadID,
		e.Detalle,
		e.IPOrigen,
	)
	if err != nil {
		return fmt.Errorf("error al insertar evento de auditoría: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		e.ID = int(id)
	}

	return nil
}

// GetFiltered consulta el log de auditoría aplicando filtros opcionales y paginación.
func (r *AuditoriaRepositorySQLite) GetFiltered(ctx context.Context, filtro ports.FiltroAuditoria) ([]*domain.EventoAuditoria, error) {
	var conditions []string
	var args []interface{}

	if filtro.UsuarioID != nil {
		conditions = append(conditions, "usuario_id = ?")
		args = append(args, *filtro.UsuarioID)
	}

	if filtro.Desde != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filtro.Desde+" 00:00:00")
	}

	if filtro.Hasta != "" {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filtro.Hasta+" 23:59:59")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	porPagina := filtro.PorPagina
	if porPagina <= 0 {
		porPagina = 50
	}

	pagina := filtro.Pagina
	if pagina <= 0 {
		pagina = 1
	}
	offset := (pagina - 1) * porPagina

	query := fmt.Sprintf(`SELECT id, usuario_id, accion, entidad_tipo, entidad_id, detalle, ip_origen, created_at
	                      FROM auditoria_acciones
	                      %s
	                      ORDER BY created_at DESC, id DESC
	                      LIMIT ? OFFSET ?`, whereClause)

	args = append(args, porPagina, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error al consultar eventos de auditoría: %w", err)
	}
	defer rows.Close()

	var eventos []*domain.EventoAuditoria
	for rows.Next() {
		var e domain.EventoAuditoria
		var usuarioID sql.NullInt64
		var entidadID sql.NullInt64
		var createdAt time.Time
		var accionStr string

		if err := rows.Scan(
			&e.ID,
			&usuarioID,
			&accionStr,
			&e.EntidadTipo,
			&entidadID,
			&e.Detalle,
			&e.IPOrigen,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("error al escanear evento de auditoría: %w", err)
		}

		if usuarioID.Valid {
			uid := int(usuarioID.Int64)
			e.UsuarioID = &uid
		}
		if entidadID.Valid {
			eid := int(entidadID.Int64)
			e.EntidadID = &eid
		}
		e.Accion = domain.AccionAuditoria(accionStr)
		e.CreatedAt = createdAt

		eventos = append(eventos, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando eventos de auditoría: %w", err)
	}

	return eventos, nil
}
