package domain_test

import (
	"math"
	"testing"

	"pos-inventory/internal/core/domain"
)

func TestCalcularConsumoTotalConMerma(t *testing.T) {
	receta := &domain.RecetaItem{
		InsumoID:          1,
		CantidadRequerida: 200.0, // 200g
		PorcentajeMerma:   10.0,  // 10%
	}

	consumoUnitario := receta.CalcularCantidadConsumidaUnitaria()
	if math.Abs(consumoUnitario-220.0) > 1e-6 {
		t.Errorf("Se esperaba consumo unitario de 220.0g, pero se obtuvo: %f", consumoUnitario)
	}

	consumoTotal := receta.CalcularConsumoTotal(3)
	if math.Abs(consumoTotal-660.0) > 1e-6 {
		t.Errorf("Se esperaba consumo total para 3 platos de 660.0g, pero se obtuvo: %f", consumoTotal)
	}
}

func TestConvertirAUnidadBase(t *testing.T) {
	g, err := domain.ConvertirAUnidadBase(5.5, "kg", "g")
	if err != nil || math.Abs(g-5500.0) > 1e-6 {
		t.Errorf("Error en conversión kg -> g: esperado 5500, obtenido %f (err: %v)", g, err)
	}

	ml, err := domain.ConvertirAUnidadBase(2.5, "L", "ml")
	if err != nil || math.Abs(ml-2500.0) > 1e-6 {
		t.Errorf("Error en conversión L -> ml: esperado 2500, obtenido %f (err: %v)", ml, err)
	}

	u, err := domain.ConvertirAUnidadBase(2.0, "docena", "u")
	if err != nil || math.Abs(u-24.0) > 1e-6 {
		t.Errorf("Error en conversión docena -> u: esperado 24, obtenido %f (err: %v)", u, err)
	}
}
