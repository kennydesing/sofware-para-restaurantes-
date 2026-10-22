package sqlite

import (
	"context"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

func SeedData(
	ctx context.Context,
	db *DB,
	inventarioService ports.InventarioService,
	insumoRepo ports.InsumoRepository,
	platoRepo ports.PlatoRepository,
	recetaRepo ports.RecetaRepository,
) error {
	_, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO mesas (numero, capacidad) VALUES (1, 4), (2, 2), (3, 6), (4, 4), (5, 4), (6, 4), (7, 4), (8, 4), (9, 4), (10, 4)`)
	if err != nil {
		return err
	}

	insumoCarne, err := getOrCreateInsumo(ctx, insumoRepo, &domain.Insumo{
		Nombre:                      "Carne de Res Molida Premium",
		UnidadBase:                  "g",
		CostoPromedio:               0.0,
		StockMinimo:                 1000.0,
		Stock:                       0.0,
		PresentacionNombre:          "paquete",
		CantidadBasePorPresentacion: 1000.0,
		EmpaqueTipo:                 domain.EmpaqueUnitario,
		UnidadesPorEmpaque:          1,
	})
	if err != nil {
		return err
	}

	insumoPan, err := getOrCreateInsumo(ctx, insumoRepo, &domain.Insumo{
		Nombre:                      "Pan Brioche Artesanal",
		UnidadBase:                  "u",
		CostoPromedio:               0.0,
		StockMinimo:                 10.0,
		Stock:                       0.0,
		PresentacionNombre:          "unidad",
		CantidadBasePorPresentacion: 1.0,
		EmpaqueTipo:                 domain.EmpaqueUnitario,
		UnidadesPorEmpaque:          1,
	})
	if err != nil {
		return err
	}

	insumoQueso, err := getOrCreateInsumo(ctx, insumoRepo, &domain.Insumo{
		Nombre:                      "Queso Cheddar Madurado",
		UnidadBase:                  "g",
		CostoPromedio:               0.0,
		StockMinimo:                 300.0,
		Stock:                       0.0,
		PresentacionNombre:          "pieza",
		CantidadBasePorPresentacion: 1500.0,
		EmpaqueTipo:                 domain.EmpaqueUnitario,
		UnidadesPorEmpaque:          1,
	})
	if err != nil {
		return err
	}

	insumoAceite, err := getOrCreateInsumo(ctx, insumoRepo, &domain.Insumo{
		Nombre:                      "Aceite de Oliva Extra Virgen",
		UnidadBase:                  "ml",
		CostoPromedio:               0.0,
		StockMinimo:                 200.0,
		Stock:                       0.0,
		PresentacionNombre:          "botella",
		CantidadBasePorPresentacion: 1000.0,
		EmpaqueTipo:                 domain.EmpaqueUnitario,
		UnidadesPorEmpaque:          1,
	})
	if err != nil {
		return err
	}

	if insumoCarne.Stock <= 0 {
		_, _ = inventarioService.RegistrarCompra(ctx, ports.CompraRequest{
			InsumoID:      insumoCarne.ID,
			Cantidad:      5.0,
			UnidadCompra:  "kg",
			CostoUnitario: 12.00,
			Referencia:    "Factura Prov #F-1029",
		})
		_, _ = inventarioService.RegistrarCompra(ctx, ports.CompraRequest{
			InsumoID:      insumoCarne.ID,
			Cantidad:      2.0,
			UnidadCompra:  "kg",
			CostoUnitario: 14.00,
			Referencia:    "Factura Prov #F-1045",
		})
	}

	if insumoPan.Stock <= 0 {
		_, _ = inventarioService.RegistrarCompra(ctx, ports.CompraRequest{
			InsumoID:      insumoPan.ID,
			Cantidad:      4.0,
			UnidadCompra:  "docena",
			CostoUnitario: 6.00,
			Referencia:    "Panadería La Espiga #P-88",
		})
	}

	if insumoQueso.Stock <= 0 {
		_, _ = inventarioService.RegistrarCompra(ctx, ports.CompraRequest{
			InsumoID:      insumoQueso.ID,
			Cantidad:      1.5,
			UnidadCompra:  "kg",
			CostoUnitario: 16.00,
			Referencia:    "Lácteos El Roble #L-44",
		})
	}

	if insumoAceite.Stock <= 0 {
		_, _ = inventarioService.RegistrarCompra(ctx, ports.CompraRequest{
			InsumoID:      insumoAceite.ID,
			Cantidad:      2.0,
			UnidadCompra:  "l",
			CostoUnitario: 10.00,
			Referencia:    "Distribuidora Sol #D-12",
		})
	}

	platoBurger, err := getOrCreatePlato(ctx, platoRepo, &domain.Plato{
		Nombre:      "Hamburguesa Doble Smash Artesanal",
		PrecioVenta: 13.50,
		Activo:      true,
	})
	if err != nil {
		return err
	}

	recetasExistentes, _ := recetaRepo.GetByPlatoID(ctx, platoBurger.ID)
	if len(recetasExistentes) == 0 {
		_ = recetaRepo.Create(ctx, &domain.RecetaItem{
			PlatoID:           platoBurger.ID,
			InsumoID:          insumoCarne.ID,
			CantidadRequerida: 200.0,
			PorcentajeMerma:   10.0,
		})
		_ = recetaRepo.Create(ctx, &domain.RecetaItem{
			PlatoID:           platoBurger.ID,
			InsumoID:          insumoPan.ID,
			CantidadRequerida: 1.0,
			PorcentajeMerma:   0.0,
		})
		_ = recetaRepo.Create(ctx, &domain.RecetaItem{
			PlatoID:           platoBurger.ID,
			InsumoID:          insumoQueso.ID,
			CantidadRequerida: 40.0,
			PorcentajeMerma:   5.0,
		})
		_ = recetaRepo.Create(ctx, &domain.RecetaItem{
			PlatoID:           platoBurger.ID,
			InsumoID:          insumoAceite.ID,
			CantidadRequerida: 10.0,
			PorcentajeMerma:   0.0,
		})
	}

	// --- Productos de demostración adicional para probar la tabla v2 (Spec 004) ---
	insumosDemo := []struct {
		nombre                      string
		unidadBase                  string
		stockMinimo                 float64
		stock                       float64
		costo                       float64
		presentacionNombre          string
		cantidadBasePorPresentacion float64
	}{
		{"Harina de Trigo", "g", 1000.0, 5000.0, 0.80, "saco", 25000.0},
		{"Azúcar Refinada", "g", 500.0, 3000.0, 0.60, "saco", 25000.0},
		{"Sal Marina", "g", 200.0, 2000.0, 0.30, "paquete", 1000.0},
		{"Tomate Fresco", "g", 300.0, 1500.0, 1.20, "kg", 1000.0},
		{"Cebolla Blanca", "g", 200.0, 1000.0, 0.90, "kg", 1000.0},
		{"Pimiento Rojo", "g", 150.0, 800.0, 1.50, "kg", 1000.0},
		{"Lechuga Criolla", "u", 50.0, 200.0, 0.50, "unidad", 1.0},
		{"Pollo Entero", "u", 8.0, 40.0, 2.50, "unidad", 1.0},
		{"Huevos Docena", "u", 10.0, 50.0, 3.00, "docena", 12.0},
		{"Pan de Molde", "u", 30.0, 100.0, 0.90, "unidad", 1.0},
		{"Cerveza Polar Negra", "ml", 20.0, 7992.0, 0.61, "caja", 7992.0},
	}

	for _, demo := range insumosDemo {
		ins := &domain.Insumo{
			Nombre:                      demo.nombre,
			UnidadBase:                  demo.unidadBase,
			CostoPromedio:               demo.costo,
			StockMinimo:                 demo.stockMinimo,
			Stock:                       demo.stock,
			PresentacionNombre:          demo.presentacionNombre,
			CantidadBasePorPresentacion: demo.cantidadBasePorPresentacion,
		}
		// Configuración de empaque para cerveza Polar: 1 caja = 36 botellas × 222 ml
		if demo.nombre == "Cerveza Polar Negra" {
			ins.UnidadesPorEmpaque = 36
			ins.ContenidoMlPorUnidad = 222.0
		}
		_, _ = getOrCreateInsumo(ctx, insumoRepo, ins)
	}

	// --- Unidades de compra comerciales para productos demo (Spec 003 + Spec 004) ---
	// Esto permite que la columna MÍNIMO REQUERIDO muestre la presentación comercial
	// en negrita (ej: "1 Litro") y el equivalente en unidad base abajo (ej: <small>1.000,00 ml</small>).
	_, _ = db.ExecContext(ctx, `
		INSERT INTO insumo_unidades_compra (insumo_id, unidad_id, factor_n, factor_d, es_principal, descripcion)
		SELECT i.id, um.id, 1000, 1, 1, 'Pote de 1 Litro'
		FROM insumos i
		CROSS JOIN unidades_medida um
		WHERE i.nombre = 'Aceite de Oliva Extra Virgen' AND um.abreviatura = 'l'
	`)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO insumo_unidades_compra (insumo_id, unidad_id, factor_n, factor_d, es_principal, descripcion)
		SELECT i.id, um.id, 1000, 1, 1, 'Bulto de 1 kg'
		FROM insumos i
		CROSS JOIN unidades_medida um
		WHERE i.nombre = 'Carne de Res Molida Premium' AND um.abreviatura = 'kg'
	`)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO insumo_unidades_compra (insumo_id, unidad_id, factor_n, factor_d, es_principal, descripcion)
		SELECT i.id, um.id, 1, 1, 1, 'Unidad suelta'
		FROM insumos i
		CROSS JOIN unidades_medida um
		WHERE i.nombre = 'Lechuga Criolla' AND um.abreviatura = 'u'
	`)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO insumo_unidades_compra (insumo_id, unidad_id, factor_n, factor_d, es_principal, descripcion)
		SELECT i.id, um.id, 1, 1, 1, 'Unidad suelta'
		FROM insumos i
		CROSS JOIN unidades_medida um
		WHERE i.nombre = 'Pan Brioche Artesanal' AND um.abreviatura = 'u'
	`)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO insumo_unidades_compra (insumo_id, unidad_id, factor_n, factor_d, es_principal, descripcion)
		SELECT i.id, um.id, 1, 1, 1, 'Unidad suelta'
		FROM insumos i
		CROSS JOIN unidades_medida um
		WHERE i.nombre = 'Pan de Molde' AND um.abreviatura = 'u'
	`)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO insumo_unidades_compra (insumo_id, unidad_id, factor_n, factor_d, es_principal, descripcion)
		SELECT i.id, um.id, 1000, 1, 1, 'Bulto de 1 kg'
		FROM insumos i
		CROSS JOIN unidades_medida um
		WHERE i.nombre = 'Queso Cheddar Madurado' AND um.abreviatura = 'kg'
	`)

	return nil
}

func getOrCreateInsumo(ctx context.Context, repo ports.InsumoRepository, insumo *domain.Insumo) (*domain.Insumo, error) {
	if err := repo.Create(ctx, insumo); err != nil {
		existing, errGet := repo.GetByNombre(ctx, insumo.Nombre)
		if errGet == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return insumo, nil
}

func getOrCreatePlato(ctx context.Context, repo ports.PlatoRepository, plato *domain.Plato) (*domain.Plato, error) {
	if err := repo.Create(ctx, plato); err != nil {
		platos, errGet := repo.GetAll(ctx)
		if errGet == nil {
			for _, p := range platos {
				if p.Nombre == plato.Nombre {
					return p, nil
				}
			}
		}
		return nil, err
	}
	return plato, nil
}