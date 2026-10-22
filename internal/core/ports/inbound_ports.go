package ports

import (
	"context"
	"time"

	"pos-inventory/internal/core/domain"
)

// DTOs de Entrada para Casos de Uso

// CompraRequest contiene los datos para registrar una entrada de mercadería.
type CompraRequest struct {
	InsumoID      int     `json:"insumo_id"`
	Cantidad      float64 `json:"cantidad"`       // Cantidad en la unidad comercial comprada (ej: 5.0 kg)
	UnidadCompra  string  `json:"unidad_compra"`  // "kg", "l", "u", "docena", "g", "ml", etc.
	CostoUnitario float64 `json:"costo_unitario"` // Costo monetario por unidad de compra
	Referencia    string  `json:"referencia"`     // Número de factura o proveedor
}

// MermaRequest contiene los datos para registrar una salida extraordinaria.
type MermaRequest struct {
	InsumoID     int     `json:"insumo_id"`
	Cantidad     float64 `json:"cantidad"`      // Cantidad en unidad de merma
	UnidadMedida string  `json:"unidad_medida"` // Unidad (se convertirá a unidad base)
	Motivo       string  `json:"motivo"`        // "Descomposición", "Error en cocina", etc.
}

// UsoPersonalRequest contiene los datos para imputar consumo de personal.
type UsoPersonalRequest struct {
	InsumoID     int     `json:"insumo_id"`
	Cantidad     float64 `json:"cantidad"`      // Magnitud en unidad física
	UnidadMedida string  `json:"unidad_medida"` // Opcional: si difiere de unidad base
	Detalle      string  `json:"detalle"`       // "Comida de turno", "Desayuno personal", etc.
}

// HistorialSalidaDTO estructura la respuesta de lectura para Mermas y Uso Personal.
type HistorialSalidaDTO struct {
	ID           int       `json:"id"`
	InsumoID     int       `json:"insumo_id"`
	InsumoNombre string    `json:"insumo_nombre"`
	Cantidad     float64   `json:"cantidad"`
	UnidadBase   string    `json:"unidad_base"`
	CostoTotal   float64   `json:"costo_total"`
	Referencia   string    `json:"referencia"`
	CreatedAt    time.Time `json:"created_at"`
}

// AjusteRequest contiene los datos de una corrección física de inventario.
type AjusteRequest struct {
	InsumoID     int     `json:"insumo_id"`
	StockFisico  float64 `json:"stock_fisico"`  // Conteo físico real obtenido en auditoría (en unidad base)
	Motivo       string  `json:"motivo"`        // "Auditoría mensual", "Corrección de descuadre", etc.
}

// ItemPedidoRequest representa un plato y cantidad dentro de una orden.
type ItemPedidoRequest struct {
	PlatoID  int `json:"plato_id"`
	Cantidad int `json:"cantidad"`
}

// CrearPedidoRequest contiene los datos para abrir una nueva orden de venta.
type CrearPedidoRequest struct {
	MesaID int                 `json:"mesa_id"`
	Items  []ItemPedidoRequest `json:"items"`
}

// AuditoriaStockDTO resume la comparación entre inventario teórico y conteo físico real.
type AuditoriaStockDTO struct {
	InsumoID             int     `json:"insumo_id"`
	NombreInsumo         string  `json:"nombre_insumo"`
	UnidadBase           string  `json:"unidad_base"`
	StockTeorico         float64 `json:"stock_teorico"`
	StockFisico          float64 `json:"stock_fisico"`
	Diferencia           float64 `json:"diferencia"`             // StockFisico - StockTeorico
	DescuadreValorizado  float64 `json:"descuadre_valorizado"`   // Diferencia * CostoPromedio
}

// Inbound Ports (Driving Ports / Interfaces de Entrada)

// InventarioService define los casos de uso para la administración integral de materias primas.
type InventarioService interface {
	// RegistrarCompra registra una entrada con conversión automática a unidad base (kg/L -> g/ml),
	// actualiza el costo promedio ponderado y registra el movimiento de auditoría.
	RegistrarCompra(ctx context.Context, req CompraRequest) (*domain.Insumo, error)

	// RegistrarMerma descuenta stock por pérdidas operativas extraordinarias y audita el motivo.
	RegistrarMerma(ctx context.Context, req MermaRequest) (*domain.MovimientoInventario, error)

	// RegistrarUsoPersonal descuenta stock por alimentación del personal del restaurante.
	RegistrarUsoPersonal(ctx context.Context, req UsoPersonalRequest) (*domain.MovimientoInventario, error)

	// ListarHistorialMermas devuelve el historial de pérdidas operativas registradas.
	ListarHistorialMermas(ctx context.Context, limite int) ([]*HistorialSalidaDTO, error)

	// ListarHistorialUsoPersonal devuelve el historial de consumo de insumos por el personal.
	ListarHistorialUsoPersonal(ctx context.Context, limite int) ([]*HistorialSalidaDTO, error)

	// AjustarStock ajusta el stock tras un conteo físico manual y registra el movimiento con la diferencia.
	AjustarStock(ctx context.Context, req AjusteRequest) (*domain.MovimientoInventario, error)

	// ObtenerInsumo consulta la información y stock actual de un insumo.
	ObtenerInsumo(ctx context.Context, id int) (*domain.Insumo, error)

	// ListarInsumos devuelve el catálogo completo de insumos con sus stocks y costos.
	ListarInsumos(ctx context.Context) ([]*domain.Insumo, error)

	// AuditarStock compara el stock teórico del sistema contra el conteo físico real.
	AuditarStock(ctx context.Context, insumoID int, conteoFisico float64) (*AuditoriaStockDTO, error)
}

// PedidoService define los casos de uso para la venta POS y consumo atómico por escandallo.
type PedidoService interface {
	// CrearPedido genera una nueva orden en estado PENDIENTE calculando los subtotales de platos.
	CrearPedido(ctx context.Context, req CrearPedidoRequest) (*domain.Pedido, error)

	// ProcesarPedido ejecuta el débito atómico de materia prima por escandallo (g/ml con merma),
	// registra los movimientos de SALIDA_VENTA, cambia el estado a EN_COCINA e imprime la comanda.
	// Si un solo insumo no tiene suficiente stock, la operación completa es abortada (Rollback).
	ProcesarPedido(ctx context.Context, pedidoID int) error

	// ActualizarEstadoPedido cambia el estado de un pedido (PENDIENTE, EN_COCINA, LISTO, ENTREGADO, CANCELADO).
	ActualizarEstadoPedido(ctx context.Context, pedidoID int, estado string) error

	// ObtenerPedido consulta el estado y detalle de una comanda.
	ObtenerPedido(ctx context.Context, id int) (*domain.Pedido, error)

	// ListarPedidos devuelve todos los pedidos con sus items para KDS.
	ListarPedidos(ctx context.Context) ([]*PedidoWithItems, error)
}

type PedidoWithItems struct {
	ID        int
	MesaID    int
	Estado    string
	Items     []ItemPedidoConPlato
	Total     float64
	CreatedAt string
}

type ItemPedidoConPlato struct {
	PlatoID int
	Nombre  string
	Cantidad int
	Precio  float64
}

// ---- DTOs de Autenticación y Auditoría ----

// LoginRequest contiene las credenciales enviadas por el usuario.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"` // contraseña en texto plano (solo en tránsito)
	IPOrigen string `json:"-"`        // extraído del request HTTP, no del body
}

// CrearUsuarioRequest contiene los datos para registrar un nuevo operador.
type CrearUsuarioRequest struct {
	Nombre   string      `json:"nombre"`
	Email    string      `json:"email"`
	Password string      `json:"password"` // texto plano — se hashea antes de persistir
	Rol      domain.Rol  `json:"rol"`
}

// ModificarUsuarioRequest permite cambiar el rol o el estado activo de un usuario.
type ModificarUsuarioRequest struct {
	Rol    *domain.Rol `json:"rol,omitempty"`
	Activo *bool       `json:"activo,omitempty"`
}

// FiltroAuditoria encapsula los parámetros de búsqueda para el visor de auditoría.
type FiltroAuditoria struct {
	UsuarioID *int
	Desde     string // formato "2006-01-02"
	Hasta     string // formato "2006-01-02"
	Pagina    int    // 1-indexed
	PorPagina int    // default 50
}

// AuthService define los casos de uso de autenticación y gestión de usuarios.
// Implementado en internal/core/services/auth_service.go
type AuthService interface {
	// Login verifica credenciales y crea una sesión persistente.
	// Retorna la sesión creada (con token) o error (ErrCredencialesInvalidas, ErrCuentaInactiva).
	Login(ctx context.Context, req LoginRequest) (*domain.Sesion, error)

	// Logout invalida la sesión asociada al token.
	Logout(ctx context.Context, token string) error

	// ValidarSesion verifica que el token sea válido y no esté expirado.
	// Retorna el Usuario completo (para poblar el contexto HTTP) o error.
	ValidarSesion(ctx context.Context, token string) (*domain.Usuario, error)

	// CrearUsuario registra un nuevo usuario. Solo el rol admin puede invocar este caso de uso.
	CrearUsuario(ctx context.Context, req CrearUsuarioRequest) (*domain.Usuario, error)

	// ModificarUsuario actualiza el rol o estado activo de un usuario existente.
	ModificarUsuario(ctx context.Context, usuarioID int, req ModificarUsuarioRequest) (*domain.Usuario, error)

	// ListarUsuarios devuelve todos los usuarios del sistema (sin hash de contraseña).
	ListarUsuarios(ctx context.Context) ([]*domain.Usuario, error)

	// LimpiarSesionesExpiradas elimina tokens cuyo expires_at ha vencido.
	LimpiarSesionesExpiradas(ctx context.Context) (int64, error)
}

// AuditoriaService define los casos de uso para el registro y consulta de eventos de auditoría.
type AuditoriaService interface {
	// RegistrarEvento inserta un evento inmutable en el log de auditoría.
	RegistrarEvento(ctx context.Context, evento *domain.EventoAuditoria) error

	// ListarEventos devuelve eventos filtrados por rango de fecha y usuario, paginados.
	ListarEventos(ctx context.Context, filtro FiltroAuditoria) ([]*domain.EventoAuditoria, error)
}

// TasaCambioService define los casos de uso para la gestión y conversión de divisas a tasa oficial BCV.
type TasaCambioService interface {
	ObtenerTasas(ctx context.Context) ([]*domain.TasaCambio, error)
	ObtenerTasa(ctx context.Context, moneda string) (*domain.TasaCambio, error)
	SincronizarTasasBCV(ctx context.Context) ([]*domain.TasaCambio, error)
	 ActualizarTasaManual(ctx context.Context, moneda string, nuevaTasa float64) error
	ConvertirUSDaBs(ctx context.Context, montoUSD float64) (float64, error)
}

// ConversionService define los casos de uso para la conversión exacta entre unidades de medida.
// Implementado en internal/core/services/conversion_service.go
type ConversionService interface {
	// Convertir convierte una cantidad entre dos unidades de la misma categoría usando
	// aritmética de fracciones exactas (INT64) y materializa el resultado con 4 decimales.
	Convertir(ctx context.Context, cantidad float64, origenAbrev, destinoAbrev string) (float64, error)

	// CostoUnitarioBase calcula el costo por unidad base mínima a partir del total de la compra,
	// la cantidad comprada y la unidad de compra registrada para el insumo.
	CostoUnitarioBase(ctx context.Context, costoTotal, cantidadComprada float64, unidadCompraAbrev string, insumoID int64) (float64, error)

	// ConsumoReceta calcula la cantidad en unidad base del insumo a descontar del stock,
	// aplicando la fórmula de merma del negocio.
	ConsumoReceta(ctx context.Context, cantidadReceta float64, unidadRecetaAbrev string, insumoID int64, porcentajeMerma float64, numPlatos int) (float64, error)
}

