package domain

import (
	"fmt"
	"strings"
)

// TipoEmpaque define las categorías estándar y personalizadas de empaque de bebidas.
type TipoEmpaque string

const (
	EmpaqueUnitario      TipoEmpaque = "UNITARIO"
	EmpaqueGavera36      TipoEmpaque = "GAVERA_36"
	EmpaqueCaja24Bot     TipoEmpaque = "CAJA_24_BOT"
	EmpaqueCaja24Lata    TipoEmpaque = "CAJA_24_LATA"
	EmpaqueSixPack6      TipoEmpaque = "SIX_PACK_6"
	EmpaqueCaja12Art     TipoEmpaque = "CAJA_12_ART"
	EmpaquePersonalizado TipoEmpaque = "PERSONALIZADO"
)

// Insumo representa una materia prima o producto físico del restaurante.
// En la base de datos y en el modelo de dominio, toda cantidad se almacena
// de forma estricta en su unidad base mínima:
//   - 'g'  (gramos)
//   - 'ml' (mililitros)
//   - 'u'  (unidades enteras/fraccionadas)
//
// Regla Hexagonal: Este archivo no importa drivers, bases de datos ni routers.
type Insumo struct {
	ID                   int         `json:"id"`
	Nombre               string      `json:"nombre"`
	UnidadBase           string      `json:"unidad_base"`           // "g", "ml", "u"
	CostoPromedio        float64     `json:"costo_promedio"`        // Costo monetario por unidad base mínima
	StockMinimo          float64     `json:"stock_minimo"`          // Umbral de alerta de reposición
	Stock                float64     `json:"stock"`                 // Stock físico actual en unidad base
	PrecioVentaUSD              float64     `json:"precio_venta_usd"`              // Precio de venta al público por unidad base (USD)
	PresentacionNombre          string      `json:"presentacion_nombre"`           // Nombre de la presentación física (pieza, caja, cabeza, saco, botella)
	CantidadBasePorPresentacion float64     `json:"cantidad_base_por_presentacion"`  // Unidades base (g/ml/u) que contiene una presentación
	EmpaqueTipo                 TipoEmpaque `json:"empaque_tipo"`                  // Categoría de empaque
	EmpaqueNombre               string      `json:"empaque_nombre"`                // Nombre comercial o personalizado
	UnidadesPorEmpaque          int         `json:"unidades_por_empaque"`          // Ej: 36, 24, 6, 12 o N
	ContenidoMlPorUnidad        float64     `json:"contenido_ml_por_unidad"`         // Ej: 222, 250, 330, 355 ml
	EsRetornable                bool        `json:"es_retornable"`                 // True para envases retornables (gaveras)
}

// ConvertirAUnidadBase convierte una cantidad dada en una unidad comercial (ej: kg, L, docena)
// a la unidad base requerida por el insumo (g, ml, u).
//
// Factores de conversión soportados:
//   - Para masa ('g'): "kg" -> 1000g, "g" -> 1g, "mg" -> 0.001g, "lb" -> 453.592g
//   - Para volumen ('ml'): "l" / "lt" -> 1000ml, "ml" -> 1ml, "cc" -> 1ml
//   - Para conteo ('u'): "u" / "un" -> 1u, "docena" -> 12u, "caja6" -> 6u, "caja12" -> 12u, "caja24" -> 24u
func ConvertirAUnidadBase(cantidad float64, unidadCompra string, unidadBaseDestino string) (float64, error) {
	unidadCompraNorm := strings.ToLower(strings.TrimSpace(unidadCompra))
	unidadBaseNorm := strings.ToLower(strings.TrimSpace(unidadBaseDestino))

	switch unidadBaseNorm {
	case "g":
		switch unidadCompraNorm {
		case "kg", "kilo", "kilogramo", "kilogramos":
			return cantidad * 1000.0, nil
		case "g", "gr", "gramo", "gramos":
			return cantidad, nil
		case "lb", "libra", "libras":
			return cantidad * 453.59237, nil
		default:
			return 0, fmt.Errorf("unidad de compra '%s' incompatible con unidad base 'g'", unidadCompra)
		}

	case "ml":
		switch unidadCompraNorm {
		case "l", "lt", "litro", "litros":
			return cantidad * 1000.0, nil
		case "ml", "mililitro", "mililitros", "cc":
			return cantidad, nil
		default:
			return 0, fmt.Errorf("unidad de compra '%s' incompatible con unidad base 'ml'", unidadCompra)
		}

	case "u":
		switch unidadCompraNorm {
		case "u", "un", "unidad", "unidades":
			return cantidad, nil
		case "docena", "docenas":
			return cantidad * 12.0, nil
		case "caja6":
			return cantidad * 6.0, nil
		case "caja12":
			return cantidad * 12.0, nil
		case "caja24":
			return cantidad * 24.0, nil
		default:
			return 0, fmt.Errorf("unidad de compra '%s' incompatible con unidad base 'u'", unidadCompra)
		}

	default:
		return 0, fmt.Errorf("unidad base destino '%s' no es válida (esperado 'g', 'ml' o 'u')", unidadBaseDestino)
	}
}

// CalcularNuevoCostoPromedio calcula el nuevo costo promedio ponderado tras una compra.
//
// Fórmula de Promedio Ponderado:
//
//	Costo_Nuevo = ((Stock_Actual * Costo_Actual) + (Cantidad_Comprada * Costo_Unitario_Compra))
//	              / (Stock_Actual + Cantidad_Comprada)
//
// Si el stock actual es <= 0 (por ajustes o inventario inicial), el nuevo costo promedio
// adopta directamente el costo unitario de la compra.
func (i *Insumo) CalcularNuevoCostoPromedio(cantidadCompradaBase float64, costoUnitarioBase float64) float64 {
	if cantidadCompradaBase <= 0 {
		return i.CostoPromedio
	}

	if i.Stock <= 0 {
		return costoUnitarioBase
	}

	valorInventarioActual := i.Stock * i.CostoPromedio
	valorCompraEntrante := cantidadCompradaBase * costoUnitarioBase
	stockTotalNuevo := i.Stock + cantidadCompradaBase

	return (valorInventarioActual + valorCompraEntrante) / stockTotalNuevo
}

// TieneStockSuficiente valida si el insumo cuenta con stock disponible para un consumo.
func (i *Insumo) TieneStockSuficiente(cantidadRequerida float64) bool {
	return i.Stock >= cantidadRequerida
}

// EstaBajoMinimo indica si el stock actual ha caído al nivel de stock mínimo o por debajo.
func (i *Insumo) EstaBajoMinimo() bool {
	return i.Stock <= i.StockMinimo
}

// UnidadesPorDefectoEmpaque devuelve el factor canónico según el tipo predefinido.
func UnidadesPorDefectoEmpaque(tipo TipoEmpaque) int {
	switch tipo {
	case EmpaqueGavera36:
		return 36
	case EmpaqueCaja24Bot, EmpaqueCaja24Lata:
		return 24
	case EmpaqueCaja12Art:
		return 12
	case EmpaqueSixPack6:
		return 6
	default:
		return 1
	}
}

// NormalizarEmpaque asegura que EmpaqueTipo y UnidadesPorEmpaque tengan valores canónicos válidos.
func (i *Insumo) NormalizarEmpaque() {
	switch i.EmpaqueTipo {
	case EmpaqueGavera36, EmpaqueCaja24Bot, EmpaqueCaja24Lata, EmpaqueSixPack6, EmpaqueCaja12Art, EmpaquePersonalizado:
		// válido
	default:
		i.EmpaqueTipo = EmpaqueUnitario
	}
	if i.UnidadesPorEmpaque < 1 {
		i.UnidadesPorEmpaque = UnidadesPorDefectoEmpaque(i.EmpaqueTipo)
		if i.UnidadesPorEmpaque < 1 {
			i.UnidadesPorEmpaque = 1
		}
	}
	if i.ContenidoMlPorUnidad < 0 {
		i.ContenidoMlPorUnidad = 0
	}
}

// NormalizarPresentacion asegura que cada insumo tenga una presentación válida.
// Si no se configura explícitamente, adopta "unidad" con factor 1 (1 base = 1 unidad).
func (i *Insumo) NormalizarPresentacion() {
	if i.CantidadBasePorPresentacion <= 0 {
		i.CantidadBasePorPresentacion = 1.0
	}
	if strings.TrimSpace(i.PresentacionNombre) == "" {
		i.PresentacionNombre = "unidad"
	}
}

// CantidadEnPresentacion devuelve cuántas presentaciones físicas representa el stock actual.
// Ej: 3000 g con presentación "pieza" de 1500 g → 2 piezas.
func (i *Insumo) CantidadEnPresentacion() float64 {
	base := i.CantidadBasePorPresentacion
	if base <= 0 {
		base = 1.0
	}
	return i.Stock / base
}

// FormatearPeso devuelve el stock en una unidad legible para humanos según la unidad base.
//   - g:  kg si stock >= 1000, de lo contrario g
//   - ml: L  si stock >= 1000, de lo contrario ml
//   - u:  u
func FormatearPeso(stock float64, unidadBase string) (valor float64, unidad string) {
	switch strings.ToLower(strings.TrimSpace(unidadBase)) {
	case "g":
		if stock >= 1000.0 {
			return stock / 1000.0, "kg"
		}
		return stock, "g"
	case "ml":
		if stock >= 1000.0 {
			return stock / 1000.0, "L"
		}
		return stock, "ml"
	case "u":
		return stock, "u"
	default:
		return stock, unidadBase
	}
}

// ConvertirEmpaquesABase multiplica el número de empaques comprados por las unidades por bulto.
func (i *Insumo) ConvertirEmpaquesABase(cantidadEmpaques float64) float64 {
	factor := float64(i.UnidadesPorEmpaque)
	if factor <= 0 {
		factor = 1.0
	}
	return cantidadEmpaques * factor
}