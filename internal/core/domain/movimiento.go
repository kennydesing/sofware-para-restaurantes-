package domain

import "time"

// TipoMovimiento identifica la naturaleza y origen de cualquier cambio en el stock físico.
type TipoMovimiento string

const (
	// EntradaCompra: Registro de mercadería comprada a proveedores (suma stock).
	EntradaCompra TipoMovimiento = "ENTRADA_COMPRA"

	// SalidaVenta: Débito automático producido por la venta de un plato según escandallo (resta stock).
	SalidaVenta TipoMovimiento = "SALIDA_VENTA"

	// SalidaMerma: Pérdidas extraordinarias por descomposición, descarte, error culinario o comida de personal (resta stock).
	SalidaMerma TipoMovimiento = "SALIDA_MERMA"

	// AjusteManual: Corrección directa de inventario derivada de auditoría física teórico vs real (+/- stock).
	AjusteManual TipoMovimiento = "AJUSTE_MANUAL"
)

// MovimientoInventario es la entidad del Libro Diario de Auditoría.
// Garantiza la trazabilidad matemática e histórica de cada gramo, mililitro o unidad.
type MovimientoInventario struct {
	ID            int            `json:"id"`
	InsumoID      int            `json:"insumo_id"`
	Tipo          TipoMovimiento `json:"tipo"`
	Cantidad      float64        `json:"cantidad"`       // Magnitud en unidad base ('g', 'ml', 'u')
	CostoUnitario float64        `json:"costo_unitario"` // Costo al momento del movimiento
	Referencia    string         `json:"referencia"`     // Número de factura, comanda, motivo de merma, etc.
	CreatedAt     time.Time      `json:"created_at"`
}