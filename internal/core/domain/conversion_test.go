package domain

import (
	"math"
	"testing"
)

func TestConvertirEntreMismaCategoria_KgAGramos(t *testing.T) {
	origen := UnidadMedida{Abreviatura: "kg", Categoria: CategoriaPeso, FactorN: 1000, FactorD: 1}
	destino := UnidadMedida{Abreviatura: "g", Categoria: CategoriaPeso, FactorN: 1, FactorD: 1}

	rN, rD, err := ConvertirEntreMismaCategoria(1, 1, origen, destino)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	resultado := float64(rN) / float64(rD)
	if resultado != 1000.0 {
		t.Fatalf("esperado 1000 g, obtenido %v", resultado)
	}
}

func TestConvertirEntreMismaCategoria_LibraAGramos(t *testing.T) {
	origen := UnidadMedida{Abreviatura: "lb", Categoria: CategoriaPeso, FactorN: 45359237, FactorD: 100000}
	destino := UnidadMedida{Abreviatura: "g", Categoria: CategoriaPeso, FactorN: 1, FactorD: 1}

	rN, rD, err := ConvertirEntreMismaCategoria(1, 1, origen, destino)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	resultado := ConvertirAFloat64(rN, rD, 4)
	esperado := 453.5924 // 453.59237 redondeado a 4 decimales
	if math.Abs(resultado-esperado) > 0.0001 {
		t.Fatalf("esperado ~%v g, obtenido %v", esperado, resultado)
	}
}

func TestConvertirEntreMismaCategoria_GalonAMl(t *testing.T) {
	origen := UnidadMedida{Abreviatura: "gal", Categoria: CategoriaVolumen, FactorN: 3785411784, FactorD: 1000000}
	destino := UnidadMedida{Abreviatura: "ml", Categoria: CategoriaVolumen, FactorN: 1, FactorD: 1}

	rN, rD, err := ConvertirEntreMismaCategoria(1, 1, origen, destino)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	resultado := ConvertirAFloat64(rN, rD, 4)
	esperado := 3785.4118 // 3785.411784 redondeado a 4 decimales
	if math.Abs(resultado-esperado) > 0.0001 {
		t.Fatalf("esperado ~%v ml, obtenido %v", esperado, resultado)
	}
}

func TestConvertirEntreMismaCategoria_CategoriaIncompatible(t *testing.T) {
	origen := UnidadMedida{Abreviatura: "kg", Categoria: CategoriaPeso, FactorN: 1000, FactorD: 1}
	destino := UnidadMedida{Abreviatura: "ml", Categoria: CategoriaVolumen, FactorN: 1, FactorD: 1}

	_, _, err := ConvertirEntreMismaCategoria(1, 1, origen, destino)
	if err == nil {
		t.Fatal("esperaba error ErrCategoriaIncompatible, no se devolvió error")
	}
	if err != ErrCategoriaIncompatible {
		t.Fatalf("esperaba ErrCategoriaIncompatible, obtenido %v", err)
	}
}

func TestConvertirAFloat64_Redondeo4Decimales(t *testing.T) {
	resultado := ConvertirAFloat64(1, 3, 4)
	esperado := 0.3333
	if math.Abs(resultado-esperado) > 0.00005 {
		t.Fatalf("esperado %v, obtenido %v", esperado, resultado)
	}
}

func TestConvertirAFloat64_DenominadorCero(t *testing.T) {
	resultado := ConvertirAFloat64(10, 0, 4)
	if resultado != 0 {
		t.Fatalf("esperado 0 para denominador cero, obtenido %v", resultado)
	}
}

func TestConsumoBaseConMerma_ConMerma5Pct(t *testing.T) {
	// 200 g × (1 + 5/100) × 1 plato = 210.0
	resultado := ConsumoBaseConMerma(200, 1, 1, 1, 5.0, 1)
	esperado := 210.0
	if math.Abs(resultado-esperado) > 0.0001 {
		t.Fatalf("esperado %v, obtenido %v", esperado, resultado)
	}
}

func TestConsumoBaseConMerma_SinMerma(t *testing.T) {
	// 500 ml × 1.0 × 2 platos = 1000.0
	resultado := ConsumoBaseConMerma(500, 1, 1, 1, 0.0, 2)
	esperado := 1000.0
	if math.Abs(resultado-esperado) > 0.0001 {
		t.Fatalf("esperado %v, obtenido %v", esperado, resultado)
	}
}

func TestConsumoBaseConMerma_DenominadorCero(t *testing.T) {
	resultado := ConsumoBaseConMerma(100, 0, 1, 1, 10.0, 1)
	if resultado != 0 {
		t.Fatalf("esperado 0 para denominador cero, obtenido %v", resultado)
	}
}

func TestConsumoBaseConMerma_FactorDCero(t *testing.T) {
	resultado := ConsumoBaseConMerma(100, 1, 1, 0, 10.0, 1)
	if resultado != 0 {
		t.Fatalf("esperado 0 para factorD cero, obtenido %v", resultado)
	}
}
