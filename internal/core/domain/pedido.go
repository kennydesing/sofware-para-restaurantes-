package domain

import "time"

// EstadoPedido define el ciclo de vida de una orden o comanda en el POS/Restaurante.
type EstadoPedido string

const (
	EstadoPendiente EstadoPedido = "PENDIENTE"
	EstadoEnCocina  EstadoPedido = "EN_COCINA"
	EstadoListo     EstadoPedido = "LISTO"
	EstadoEntregado EstadoPedido = "ENTREGADO"
	EstadoCancelado EstadoPedido = "CANCELADO"
)

// Mesa representa una mesa física o virtual de atención al cliente.
type Mesa struct {
	ID        int  `json:"id"`
	Numero    int  `json:"numero"`
	Capacidad int  `json:"capacidad"`
	Activa    bool `json:"activa"`
}

// DetallePedido representa una línea de comanda (un plato y su cantidad solicitada).
type DetallePedido struct {
	ID             int     `json:"id"`
	PedidoID       int     `json:"pedido_id"`
	PlatoID        int     `json:"plato_id"`
	NombrePlato    string  `json:"nombre_plato,omitempty"`
	Cantidad       int     `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
	Subtotal       float64 `json:"subtotal"`
}

// Pedido representa la orden de venta completa de una mesa.
type Pedido struct {
	ID        int             `json:"id"`
	MesaID    int             `json:"mesa_id"`
	Estado    EstadoPedido    `json:"estado"`
	Total     float64         `json:"total"`
	Detalles  []DetallePedido `json:"detalles"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// CalcularTotal calcula el monto total del pedido a partir de sus líneas de detalle.
func (p *Pedido) CalcularTotal() {
	var total float64
	for i := range p.Detalles {
		p.Detalles[i].Subtotal = float64(p.Detalles[i].Cantidad) * p.Detalles[i].PrecioUnitario
		total += p.Detalles[i].Subtotal
	}
	p.Total = total
}

// PuedeSerProcesado valida si el pedido se encuentra en un estado apto para descuento de stock y pase a cocina.
func (p *Pedido) PuedeSerProcesado() bool {
	return p.Estado == EstadoPendiente && len(p.Detalles) > 0
}
