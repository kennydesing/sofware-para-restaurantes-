package ports

import (
	"context"
	"pos-inventory/internal/core/domain"
)

// Outbound Ports (Driven Ports / Interfaces de Salida hacia Adaptadores)

// InsumoRepository define el contrato de persistencia para materias primas.
type InsumoRepository interface {
	GetByID(ctx context.Context, id int) (*domain.Insumo, error)
	GetByNombre(ctx context.Context, nombre string) (*domain.Insumo, error)
	GetAll(ctx context.Context) ([]*domain.Insumo, error)
	GetAllEnriquecido(ctx context.Context) ([]*InsumoEnriquecido, error)
	Create(ctx context.Context, insumo *domain.Insumo) error
	Update(ctx context.Context, insumo *domain.Insumo) error
	Delete(ctx context.Context, id int) error
	UpdateStockAndCosto(ctx context.Context, id int, nuevoStock float64, nuevoCostoPromedio float64) error
}

// PlatoRepository define el contrato de persistencia para el catálogo de platos.
type PlatoRepository interface {
	GetByID(ctx context.Context, id int) (*domain.Plato, error)
	GetAll(ctx context.Context) ([]*domain.Plato, error)
	Create(ctx context.Context, plato *domain.Plato) error
}

// RecetaRepository define el contrato para consultar las fichas técnicas / escandallos.
type RecetaRepository interface {
	GetByPlatoID(ctx context.Context, platoID int) ([]domain.RecetaItem, error)
	Create(ctx context.Context, item *domain.RecetaItem) error
}

// PedidoRepository define el contrato de persistencia para comandas y ventas.
type PedidoRepository interface {
	Create(ctx context.Context, pedido *domain.Pedido) error
	GetByID(ctx context.Context, id int) (*domain.Pedido, error)
	GetAll(ctx context.Context) ([]*domain.Pedido, error)
	UpdateEstado(ctx context.Context, id int, estado domain.EstadoPedido) error
}

// MovimientoRepository define el contrato de persistencia para el libro diario de auditoría.
type MovimientoRepository interface {
	Create(ctx context.Context, mov *domain.MovimientoInventario) error
	GetByInsumoID(ctx context.Context, insumoID int) ([]*domain.MovimientoInventario, error)
	// ListarHistorialMermas devuelve movimientos SALIDA_MERMA que no son uso personal.
	ListarHistorialMermas(ctx context.Context, limite int) ([]*HistorialSalidaDTO, error)
	// ListarHistorialUsoPersonal devuelve movimientos SALIDA_MERMA marcados como USO_PERSONAL.
	ListarHistorialUsoPersonal(ctx context.Context, limite int) ([]*HistorialSalidaDTO, error)
}

// ConsumoInsumoDTO encapsula el requerimiento de descuento de un insumo dentro de una transacción de pedido.
type ConsumoInsumoDTO struct {
	InsumoID      int
	NombreInsumo  string
	Cantidad      float64 // Cantidad total calculada en unidad base (incluyendo merma)
	CostoUnitario float64
	Referencia    string
}

// AtomicDebitor define el contrato de transacción atómica para el procesamiento de pedidos.
// En una única transacción SQLite (ACID):
// 1. Verifica con bloqueo que todos los insumos tengan stock suficiente.
// 2. Descuenta el stock de cada insumo.
// 3. Inserta los registros correspondientes en movimientos_inventario (SALIDA_VENTA).
// 4. Actualiza el estado del pedido a EN_COCINA.
// Si cualquier paso o ingrediente falla, ejecuta Rollback total garantizando consistencia absoluta.
type AtomicDebitor interface {
	DebitarStockPedidoAtomic(ctx context.Context, pedidoID int, consumos []ConsumoInsumoDTO) error
}

// PrinterRepository define el contrato de salida para enviar comandas a la impresora térmica de cocina.
type PrinterRepository interface {
	ImprimirComanda(ctx context.Context, pedido *domain.Pedido) error
}

// NotifierRepository define el contrato de salida para alertas operativas (ej: stock bajo).
type NotifierRepository interface {
	NotificarStockBajo(ctx context.Context, insumo *domain.Insumo) error
	NotificarCambioEstado(ctx context.Context, pedidoID int, estado string) error
}

// ---- Outbound Ports para Autenticación y Auditoría ----

// UsuarioRepository define el contrato de persistencia para usuarios.
// Implementado en internal/adapters/outbound/sqlite/usuario_repo.go
type UsuarioRepository interface {
	GetByID(ctx context.Context, id int) (*domain.Usuario, error)
	GetByEmail(ctx context.Context, email string) (*domain.Usuario, error)
	GetAll(ctx context.Context) ([]*domain.Usuario, error)
	Create(ctx context.Context, u *domain.Usuario) error
	Update(ctx context.Context, u *domain.Usuario) error // solo para rol/activo — nunca para hash
}

// SesionRepository define el contrato de persistencia para sesiones activas.
// Implementado en internal/adapters/outbound/sqlite/sesion_repo.go
type SesionRepository interface {
	Create(ctx context.Context, s *domain.Sesion) error
	GetByToken(ctx context.Context, token string) (*domain.Sesion, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteByUsuarioID(ctx context.Context, usuarioID int) error   // para SINGLE_SESSION
	DeleteExpired(ctx context.Context) (int64, error)             // limpieza periódica
}

// AuditoriaRepository define el contrato INSERT-only para el log de auditoría.
// Implementado en internal/adapters/outbound/sqlite/auditoria_repo.go
// IMPORTANTE: No se define Update ni Delete — violación de RNF-05.
type AuditoriaRepository interface {
	Create(ctx context.Context, e *domain.EventoAuditoria) error
	GetFiltered(ctx context.Context, filtro FiltroAuditoria) ([]*domain.EventoAuditoria, error)
}

// TasaCambioRepository define la persistencia para tasas oficiales de cambio (USD/EUR a Bs).
type TasaCambioRepository interface {
	GetTasa(ctx context.Context, moneda string) (*domain.TasaCambio, error)
	GetAllTasas(ctx context.Context) ([]*domain.TasaCambio, error)
	GuardarTasa(ctx context.Context, tasa *domain.TasaCambio) error
}

// BCVRateProvider define el cliente HTTP de integración para consultar las tasas oficiales del BCV.
type BCVRateProvider interface {
	ObtenerTasasBCV(ctx context.Context) (usd float64, eur float64, err error)
}

// ---- Outbound Ports para Unidades de Medida (Spec 003) ----

// UnidadMedidaRepository define el contrato de persistencia para el catálogo de unidades
// y las equivalencias de compra por insumo.
// Implementado en internal/adapters/outbound/sqlite/unidad_repo.go
type UnidadMedidaRepository interface {
	ObtenerTodas(ctx context.Context) ([]domain.UnidadMedida, error)
	ObtenerPorID(ctx context.Context, id int64) (domain.UnidadMedida, error)
	ObtenerPorAbreviatura(ctx context.Context, abrev string) (domain.UnidadMedida, error)
	ObtenerPorCategoria(ctx context.Context, cat domain.Categoria) ([]domain.UnidadMedida, error)
	Crear(ctx context.Context, u domain.UnidadMedida) (int64, error)
	ObtenerUnidadesCompraInsumo(ctx context.Context, insumoID int64) ([]domain.InsumoUnidadCompra, error)
	GuardarUnidadCompraInsumo(ctx context.Context, iuc domain.InsumoUnidadCompra) error
}

// InsumoEnriquecido extiende domain.Insumo con campos calculados de la BD.
// No pertenece al dominio puro; vive en ports como DTO de capa de aplicación.
type InsumoEnriquecido struct {
	domain.Insumo
	PrecioVentaUSD      float64 // USD, default 0.0
	MermaAcumulada      float64 // suma de SALIDA_MERMA en unidad base
	UnidadCompraAbrev   string  // abreviatura de la unidad de compra principal (ej: "kg", "L", "u")
	StockMinimoEnCompra float64 // stock mínimo convertido a la unidad de compra principal
}
