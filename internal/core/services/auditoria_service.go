package services

import (
	"context"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

type auditoriaService struct {
	auditoriaRepo ports.AuditoriaRepository
}

func NewAuditoriaService(auditoriaRepo ports.AuditoriaRepository) ports.AuditoriaService {
	return &auditoriaService{auditoriaRepo: auditoriaRepo}
}

var _ ports.AuditoriaService = (*auditoriaService)(nil)

func (s *auditoriaService) RegistrarEvento(ctx context.Context, evento *domain.EventoAuditoria) error {
	return s.auditoriaRepo.Create(ctx, evento)
}

func (s *auditoriaService) ListarEventos(ctx context.Context, filtro ports.FiltroAuditoria) ([]*domain.EventoAuditoria, error) {
	return s.auditoriaRepo.GetFiltered(ctx, filtro)
}