package domain

import (
	"errors"
	"math"
	"time"
)

var (
	ErrTasaInvalida      = errors.New("la tasa de cambio debe ser mayor a cero")
	ErrMonedaNoSoportada = errors.New("moneda no soportada, solo se admite USD y EUR")
	ErrTasaNoEncontrada  = errors.New("tasa de cambio no encontrada")
)

// TasaCambio representa el valor de conversión oficial de una divisa respecto al Bolívar (Bs).
type TasaCambio struct {
	Moneda             string    `json:"moneda"` // "USD", "EUR"
	Tasa               float64   `json:"tasa"`   // Valor en Bs por cada unidad de divisa
	Fuente             string    `json:"fuente"` // "BCV_AUTO", "MANUAL"
	FechaActualizacion time.Time `json:"fecha_actualizacion"`
}

// ConvertirUSDaBs convierte un importe en USD a su equivalente en Bolívares usando la tasa proporcionada.
func ConvertirUSDaBs(montoUSD float64, tasaUSD float64) float64 {
	if tasaUSD <= 0 {
		return 0
	}
	return redondearDecimales(montoUSD*tasaUSD, 2)
}

// ConvertirBsaUSD convierte un importe en Bolívares a su equivalente en USD usando la tasa proporcionada.
func ConvertirBsaUSD(montoBs float64, tasaUSD float64) float64 {
	if tasaUSD <= 0 {
		return 0
	}
	return redondearDecimales(montoBs/tasaUSD, 4)
}

func redondearDecimales(valor float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(valor*ratio) / ratio
}
