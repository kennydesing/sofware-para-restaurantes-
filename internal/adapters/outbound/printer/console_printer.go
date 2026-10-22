package printer

import (
	"context"
	"fmt"
	"strings"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// ConsolePrinterAdapter implementa ports.PrinterRepository para enviar la comanda a consola / driver térmico.
type ConsolePrinterAdapter struct{}

// NewConsolePrinterAdapter crea una nueva instancia del adaptador de impresora.
func NewConsolePrinterAdapter() ports.PrinterRepository {
	return &ConsolePrinterAdapter{}
}

// ImprimirComanda formatea y envía la comanda de cocina.
func (p *ConsolePrinterAdapter) ImprimirComanda(ctx context.Context, pedido *domain.Pedido) error {
	var sb strings.Builder
	sb.WriteString("\n========================================\n")
	sb.WriteString("        TICKET DE COMANDA (COCINA)      \n")
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf(" PEDIDO #%-5d | MESA: #%-3d\n", pedido.ID, pedido.MesaID))
	sb.WriteString(fmt.Sprintf(" FECHA: %s\n", pedido.CreatedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString("----------------------------------------\n")
	sb.WriteString(" CANT  DESCRIPCIÓN                      \n")
	sb.WriteString("----------------------------------------\n")
	for _, d := range pedido.Detalles {
		sb.WriteString(fmt.Sprintf(" [%2dx]  %-30s\n", d.Cantidad, d.NombrePlato))
	}
	sb.WriteString("========================================\n")
	sb.WriteString(" ESTADO: ENVIADO A COCINA              \n")
	sb.WriteString("========================================\n\n")

	fmt.Print(sb.String())
	return nil
}
