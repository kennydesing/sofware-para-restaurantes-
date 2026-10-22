package services_test

import (
	"context"
	"testing"
	"time"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/services"
)

type mockTasaRepo struct {
	tasas map[string]*domain.TasaCambio
}

func (m *mockTasaRepo) GetTasa(ctx context.Context, moneda string) (*domain.TasaCambio, error) {
	if t, ok := m.tasas[moneda]; ok {
		return t, nil
	}
	return nil, domain.ErrTasaNoEncontrada
}

func (m *mockTasaRepo) GetAllTasas(ctx context.Context) ([]*domain.TasaCambio, error) {
	var list []*domain.TasaCambio
	for _, t := range m.tasas {
		list = append(list, t)
	}
	return list, nil
}

func (m *mockTasaRepo) GuardarTasa(ctx context.Context, tasa *domain.TasaCambio) error {
	m.tasas[tasa.Moneda] = tasa
	return nil
}

type mockBCVProvider struct {
	usd float64
	eur float64
	err error
}

func (m *mockBCVProvider) ObtenerTasasBCV(ctx context.Context) (float64, float64, error) {
	return m.usd, m.eur, m.err
}

func TestTasaCambioService_SincronizarYConsultar(t *testing.T) {
	repo := &mockTasaRepo{tasas: make(map[string]*domain.TasaCambio)}
	provider := &mockBCVProvider{usd: 880.50, eur: 985.20}
	svc := services.NewTasaCambioService(repo, provider)

	ctx := context.Background()

	// Sincronizar
	tasas, err := svc.SincronizarTasasBCV(ctx)
	if err != nil {
		t.Fatalf("Error inesperado en sincronización: %v", err)
	}
	if len(tasas) != 2 {
		t.Errorf("Se esperaban 2 tasas, se obtuvieron %d", len(tasas))
	}

	// Consultar USD
	tasaUSD, err := svc.ObtenerTasa(ctx, "USD")
	if err != nil {
		t.Fatalf("Error al obtener tasa USD: %v", err)
	}
	if tasaUSD.Tasa != 880.50 {
		t.Errorf("Tasa USD esperada 880.50, obtenida %v", tasaUSD.Tasa)
	}

	// Conversión USD a Bs
	bs, err := svc.ConvertirUSDaBs(ctx, 10.0)
	if err != nil {
		t.Fatalf("Error convirtiendo USD a Bs: %v", err)
	}
	if bs != 8805.0 {
		t.Errorf("Esperado 8805.0 Bs, obtenido %v", bs)
	}

	// Actualización manual
	if err := svc.ActualizarTasaManual(ctx, "USD", 900.0); err != nil {
		t.Fatalf("Error en actualización manual: %v", err)
	}

	tasaUSDMod, _ := svc.ObtenerTasa(ctx, "USD")
	if tasaUSDMod.Tasa != 900.0 || tasaUSDMod.Fuente != "MANUAL" {
		t.Errorf("Tasa no actualizada correctamente a MANUAL: %+v", tasaUSDMod)
	}
}

func TestTasaCambioService_Validaciones(t *testing.T) {
	repo := &mockTasaRepo{tasas: make(map[string]*domain.TasaCambio)}
	provider := &mockBCVProvider{usd: 880.50, eur: 985.20}
	svc := services.NewTasaCambioService(repo, provider)
	ctx := context.Background()

	if err := svc.ActualizarTasaManual(ctx, "GBP", 1000); err != domain.ErrMonedaNoSoportada {
		t.Errorf("Esperado ErrMonedaNoSoportada, obtenido %v", err)
	}

	if err := svc.ActualizarTasaManual(ctx, "USD", -5); err != domain.ErrTasaInvalida {
		t.Errorf("Esperado ErrTasaInvalida, obtenido %v", err)
	}
}

var _ = time.Now
