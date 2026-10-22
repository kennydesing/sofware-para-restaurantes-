package domain

// Plato representa un ítem o producto terminado disponible en el menú del restaurante.
// Cada plato tiene un precio de venta y puede estar asociado a una receta de escandallo
// para el débito automático de insumos en cocina.
type Plato struct {
	ID          int     `json:"id"`
	Nombre      string  `json:"nombre"`
	PrecioVenta float64 `json:"precio_venta"`
	Activo      bool    `json:"activo"`
}