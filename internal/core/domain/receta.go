package domain

// RecetaItem representa un ingrediente dentro de la ficha técnica / escandallo de un plato.
// Define la cantidad bruta teórica requerida y el porcentaje de merma inherente
// a su manipulación, cocción, limpieza o descarte en cocina.
type RecetaItem struct {
	ID                int     `json:"id"`
	PlatoID           int     `json:"plato_id"`
	InsumoID          int     `json:"insumo_id"`
	CantidadRequerida float64 `json:"cantidad_requerida"` // Cantidad neta/bruta en unidad base (g, ml, u)
	PorcentajeMerma   float64 `json:"porcentaje_merma"`   // Porcentaje de merma estimado (ej: 15 para 15%)
	UnidadRecetaID    int     `json:"unidad_receta_id,omitempty"` // FK a unidades_medida; 0 = usa unidad base del insumo (legacy)
	UnidadRecetaAbrev string  `json:"unidad_receta_abrev,omitempty"` // Abreviatura de la unidad de receta (ej: "tbsp", "ml", "g")

	// Campos enriquecidos opcionales para visualización o reportes
	NombreInsumo string `json:"nombre_insumo,omitempty"`
	UnidadBase   string `json:"unidad_base,omitempty"`
}

// CalcularCantidadConsumidaUnitaria calcula el consumo real de materia prima para 1 unidad del plato,
// incorporando el factor de merma según la fórmula del negocio:
//
//	Cantidad Consumida = Cantidad Requerida * (1 + Porcentaje Merma / 100)
//
// Ejemplo: 200g de carne con 10% de merma por cocción/recorte:
//
//	Consumo = 200 * (1 + 10/100) = 200 * 1.10 = 220g reales a descontar del inventario.
func (r *RecetaItem) CalcularCantidadConsumidaUnitaria() float64 {
	return r.CantidadRequerida * (1.0 + (r.PorcentajeMerma / 100.0))
}

// CalcularConsumoTotal calcula la cantidad total a descontar de stock para una cantidad específica de platos pedidos.
func (r *RecetaItem) CalcularConsumoTotal(cantidadPlatos int) float64 {
	if cantidadPlatos <= 0 {
		return 0
	}
	return r.CalcularCantidadConsumidaUnitaria() * float64(cantidadPlatos)
}