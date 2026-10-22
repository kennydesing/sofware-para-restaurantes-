package domain

import (
	"errors"
	"math"
)

var (
	// ErrCategoriaIncompatible se devuelve al intentar convertir entre unidades de categorías distintas.
	ErrCategoriaIncompatible = errors.New("unidades de categorías incompatibles")

	// ErrDenominadorCero se devuelve cuando algún denominador de la fracción es cero.
	ErrDenominadorCero = errors.New("denominador de conversión es cero")
)

// ConvertirEntreMismaCategoria realiza una conversión exacta usando aritmética de fracciones.
//
// Fórmula:
//
//	cantidad_base = (cantidadN * factorN_origen * factorD_destino)
//	              / (cantidadD * factorD_origen * factorN_destino)
//
// La división entera se omite deliberadamente: se retorna la fracción (rN, rD)
// y se materializa a float64 solo en la capa de presentación.
func ConvertirEntreMismaCategoria(
	cantidadN, cantidadD int64,
	origen, destino UnidadMedida,
) (rN, rD int64, err error) {
	if origen.Categoria != destino.Categoria {
		return 0, 0, ErrCategoriaIncompatible
	}
	if origen.FactorD == 0 || destino.FactorD == 0 || cantidadD == 0 || destino.FactorN == 0 {
		return 0, 0, ErrDenominadorCero
	}

	rN = cantidadN * origen.FactorN * destino.FactorD
	rD = cantidadD * origen.FactorD * destino.FactorN
	return rN, rD, nil
}

// ConvertirAFloat64 materializa la fracción (n/d) con redondeo a `decimals` lugares.
func ConvertirAFloat64(n, d int64, decimals int) float64 {
	if d == 0 {
		return 0
	}
	factor := math.Pow(10, float64(decimals))
	return math.Round(float64(n)/float64(d)*factor) / factor
}

// ConsumoBaseConMerma calcula la cantidad en unidad base a descontar del stock,
// aplicando la fórmula de merma del negocio:
//
//	Cantidad Consumida = Cantidad Receta × (1 + Porcentaje Merma / 100) × Cantidad Platos
//
// Los parámetros cantN/cantD representan la cantidad de receta como fracción exacta
// (ej: 2 tbsp = 2/1, 1/2 porción = 1/2). factN/factD es la fracción de conversión
// de la unidad de receta a la unidad base del insumo.
func ConsumoBaseConMerma(
	cantidadRecetaN, cantidadRecetaD, factorN, factorD int64,
	porcentajeMerma float64,
	numPlatos int64,
) float64 {
	if factorD == 0 || cantidadRecetaD == 0 {
		return 0
	}

	base := float64(cantidadRecetaN*factorN) / float64(cantidadRecetaD*factorD)
	return base * (1.0 + porcentajeMerma/100.0) * float64(numPlatos)
}