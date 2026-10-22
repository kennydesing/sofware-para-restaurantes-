package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

type TasaCambioService struct {
	tasaRepo    ports.TasaCambioRepository
	bcvProvider ports.BCVRateProvider
}

func NewTasaCambioService(
	tasaRepo ports.TasaCambioRepository,
	bcvProvider ports.BCVRateProvider,
) ports.TasaCambioService {
	return &TasaCambioService{
		tasaRepo:    tasaRepo,
		bcvProvider: bcvProvider,
	}
}

var _ ports.TasaCambioService = (*TasaCambioService)(nil)

func (s *TasaCambioService) ObtenerTasas(ctx context.Context) ([]*domain.TasaCambio, error) {
	return s.tasaRepo.GetAllTasas(ctx)
}

func (s *TasaCambioService) ObtenerTasa(ctx context.Context, moneda string) (*domain.TasaCambio, error) {
	monedaNorm := strings.ToUpper(strings.TrimSpace(moneda))
	return s.tasaRepo.GetTasa(ctx, monedaNorm)
}

func (s *TasaCambioService) SincronizarTasasBCV(ctx context.Context) ([]*domain.TasaCambio, error) {
	usd, eur, err := s.bcvProvider.ObtenerTasasBCV(ctx)
	if err != nil {
		return nil, fmt.Errorf("fallo al sincronizar tasas oficiales con BCV: %w", err)
	}

	now := time.Now()
	tasaUSD := &domain.TasaCambio{
		Moneda:             "USD",
		Tasa:               usd,
		Fuente:             "BCV_AUTO",
		FechaActualizacion: now,
	}
	if err := s.tasaRepo.GuardarTasa(ctx, tasaUSD); err != nil {
		return nil, fmt.Errorf("error guardando tasa USD en base de datos: %w", err)
	}

	tasaEUR := &domain.TasaCambio{
		Moneda:             "EUR",
		Tasa:               eur,
		Fuente:             "BCV_AUTO",
		FechaActualizacion: now,
	}
	if err := s.tasaRepo.GuardarTasa(ctx, tasaEUR); err != nil {
		return nil, fmt.Errorf("error guardando tasa EUR en base de datos: %w", err)
	}

	return s.tasaRepo.GetAllTasas(ctx)
}

func (s *TasaCambioService) ActualizarTasaManual(ctx context.Context, moneda string, nuevaTasa float64) error {
	monedaNorm := strings.ToUpper(strings.TrimSpace(moneda))
	if monedaNorm != "USD" && monedaNorm != "EUR" {
		return domain.ErrMonedaNoSoportada
	}
	if nuevaTasa <= 0 {
		return domain.ErrTasaInvalida
	}

	tasa := &domain.TasaCambio{
		Moneda:             monedaNorm,
		Tasa:               nuevaTasa,
		Fuente:             "MANUAL",
		FechaActualizacion: time.Now(),
	}
	return s.tasaRepo.GuardarTasa(ctx, tasa)
}

func (s *TasaCambioService) ConvertirUSDaBs(ctx context.Context, montoUSD float64) (float64, error) {
	tasa, err := s.tasaRepo.GetTasa(ctx, "USD")
	if err != nil {
		return 0, fmt.Errorf("no se pudo obtener la tasa USD para conversión: %w", err)
	}
	return domain.ConvertirUSDaBs(montoUSD, tasa.Tasa), nil
}
