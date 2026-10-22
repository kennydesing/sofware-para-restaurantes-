package services_test

import (
	"context"
	"testing"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
	"pos-inventory/internal/core/services"
)

// mockInsumoRepo implementa ports.InsumoRepository en memoria para pruebas.
type mockInsumoRepo struct {
	insumos []*domain.Insumo
	lastID  int
}

func (m *mockInsumoRepo) GetByID(ctx context.Context, id int) (*domain.Insumo, error) {
	for _, ins := range m.insumos {
		if ins.ID == id {
			copyIns := *ins
			return &copyIns, nil
		}
	}
	return nil, domain.ErrInsumoNoEncontrado
}

func (m *mockInsumoRepo) GetByNombre(ctx context.Context, nombre string) (*domain.Insumo, error) {
	return nil, domain.ErrInsumoNoEncontrado
}

func (m *mockInsumoRepo) GetAll(ctx context.Context) ([]*domain.Insumo, error) {
	return m.insumos, nil
}

func (m *mockInsumoRepo) GetAllEnriquecido(ctx context.Context) ([]*ports.InsumoEnriquecido, error) {
	return nil, nil
}

func (m *mockInsumoRepo) Create(ctx context.Context, insumo *domain.Insumo) error {
	m.lastID++
	insumo.ID = m.lastID
	copyIns := *insumo
	m.insumos = append(m.insumos, &copyIns)
	return nil
}

func (m *mockInsumoRepo) Update(ctx context.Context, insumo *domain.Insumo) error {
	for i, ins := range m.insumos {
		if ins.ID == insumo.ID {
			m.insumos[i] = insumo
			return nil
		}
	}
	return domain.ErrInsumoNoEncontrado
}

func (m *mockInsumoRepo) Delete(ctx context.Context, id int) error {
	return nil
}

func (m *mockInsumoRepo) UpdateStockAndCosto(ctx context.Context, id int, nuevoStock float64, nuevoCostoPromedio float64) error {
	for _, ins := range m.insumos {
		if ins.ID == id {
			ins.Stock = nuevoStock
			ins.CostoPromedio = nuevoCostoPromedio
			return nil
		}
	}
	return domain.ErrInsumoNoEncontrado
}

// mockMovimientoRepo implementa ports.MovimientoRepository en memoria para pruebas.
type mockMovimientoRepo struct {
	movimientos []*domain.MovimientoInventario
	lastID      int
}

func (m *mockMovimientoRepo) Create(ctx context.Context, mov *domain.MovimientoInventario) error {
	m.lastID++
	mov.ID = m.lastID
	copyMov := *mov
	m.movimientos = append(m.movimientos, &copyMov)
	return nil
}

func (m *mockMovimientoRepo) GetByInsumoID(ctx context.Context, insumoID int) ([]*domain.MovimientoInventario, error) {
	var result []*domain.MovimientoInventario
	for _, mov := range m.movimientos {
		if mov.InsumoID == insumoID {
			copyMov := *mov
			result = append(result, &copyMov)
		}
	}
	return result, nil
}

func (m *mockMovimientoRepo) ListarHistorialMermas(ctx context.Context, limite int) ([]*ports.HistorialSalidaDTO, error) {
	var result []*ports.HistorialSalidaDTO
	for _, mov := range m.movimientos {
		if mov.Tipo == domain.SalidaMerma && !esUsoPersonal(mov.Referencia) {
			result = append(result, &ports.HistorialSalidaDTO{
				ID:         mov.ID,
				InsumoID:   mov.InsumoID,
				Cantidad:   mov.Cantidad,
				CostoTotal: mov.Cantidad * mov.CostoUnitario,
				Referencia: mov.Referencia,
			})
		}
	}
	return result, nil
}

func (m *mockMovimientoRepo) ListarHistorialUsoPersonal(ctx context.Context, limite int) ([]*ports.HistorialSalidaDTO, error) {
	var result []*ports.HistorialSalidaDTO
	for _, mov := range m.movimientos {
		if mov.Tipo == domain.SalidaMerma && esUsoPersonal(mov.Referencia) {
			result = append(result, &ports.HistorialSalidaDTO{
				ID:         mov.ID,
				InsumoID:   mov.InsumoID,
				Cantidad:   mov.Cantidad,
				CostoTotal: mov.Cantidad * mov.CostoUnitario,
				Referencia: mov.Referencia,
			})
		}
	}
	return result, nil
}

func esUsoPersonal(ref string) bool {
	return len(ref) >= 13 && ref[:13] == "USO_PERSONAL:"
}

// mockNotifier implementa ports.NotifierRepository sin efectos secundarios.
type mockNotifier struct{}

func (m *mockNotifier) NotificarStockBajo(ctx context.Context, insumo *domain.Insumo) error { return nil }
func (m *mockNotifier) NotificarCambioEstado(ctx context.Context, pedidoID int, estado string) error {
	return nil
}

func setupInventarioService() (ports.InventarioService, *mockInsumoRepo, *mockMovimientoRepo) {
	insumoRepo := &mockInsumoRepo{insumos: make([]*domain.Insumo, 0)}
	movRepo := &mockMovimientoRepo{movimientos: make([]*domain.MovimientoInventario, 0)}
	notifier := &mockNotifier{}
	svc := services.NewInventarioService(insumoRepo, movRepo, notifier)
	return svc, insumoRepo, movRepo
}

func TestInventarioService_RegistrarMerma_DebitoExitoso(t *testing.T) {
	ctx := context.Background()
	svc, insumoRepo, _ := setupInventarioService()

	insumoRepo.Create(ctx, &domain.Insumo{
		Nombre:        "Tomate Manzano",
		UnidadBase:    "g",
		Stock:         5000,
		StockMinimo:   1000,
		CostoPromedio: 0.50,
	})

	mov, err := svc.RegistrarMerma(ctx, ports.MermaRequest{
		InsumoID: 1,
		Cantidad: 500,
		Motivo:   "Descomposición por falla de refrigeración",
	})
	if err != nil {
		t.Fatalf("Error inesperado al registrar merma: %v", err)
	}
	if mov.Cantidad != 500 {
		t.Errorf("Cantidad de merma esperada 500, obtenida %v", mov.Cantidad)
	}
	if mov.Tipo != domain.SalidaMerma {
		t.Errorf("Tipo de movimiento esperado SALIDA_MERMA, obtenido %v", mov.Tipo)
	}

	insumo, _ := insumoRepo.GetByID(ctx, 1)
	if insumo.Stock != 4500 {
		t.Errorf("Stock esperado 4500 después de merma, obtenido %v", insumo.Stock)
	}
}

func TestInventarioService_RegistrarMerma_CantidadInvalida(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := setupInventarioService()

	_, err := svc.RegistrarMerma(ctx, ports.MermaRequest{
		InsumoID: 1,
		Cantidad: 0,
	})
	if err != domain.ErrCantidadInvalida {
		t.Errorf("Esperado ErrCantidadInvalida, obtenido %v", err)
	}
}

func TestInventarioService_RegistrarMerma_StockInsuficiente(t *testing.T) {
	ctx := context.Background()
	svc, insumoRepo, _ := setupInventarioService()

	insumoRepo.Create(ctx, &domain.Insumo{
		Nombre:     "Aceite Vegetal",
		UnidadBase: "ml",
		Stock:      200,
	})

	_, err := svc.RegistrarMerma(ctx, ports.MermaRequest{
		InsumoID: 1,
		Cantidad: 500,
	})
	if err != domain.ErrStockInsuficiente {
		t.Errorf("Esperado ErrStockInsuficiente, obtenido %v", err)
	}
}

func TestInventarioService_RegistrarUsoPersonal_DebitoExitoso(t *testing.T) {
	ctx := context.Background()
	svc, insumoRepo, movRepo := setupInventarioService()

	insumoRepo.Create(ctx, &domain.Insumo{
		Nombre:        "Arroz Blanco",
		UnidadBase:    "g",
		Stock:         10000,
		StockMinimo:   2000,
		CostoPromedio: 0.30,
	})

	mov, err := svc.RegistrarUsoPersonal(ctx, ports.UsoPersonalRequest{
		InsumoID: 1,
		Cantidad: 2,
		Detalle:  "Comida de turno cocina",
	})
	if err != nil {
		t.Fatalf("Error inesperado al registrar uso personal: %v", err)
	}
	if mov.Cantidad != 2 {
		t.Errorf("Cantidad esperada 2, obtenida %v", mov.Cantidad)
	}
	if mov.Tipo != domain.SalidaMerma {
		t.Errorf("Tipo esperado SALIDA_MERMA (Estrategia A), obtenido %v", mov.Tipo)
	}
	if mov.Referencia != "USO_PERSONAL: Comida de turno cocina" {
		t.Errorf("Referencia esperada 'USO_PERSONAL: Comida de turno cocina', obtenida '%s'", mov.Referencia)
	}

	insumo, _ := insumoRepo.GetByID(ctx, 1)
	if insumo.Stock != 9998 {
		t.Errorf("Stock esperado 9998 después de uso personal, obtenido %v", insumo.Stock)
	}

	historial, _ := movRepo.ListarHistorialUsoPersonal(ctx, 10)
	if len(historial) != 1 {
		t.Errorf("Historial de uso personal esperado 1 registro, obtenido %d", len(historial))
	}
}

func TestInventarioService_RegistrarUsoPersonal_CantidadInvalida(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := setupInventarioService()

	_, err := svc.RegistrarUsoPersonal(ctx, ports.UsoPersonalRequest{
		InsumoID: 1,
		Cantidad: -1,
	})
	if err != domain.ErrCantidadInvalida {
		t.Errorf("Esperado ErrCantidadInvalida, obtenido %v", err)
	}
}

func TestInventarioService_RegistrarUsoPersonal_StockInsuficiente(t *testing.T) {
	ctx := context.Background()
	svc, insumoRepo, _ := setupInventarioService()

	insumoRepo.Create(ctx, &domain.Insumo{
		Nombre:     "Pollo Entero",
		UnidadBase: "u",
		Stock:      5,
	})

	_, err := svc.RegistrarUsoPersonal(ctx, ports.UsoPersonalRequest{
		InsumoID: 1,
		Cantidad: 10,
	})
	if err != domain.ErrStockInsuficiente {
		t.Errorf("Esperado ErrStockInsuficiente, obtenido %v", err)
	}
}

func TestInventarioService_RegistrarUsoPersonal_ConversionUnidad(t *testing.T) {
	ctx := context.Background()
	svc, insumoRepo, _ := setupInventarioService()

	insumoRepo.Create(ctx, &domain.Insumo{
		Nombre:     "Harina",
		UnidadBase: "g",
		Stock:      5000,
	})

	_, err := svc.RegistrarUsoPersonal(ctx, ports.UsoPersonalRequest{
		InsumoID:     1,
		Cantidad:     1, // 1 kg
		UnidadMedida: "kg",
		Detalle:      "Desayuno personal",
	})
	if err != nil {
		t.Fatalf("Error inesperado en conversión de unidad: %v", err)
	}

	insumo, _ := insumoRepo.GetByID(ctx, 1)
	if insumo.Stock != 4000 {
		t.Errorf("Stock esperado 4000 g (1 kg convertido), obtenido %v", insumo.Stock)
	}
}

func TestInventarioService_ListarHistorialMermas(t *testing.T) {
	ctx := context.Background()
	svc, insumoRepo, _ := setupInventarioService()

	insumoRepo.Create(ctx, &domain.Insumo{Nombre: "Insumo A", UnidadBase: "g", Stock: 1000})
	svc.RegistrarMerma(ctx, ports.MermaRequest{InsumoID: 1, Cantidad: 100, Motivo: "Rotura"})
	svc.RegistrarUsoPersonal(ctx, ports.UsoPersonalRequest{InsumoID: 1, Cantidad: 50, Detalle: "Comida"})

	historial, err := svc.ListarHistorialMermas(ctx, 10)
	if err != nil {
		t.Fatalf("Error al listar historial de mermas: %v", err)
	}
	if len(historial) != 1 {
		t.Errorf("Esperado 1 registro de merma, obtenido %d", len(historial))
	}
	if historial[0].Referencia != "Rotura" {
		t.Errorf("Referencia de merma esperada 'Rotura', obtenida '%s'", historial[0].Referencia)
	}
}
