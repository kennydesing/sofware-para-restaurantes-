package domain_test

import (
	"testing"

	"pos-inventory/internal/core/domain"
)

func TestConvertirUSDaBs(t *testing.T) {
	tests := []struct {
		name     string
		montoUSD float64
		tasaUSD  float64
		esperado float64
	}{
		{"Monto estándar", 10.0, 877.90, 8779.0},
		{"Monto fraccionado", 5.5, 877.9023, 4828.46},
		{"Tasa cero retorna cero", 10.0, 0, 0},
		{"Monto cero retorna cero", 0, 877.90, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.ConvertirUSDaBs(tt.montoUSD, tt.tasaUSD)
			if got != tt.esperado {
				t.Errorf("ConvertirUSDaBs(%v, %v) = %v; esperado %v", tt.montoUSD, tt.tasaUSD, got, tt.esperado)
			}
		})
	}
}

func TestConvertirBsaUSD(t *testing.T) {
	tests := []struct {
		name     string
		montoBs  float64
		tasaUSD  float64
		esperado float64
	}{
		{"Monto estándar", 8779.0, 877.90, 10.0},
		{"Tasa cero retorna cero", 100.0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.ConvertirBsaUSD(tt.montoBs, tt.tasaUSD)
			if got != tt.esperado {
				t.Errorf("ConvertirBsaUSD(%v, %v) = %v; esperado %v", tt.montoBs, tt.tasaUSD, got, tt.esperado)
			}
		})
	}
}
