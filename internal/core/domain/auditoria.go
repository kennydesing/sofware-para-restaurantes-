package domain

import "time"

// AccionAuditoria define el catálogo cerrado de eventos auditables.
type AccionAuditoria string

const (
	AccionLogin              AccionAuditoria = "LOGIN"
	AccionLogout             AccionAuditoria = "LOGOUT"
	AccionLoginFallido       AccionAuditoria = "LOGIN_FALLIDO"
	AccionCrearUsuario       AccionAuditoria = "CREAR_USUARIO"
	AccionDesactivarUsuario  AccionAuditoria = "DESACTIVAR_USUARIO"
	AccionModificarRol       AccionAuditoria = "MODIFICAR_ROL"
	AccionCambioEstadoPedido AccionAuditoria = "CAMBIO_ESTADO_PEDIDO"
	AccionCobroPedido        AccionAuditoria = "COBRO_PEDIDO"
	AccionCancelarPedido     AccionAuditoria = "CANCELAR_PEDIDO"
	AccionRegistrarCompra    AccionAuditoria = "REGISTRAR_COMPRA"
	AccionRegistrarMerma     AccionAuditoria = "REGISTRAR_MERMA"
	AccionAjusteManual       AccionAuditoria = "AJUSTE_MANUAL"
	AccionConteoFisico       AccionAuditoria = "CONTEO_FISICO"
)

// EventoAuditoria es el registro inmutable de una acción crítica del sistema.
type EventoAuditoria struct {
	ID          int
	UsuarioID   *int            // nil si es acción del sistema
	Accion      AccionAuditoria
	EntidadTipo string          // "pedido", "usuario", "insumo"
	EntidadID   *int            // nil si no aplica
	Detalle     string          // contexto libre: "PENDIENTE→EN_COCINA", "$1500.00", etc.
	IPOrigen    string
	CreatedAt   time.Time
}