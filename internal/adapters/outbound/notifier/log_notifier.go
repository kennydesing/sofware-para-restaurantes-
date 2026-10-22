package notifier

import (
	"context"
	"fmt"
	"log"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// LogNotifierAdapter implementa ports.NotifierRepository registrando alertas operativas en log/pantalla.
type LogNotifierAdapter struct{}

// NewLogNotifierAdapter crea una nueva instancia del notificador de eventos.
func NewLogNotifierAdapter() ports.NotifierRepository {
	return &LogNotifierAdapter{}
}

// NotificarStockBajo emite una advertencia cuando un insumo alcanza su stock crítico.
func (n *LogNotifierAdapter) NotificarStockBajo(ctx context.Context, insumo *domain.Insumo) error {
	log.Printf("[ALERTA DE INVENTARIO] Insumo '%s' (ID %d) ha caído a nivel crítico: Stock Actual = %.2f %s (Mínimo = %.2f %s)\n",
		insumo.Nombre,
		insumo.ID,
		insumo.Stock,
		insumo.UnidadBase,
		insumo.StockMinimo,
		insumo.UnidadBase,
	)
	fmt.Printf("⚠️  [ALERTA STOCK MÍNIMO] %s: %.2f %s restantes (Umbral: %.2f %s)\n",
		insumo.Nombre, insumo.Stock, insumo.UnidadBase, insumo.StockMinimo, insumo.UnidadBase)
	return nil
}

// NotificarCambioEstado registra el cambio de estado de un pedido.
func (n *LogNotifierAdapter) NotificarCambioEstado(ctx context.Context, pedidoID int, estado string) error {
	log.Printf("[CAMBIO ESTADO PEDIDO] Pedido #%d -> %s", pedidoID, estado)
	fmt.Printf("🔄 [ESTADO PEDIDO] #%d: %s\n", pedidoID, estado)
	return nil
}
