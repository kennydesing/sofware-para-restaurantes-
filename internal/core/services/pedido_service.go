package services

import (
	"context"
	"fmt"
	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// PedidoServiceImpl implementa el puerto de entrada ports.PedidoService.
// Orquesta la creación de órdenes y el débito atómico por escandallo (ficha técnica).
type PedidoServiceImpl struct {
	pedidoRepo    ports.PedidoRepository
	platoRepo     ports.PlatoRepository
	recetaRepo    ports.RecetaRepository
	insumoRepo    ports.InsumoRepository
	atomicDebitor ports.AtomicDebitor
	printer       ports.PrinterRepository
	notifier      ports.NotifierRepository
	conversionSvc ports.ConversionService
}

// NewPedidoService construye una nueva instancia del servicio de pedidos.
func NewPedidoService(
	pedidoRepo ports.PedidoRepository,
	platoRepo ports.PlatoRepository,
	recetaRepo ports.RecetaRepository,
	insumoRepo ports.InsumoRepository,
	atomicDebitor ports.AtomicDebitor,
	printer ports.PrinterRepository,
	notifier ports.NotifierRepository,
	conversionSvc ports.ConversionService,
) ports.PedidoService {
	return &PedidoServiceImpl{
		pedidoRepo:    pedidoRepo,
		platoRepo:     platoRepo,
		recetaRepo:    recetaRepo,
		insumoRepo:    insumoRepo,
		atomicDebitor: atomicDebitor,
		printer:       printer,
		notifier:      notifier,
		conversionSvc: conversionSvc,
	}
}

// CrearPedido apertura una comanda en estado PENDIENTE.
func (s *PedidoServiceImpl) CrearPedido(ctx context.Context, req ports.CrearPedidoRequest) (*domain.Pedido, error) {
	if req.MesaID <= 0 {
		return nil, fmt.Errorf("id de mesa inválido")
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("el pedido debe contener al menos un plato")
	}

	detalles := make([]domain.DetallePedido, 0, len(req.Items))

	for _, item := range req.Items {
		if item.Cantidad <= 0 {
			return nil, domain.ErrCantidadInvalida
		}

		plato, err := s.platoRepo.GetByID(ctx, item.PlatoID)
		if err != nil {
			return nil, fmt.Errorf("error al obtener plato ID %d: %w", item.PlatoID, err)
		}
		if !plato.Activo {
			return nil, fmt.Errorf("el plato '%s' no está activo para la venta", plato.Nombre)
		}

		detalles = append(detalles, domain.DetallePedido{
			PlatoID:        plato.ID,
			NombrePlato:    plato.Nombre,
			Cantidad:       item.Cantidad,
			PrecioUnitario: plato.PrecioVenta,
			Subtotal:       float64(item.Cantidad) * plato.PrecioVenta,
		})
	}

	pedido := &domain.Pedido{
		MesaID:   req.MesaID,
		Estado:   domain.EstadoPendiente,
		Detalles: detalles,
	}
	pedido.CalcularTotal()

	if err := s.pedidoRepo.Create(ctx, pedido); err != nil {
		return nil, fmt.Errorf("error al persistir pedido: %w", err)
	}

	return pedido, nil
}

// ProcesarPedido ejecuta el caso de uso central del restaurante:
// 1. Obtiene el pedido y valida que esté en estado PENDIENTE.
// 2. Desglosa cada plato en su receta/escandallo y calcula el consumo neto + merma en gramos/ml:
//    Cantidad Consumida = Cantidad Requerida * (1 + Porcentaje Merma / 100) * Cantidad Platos
// 3. Agrupa los consumos requeridos por cada insumo_id.
// 4. Delega al AtomicDebitor la ejecución de la transacción SQL atómica:
//    - Bloqueo y verificación de stock de cada ingrediente.
//    - Débito de stock en la tabla insumos.
//    - Inserción de movimientos de auditoría (SALIDA_VENTA).
//    - Actualización del estado del pedido a EN_COCINA.
// 5. Si la transacción confirma (Commit), envía la comanda a la impresora de cocina
//    y notifica si algún ingrediente quedó en nivel crítico.
func (s *PedidoServiceImpl) ProcesarPedido(ctx context.Context, pedidoID int) error {
	// 1. Obtener pedido con detalles
	pedido, err := s.pedidoRepo.GetByID(ctx, pedidoID)
	if err != nil {
		return fmt.Errorf("error al obtener pedido %d: %w", pedidoID, err)
	}

	if !pedido.PuedeSerProcesado() {
		return fmt.Errorf("%w: el pedido #%d está en estado '%s'", domain.ErrEstadoPedidoInvalido, pedido.ID, pedido.Estado)
	}

	// 2. Calcular consumos consolidados por insumo aplicando la fórmula de escandallo y merma
	consumosMap := make(map[int]*ports.ConsumoInsumoDTO)

	for _, detalle := range pedido.Detalles {
		recetas, err := s.recetaRepo.GetByPlatoID(ctx, detalle.PlatoID)
		if err != nil {
			return fmt.Errorf("error al consultar receta del plato #%d (%s): %w", detalle.PlatoID, detalle.NombrePlato, err)
		}

		if len(recetas) == 0 {
			return fmt.Errorf("%w para el plato '%s' (ID %d)", domain.ErrRecetaVacia, detalle.NombrePlato, detalle.PlatoID)
		}

		for _, item := range recetas {
			var consumoTotal float64

			if item.UnidadRecetaID != 0 && item.UnidadRecetaAbrev != "" && s.conversionSvc != nil {
				// Ruta nueva: convertir cantidad de receta a unidad base del insumo usando ConversionService
				consumoTotal, err = s.conversionSvc.ConsumoReceta(
					ctx,
					item.CantidadRequerida,
					item.UnidadRecetaAbrev,
					int64(item.InsumoID),
					item.PorcentajeMerma,
					detalle.Cantidad,
				)
				if err != nil {
					return fmt.Errorf("error al calcular consumo de receta para insumo #%d: %w", item.InsumoID, err)
				}
			} else {
				// Ruta legacy: cantidad ya en unidad base del insumo
				consumoTotal = item.CalcularConsumoTotal(detalle.Cantidad)
			}

			if c, existe := consumosMap[item.InsumoID]; existe {
				c.Cantidad += consumoTotal
			} else {
				insumo, err := s.insumoRepo.GetByID(ctx, item.InsumoID)
				if err != nil {
					return fmt.Errorf("error al consultar insumo #%d: %w", item.InsumoID, err)
				}

				consumosMap[item.InsumoID] = &ports.ConsumoInsumoDTO{
					InsumoID:      item.InsumoID,
					NombreInsumo:  insumo.Nombre,
					Cantidad:      consumoTotal,
					CostoUnitario: insumo.CostoPromedio,
					Referencia:    fmt.Sprintf("Venta Pedido #%d (Mesa %d)", pedido.ID, pedido.MesaID),
				}
			}
		}
	}

	// Convertir mapa a lista
	consumosList := make([]ports.ConsumoInsumoDTO, 0, len(consumosMap))
	for _, c := range consumosMap {
		consumosList = append(consumosList, *c)
	}

	// 3. Ejecución atómica en SQLite con transacción y rollback automático en caso de falta de stock
	if err := s.atomicDebitor.DebitarStockPedidoAtomic(ctx, pedido.ID, consumosList); err != nil {
		return fmt.Errorf("débito atómico fallido: %w", err)
	}

	// Actualizar estado local
	pedido.Estado = domain.EstadoEnCocina

	// 4. Imprimir comanda en cocina
	if s.printer != nil {
		_ = s.printer.ImprimirComanda(ctx, pedido)
	}

	// 5. Verificar si algún insumo quedó bajo su stock mínimo para alertar
	if s.notifier != nil {
		for _, c := range consumosList {
			if insumoActualizado, err := s.insumoRepo.GetByID(ctx, c.InsumoID); err == nil {
				if insumoActualizado.EstaBajoMinimo() {
					_ = s.notifier.NotificarStockBajo(ctx, insumoActualizado)
				}
			}
		}
	}

	return nil
}

// ObtenerPedido consulta una orden de venta por su ID.
func (s *PedidoServiceImpl) ObtenerPedido(ctx context.Context, id int) (*domain.Pedido, error) {
	return s.pedidoRepo.GetByID(ctx, id)
}

// ActualizarEstadoPedido cambia el estado de un pedido.
func (s *PedidoServiceImpl) ActualizarEstadoPedido(ctx context.Context, pedidoID int, estado string) error {
	_, err := s.pedidoRepo.GetByID(ctx, pedidoID)
	if err != nil {
		return fmt.Errorf("error al obtener pedido %d: %w", pedidoID, err)
	}

	nuevoEstado := domain.EstadoPedido(estado)
	estadosValidos := map[domain.EstadoPedido]bool{
		domain.EstadoPendiente: true,
		domain.EstadoEnCocina:  true,
		domain.EstadoListo:     true,
		domain.EstadoEntregado: true,
		domain.EstadoCancelado: true,
	}
	if !estadosValidos[nuevoEstado] {
		return fmt.Errorf("estado inválido: %s", estado)
	}

	return s.pedidoRepo.UpdateEstado(ctx, pedidoID, nuevoEstado)
}

// ListarPedidos devuelve todos los pedidos con sus items para KDS.
func (s *PedidoServiceImpl) ListarPedidos(ctx context.Context) ([]*ports.PedidoWithItems, error) {
	pedidos, err := s.pedidoRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*ports.PedidoWithItems, 0, len(pedidos))
	for _, p := range pedidos {
		items := make([]ports.ItemPedidoConPlato, len(p.Detalles))
		for i, d := range p.Detalles {
			items[i] = ports.ItemPedidoConPlato{
				PlatoID:  d.PlatoID,
				Nombre:   d.NombrePlato,
				Cantidad: d.Cantidad,
				Precio:   d.PrecioUnitario,
			}
		}
		result = append(result, &ports.PedidoWithItems{
			ID:        p.ID,
			MesaID:    p.MesaID,
			Estado:    string(p.Estado),
			Items:     items,
			Total:     p.Total,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05"),
		})
	}
	return result, nil
}
