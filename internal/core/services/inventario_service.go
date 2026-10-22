package services

import (
	"context"
	"fmt"
	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// InventarioServiceImpl implementa el puerto de entrada ports.InventarioService.
// Encapsula toda la lógica de negocio de materias primas, costos y auditoría.
type InventarioServiceImpl struct {
	insumoRepo     ports.InsumoRepository
	movimientoRepo ports.MovimientoRepository
	notifier       ports.NotifierRepository
}

// NewInventarioService construye una nueva instancia del servicio de inventario.
func NewInventarioService(
	insumoRepo ports.InsumoRepository,
	movimientoRepo ports.MovimientoRepository,
	notifier ports.NotifierRepository,
) ports.InventarioService {
	return &InventarioServiceImpl{
		insumoRepo:     insumoRepo,
		movimientoRepo: movimientoRepo,
		notifier:       notifier,
	}
}

// RegistrarCompra registra una entrada de insumo por compra a proveedor.
// 1. Si el insumo tiene empaque configurado (ej: Gavera 36u), la cantidad comprada está en empaques
//    y se convierte a unidad base multiplicando por unidades_por_empaque.
// 2. Determina el costo unitario por botella/lata y recalcula el costo promedio ponderado.
// 3. Incrementa el stock persistido y actualiza el costo promedio.
// 4. Registra el movimiento obligatorio en el libro diario (ENTRADA_COMPRA).
func (s *InventarioServiceImpl) RegistrarCompra(ctx context.Context, req ports.CompraRequest) (*domain.Insumo, error) {
	if req.Cantidad <= 0 {
		return nil, domain.ErrCantidadInvalida
	}
	if req.CostoUnitario < 0 {
		return nil, fmt.Errorf("el costo unitario no puede ser negativo")
	}

	// 1. Obtener el insumo actual
	insumo, err := s.insumoRepo.GetByID(ctx, req.InsumoID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener insumo: %w", err)
	}

	// 2. Normalizar la cantidad comprada a la unidad base del insumo
	var cantidadBase float64
	var esEmpaque bool

	// Si el insumo tiene un empaque configurado (distinto de UNITARIO y con unidades > 1),
	// la cantidad comprada está en empaques (cajas, gaveras, packs), no en unidades individuales.
	if insumo.EmpaqueTipo != domain.EmpaqueUnitario && insumo.UnidadesPorEmpaque > 1 {
		cantidadBase = insumo.ConvertirEmpaquesABase(req.Cantidad)
		esEmpaque = true
	} else {
		cantidadBase, err = domain.ConvertirAUnidadBase(req.Cantidad, req.UnidadCompra, insumo.UnidadBase)
		if err != nil {
			return nil, fmt.Errorf("error en conversión de unidades: %w", err)
		}
	}

	// 3. Calcular el costo por unidad base mínima
	// Ejemplo: 2 gaveras a $18/gavera = $36 total. 72 botellas => $36 / 72 = $0.50 por botella.
	costoTotalCompra := req.Cantidad * req.CostoUnitario
	costoUnitarioBase := costoTotalCompra / cantidadBase

	// 4. Calcular el nuevo costo promedio ponderado
	nuevoCostoPromedio := insumo.CalcularNuevoCostoPromedio(cantidadBase, costoUnitarioBase)
	nuevoStock := insumo.Stock + cantidadBase

	// 5. Persistir nuevo stock y costo
	if err := s.insumoRepo.UpdateStockAndCosto(ctx, insumo.ID, nuevoStock, nuevoCostoPromedio); err != nil {
		return nil, fmt.Errorf("error al actualizar stock y costo del insumo: %w", err)
	}

	// Actualizar entidad en memoria
	insumo.Stock = nuevoStock
	insumo.CostoPromedio = nuevoCostoPromedio

	// 6. Registrar en el libro diario de auditoría
	ref := req.Referencia
	if ref == "" {
		if esEmpaque {
			ref = fmt.Sprintf("Compra %.0f %s (%d u) a $%.4f/empaque", req.Cantidad, insumo.EmpaqueNombre, insumo.UnidadesPorEmpaque, req.CostoUnitario)
		} else {
			ref = fmt.Sprintf("Compra de %.2f %s a $%.4f/%s", req.Cantidad, req.UnidadCompra, req.CostoUnitario, req.UnidadCompra)
		}
	}

	mov := &domain.MovimientoInventario{
		InsumoID:      insumo.ID,
		Tipo:          domain.EntradaCompra,
		Cantidad:      cantidadBase,
		CostoUnitario: costoUnitarioBase,
		Referencia:    ref,
	}

	if err := s.movimientoRepo.Create(ctx, mov); err != nil {
		return nil, fmt.Errorf("error al registrar movimiento de entrada: %w", err)
	}

	return insumo, nil
}

// RegistrarMerma descuenta stock derivado de pérdidas operativas extraordinarias.
func (s *InventarioServiceImpl) RegistrarMerma(ctx context.Context, req ports.MermaRequest) (*domain.MovimientoInventario, error) {
	if req.Cantidad <= 0 {
		return nil, domain.ErrCantidadInvalida
	}

	insumo, err := s.insumoRepo.GetByID(ctx, req.InsumoID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener insumo: %w", err)
	}

	// Normalizar unidad de merma si fue provista, o asumir unidad base
	unidadMerma := req.UnidadMedida
	if unidadMerma == "" {
		unidadMerma = insumo.UnidadBase
	}

	cantidadBase, err := domain.ConvertirAUnidadBase(req.Cantidad, unidadMerma, insumo.UnidadBase)
	if err != nil {
		return nil, fmt.Errorf("error en conversión de unidad de merma: %w", err)
	}

	if insumo.Stock < cantidadBase {
		return nil, domain.ErrStockInsuficiente
	}

	// Descontar stock
	nuevoStock := insumo.Stock - cantidadBase
	if err := s.insumoRepo.UpdateStockAndCosto(ctx, insumo.ID, nuevoStock, insumo.CostoPromedio); err != nil {
		return nil, fmt.Errorf("error al descontar stock por merma: %w", err)
	}

	insumo.Stock = nuevoStock

	// Registrar movimiento de merma
	motivo := req.Motivo
	if motivo == "" {
		motivo = "Merma operativa sin motivo especificado"
	}

	mov := &domain.MovimientoInventario{
		InsumoID:      insumo.ID,
		Tipo:          domain.SalidaMerma,
		Cantidad:      cantidadBase,
		CostoUnitario: insumo.CostoPromedio,
		Referencia:    motivo,
	}

	if err := s.movimientoRepo.Create(ctx, mov); err != nil {
		return nil, fmt.Errorf("error al auditar movimiento de merma: %w", err)
	}

	// Notificar alerta si cae bajo el mínimo configurado
	if insumo.EstaBajoMinimo() && s.notifier != nil {
		_ = s.notifier.NotificarStockBajo(ctx, insumo)
	}

	return mov, nil
}

// AjustarStock actualiza el stock físico del insumo tras una auditoría presencial.
func (s *InventarioServiceImpl) AjustarStock(ctx context.Context, req ports.AjusteRequest) (*domain.MovimientoInventario, error) {
	insumo, err := s.insumoRepo.GetByID(ctx, req.InsumoID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener insumo: %w", err)
	}

	diferencia := req.StockFisico - insumo.Stock

	if err := s.insumoRepo.UpdateStockAndCosto(ctx, insumo.ID, req.StockFisico, insumo.CostoPromedio); err != nil {
		return nil, fmt.Errorf("error al aplicar ajuste de stock: %w", err)
	}

	insumo.Stock = req.StockFisico

	ref := req.Motivo
	if ref == "" {
		ref = fmt.Sprintf("Ajuste de inventario (Diferencia: %.2f %s)", diferencia, insumo.UnidadBase)
	}

	mov := &domain.MovimientoInventario{
		InsumoID:      insumo.ID,
		Tipo:          domain.AjusteManual,
		Cantidad:      diferencia,
		CostoUnitario: insumo.CostoPromedio,
		Referencia:    ref,
	}

	if err := s.movimientoRepo.Create(ctx, mov); err != nil {
		return nil, fmt.Errorf("error al auditar ajuste de inventario: %w", err)
	}

	if insumo.EstaBajoMinimo() && s.notifier != nil {
		_ = s.notifier.NotificarStockBajo(ctx, insumo)
	}

	return mov, nil
}

// ObtenerInsumo devuelve el insumo solicitado por su identificador.
func (s *InventarioServiceImpl) ObtenerInsumo(ctx context.Context, id int) (*domain.Insumo, error) {
	return s.insumoRepo.GetByID(ctx, id)
}

// RegistrarUsoPersonal descuenta stock por alimentación del personal del restaurante.
func (s *InventarioServiceImpl) RegistrarUsoPersonal(ctx context.Context, req ports.UsoPersonalRequest) (*domain.MovimientoInventario, error) {
	if req.Cantidad <= 0 {
		return nil, domain.ErrCantidadInvalida
	}

	insumo, err := s.insumoRepo.GetByID(ctx, req.InsumoID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener insumo: %w", err)
	}

	unidad := req.UnidadMedida
	if unidad == "" {
		unidad = insumo.UnidadBase
	}

	cantidadBase, err := domain.ConvertirAUnidadBase(req.Cantidad, unidad, insumo.UnidadBase)
	if err != nil {
		return nil, fmt.Errorf("error en conversión de unidad de uso personal: %w", err)
	}

	if insumo.Stock < cantidadBase {
		return nil, domain.ErrStockInsuficiente
	}

	nuevoStock := insumo.Stock - cantidadBase
	if err := s.insumoRepo.UpdateStockAndCosto(ctx, insumo.ID, nuevoStock, insumo.CostoPromedio); err != nil {
		return nil, fmt.Errorf("error al descontar stock por uso personal: %w", err)
	}
	insumo.Stock = nuevoStock

	detalle := req.Detalle
	if detalle == "" {
		detalle = "Consumo de personal de cocina/sala"
	}

	mov := &domain.MovimientoInventario{
		InsumoID:      insumo.ID,
		Tipo:          domain.SalidaMerma,
		Cantidad:      cantidadBase,
		CostoUnitario: insumo.CostoPromedio,
		Referencia:    fmt.Sprintf("USO_PERSONAL: %s", detalle),
	}

	if err := s.movimientoRepo.Create(ctx, mov); err != nil {
		return nil, fmt.Errorf("error al auditar movimiento de uso personal: %w", err)
	}

	if insumo.EstaBajoMinimo() && s.notifier != nil {
		_ = s.notifier.NotificarStockBajo(ctx, insumo)
	}

	return mov, nil
}

// ListarHistorialMermas devuelve el historial de pérdidas operativas registradas.
func (s *InventarioServiceImpl) ListarHistorialMermas(ctx context.Context, limite int) ([]*ports.HistorialSalidaDTO, error) {
	return s.movimientoRepo.ListarHistorialMermas(ctx, limite)
}

// ListarHistorialUsoPersonal devuelve el historial de consumo de insumos por el personal.
func (s *InventarioServiceImpl) ListarHistorialUsoPersonal(ctx context.Context, limite int) ([]*ports.HistorialSalidaDTO, error) {
	return s.movimientoRepo.ListarHistorialUsoPersonal(ctx, limite)
}

// ListarInsumos obtiene la totalidad de materias primas registradas.
func (s *InventarioServiceImpl) ListarInsumos(ctx context.Context) ([]*domain.Insumo, error) {
	return s.insumoRepo.GetAll(ctx)
}

// AuditarStock compara el stock teórico del sistema contra el conteo físico real sin modificar aún el stock.
func (s *InventarioServiceImpl) AuditarStock(ctx context.Context, insumoID int, conteoFisico float64) (*ports.AuditoriaStockDTO, error) {
	insumo, err := s.insumoRepo.GetByID(ctx, insumoID)
	if err != nil {
		return nil, err
	}

	diferencia := conteoFisico - insumo.Stock
	descuadreValorizado := diferencia * insumo.CostoPromedio

	return &ports.AuditoriaStockDTO{
		InsumoID:            insumo.ID,
		NombreInsumo:        insumo.Nombre,
		UnidadBase:          insumo.UnidadBase,
		StockTeorico:        insumo.Stock,
		StockFisico:         conteoFisico,
		Diferencia:          diferencia,
		DescuadreValorizado: descuadreValorizado,
	}, nil
}