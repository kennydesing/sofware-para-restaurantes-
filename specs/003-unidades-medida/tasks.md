# Tareas de Implementación — 003: Sistema Integral de Unidades de Medida

**Versión:** 1.0  
**Estado:** Pendiente de ejecución  
**Fecha:** 2026-10-14  
**Estimación total:** ~6–8 horas de desarrollo  
**Prerequisito:** `plan.md` aprobado, migraciones 000001–000003 aplicadas.

---

## Leyenda de Prioridad

| Símbolo | Significado |
|---------|-------------|
| 🔴 P0 | Bloqueante — sin esto no avanza nada |
| 🟠 P1 | Crítico — requerido para funcionalidad core |
| 🟡 P2 | Importante — mejora la experiencia |
| 🟢 P3 | Nice-to-have — optimización o UI |

---

## FASE 1 — Base de Datos y Dominio

### T-01 🔴 Crear migración `000004_unidades_medida.up.sql`
**Estimado:** 30 min  
**Archivo:** `migrations/000004_unidades_medida.up.sql`

- [ ] Crear tabla `unidades_medida` con campos: `id`, `nombre`, `abreviatura`, `categoria`, `es_base`, `factor_n`, `factor_d`, `created_at`
- [ ] Crear índice único `idx_unidades_abrev` sobre `abreviatura`
- [ ] Crear índice único parcial `idx_unidades_base_cat` sobre `(categoria) WHERE es_base = 1`
- [ ] Crear tabla `factores_conversion` con FK a `unidades_medida` y `UNIQUE(origen_id, destino_id)`
- [ ] Crear tabla `insumo_unidades_compra` con FK a `insumos` e `unidades_medida`
- [ ] Crear índice único parcial `idx_iuc_principal` sobre `(insumo_id) WHERE es_principal = 1`
- [ ] `ALTER TABLE recetas_escandallo ADD COLUMN unidad_receta_id INTEGER NULL`
- [ ] INSERT OR IGNORE del catálogo de 18 unidades estándar (ver `plan.md §2.1`)
- [ ] Verificar que `go run cmd/main.go` aplica la migración sin errores

### T-02 🔴 Crear migración `000004_unidades_medida.down.sql`
**Estimado:** 10 min  
**Archivo:** `migrations/000004_unidades_medida.down.sql`

- [ ] DROP TABLE `insumo_unidades_compra`
- [ ] DROP TABLE `factores_conversion`
- [ ] DROP TABLE `unidades_medida`
- [ ] Comentar instrucción `DROP COLUMN` con nota de versión SQLite requerida

### T-03 🔴 Crear entidades de dominio puras
**Estimado:** 45 min  
**Archivos:** `internal/core/domain/unidad.go`, `internal/core/domain/conversion.go`

- [ ] Definir tipo `Categoria` con constantes `CategoriaVolumen`, `CategoriaPeso`, `CategoriaConteo`, `CategoriaCulinaria`
- [ ] Definir struct `UnidadMedida` con campos `ID`, `Nombre`, `Abreviatura`, `Categoria`, `EsBase`, `FactorN`, `FactorD`
- [ ] Definir struct `InsumoUnidadCompra` con todos sus campos
- [ ] Implementar `ConvertirEntreMismaCategoria(cantN, cantD int64, origen, destino UnidadMedida) (rN, rD int64, err error)`
  - Validar misma categoría → `ErrCategoriaIncompatible`
  - Validar denominadores ≠ 0 → `ErrDenominadorCero`
  - Fórmula: `rN = cantN × factorN_orig × factorD_dest`, `rD = cantD × factorD_orig × factorN_dest`
- [ ] Implementar `ConvertirAFloat64(n, d int64, decimals int) float64` con `math.Round`
- [ ] Implementar `ConsumoBaseConMerma(cantN, cantD, factN, factD int64, merma float64, platos int64) float64`
- [ ] Agregar errores a `errors.go`: `ErrCategoriaIncompatible`, `ErrDenominadorCero`
- [ ] Verificar que ningún import rompe la regla: **cero dependencias de `database/sql`** en `domain/`

---

## FASE 2 — Pruebas Unitarias de Dominio

### T-04 🔴 Pruebas de precisión matemática
**Estimado:** 60 min  
**Archivo:** `internal/core/domain/conversion_test.go`

- [ ] Test `TestConvertirEntreMismaCategoria_KgAGramos`: 1 kg → 1000 g, exacto
- [ ] Test `TestConvertirEntreMismaCategoria_LibraAGramos`: 1 lb → 453.59237 g (4 decimales)
- [ ] Test `TestConvertirEntreMismaCategoria_GalonAMl`: 1 gal → 3785.4118 ml (4 decimales)
- [ ] Test `TestConvertirEntreMismaCategoria_CategoriaIncompatible`: kg → ml → `ErrCategoriaIncompatible`
- [ ] Test `TestConvertirAFloat64_Redondeo4Decimales`: verificar que 1/3 → 0.3333 (no 0.33333333...)
- [ ] Test `TestConsumoBaseConMerma_ConMerma5Pct`: 200g × (1+0.05) × 1 plato = 210.0
- [ ] Test `TestConsumoBaseConMerma_SinMerma`: 500 ml × 1.0 × 2 platos = 1000.0
- [ ] Test `TestConsumoBaseConMerma_DenominadorCero`: retorna 0 sin panic
- [ ] Ejecutar `go test ./internal/core/domain/... -v` y confirmar 100% PASS

---

## FASE 3 — Puertos

### T-05 🟠 Extender `outbound_ports.go` con `UnidadMedidaRepository`
**Estimado:** 20 min  
**Archivo:** `internal/core/ports/outbound_ports.go`

- [ ] Añadir interfaz `UnidadMedidaRepository` con métodos:
  - `ObtenerTodas(ctx) ([]domain.UnidadMedida, error)`
  - `ObtenerPorID(ctx, id int64) (domain.UnidadMedida, error)`
  - `ObtenerPorAbreviatura(ctx, abrev string) (domain.UnidadMedida, error)`
  - `ObtenerPorCategoria(ctx, cat domain.Categoria) ([]domain.UnidadMedida, error)`
  - `Crear(ctx, u domain.UnidadMedida) (int64, error)`
  - `ObtenerUnidadesCompraInsumo(ctx, insumoID int64) ([]domain.InsumoUnidadCompra, error)`
  - `GuardarUnidadCompraInsumo(ctx, iuc domain.InsumoUnidadCompra) error`
- [ ] Verificar que `go vet ./...` pasa limpio

### T-06 🟠 Extender `inbound_ports.go` con `ConversionService`
**Estimado:** 15 min  
**Archivo:** `internal/core/ports/inbound_ports.go`

- [ ] Añadir interfaz `ConversionService` con métodos:
  - `Convertir(ctx, cantidad float64, origenAbrev, destinoAbrev string) (float64, error)`
  - `CostoUnitarioBase(ctx, costoTotal, cantidadComprada float64, unidadCompraAbrev string, insumoID int64) (float64, error)`
  - `ConsumoReceta(ctx, cantidadReceta float64, unidadRecetaAbrev string, insumoID int64, porcentajeMerma float64, numPlatos int) (float64, error)`

---

## FASE 4 — Adaptador SQLite

### T-07 🟠 Implementar `UnidadMedidaRepositorySQLite`
**Estimado:** 60 min  
**Archivo:** `internal/adapters/outbound/sqlite/unidad_repo.go`

- [ ] Struct `UnidadMedidaRepositorySQLite` con campo `db *sql.DB`
- [ ] `ObtenerTodas`: `SELECT id, nombre, abreviatura, categoria, es_base, factor_n, factor_d FROM unidades_medida ORDER BY categoria, es_base DESC`
- [ ] `ObtenerPorAbreviatura`: query por índice único (prepared statement)
- [ ] `ObtenerPorCategoria`: query con `WHERE categoria = ?`
- [ ] `Crear`: INSERT con validación de denominador > 0 antes de persistir
- [ ] `ObtenerUnidadesCompraInsumo`: SELECT de `insumo_unidades_compra JOIN unidades_medida`
- [ ] `GuardarUnidadCompraInsumo`: INSERT OR REPLACE con manejo de conflicto en `es_principal`
- [ ] Usar siempre **prepared statements** (nunca concatenación de strings en SQL)
- [ ] Verificar que `go vet ./...` pasa limpio

---

## FASE 5 — Servicio de Conversión

### T-08 🟠 Implementar `ConversionServiceImpl`
**Estimado:** 75 min  
**Archivo:** `internal/core/services/conversion_service.go`

- [ ] Struct con `repo UnidadMedidaRepository`, `cache map[string]domain.UnidadMedida`, `mu sync.RWMutex`
- [ ] `CargarCache(ctx)`: carga todas las unidades y construye mapa `abreviatura → UnidadMedida`
- [ ] `InvalidarCache()`: resetea el mapa (llamado tras crear nueva unidad)
- [ ] `Convertir()`:
  - [ ] Lookup origen y destino en cache (lock RRead)
  - [ ] Llamar `domain.ConvertirEntreMismaCategoria()`
  - [ ] Retornar `domain.ConvertirAFloat64(rN, rD, 4)`
- [ ] `CostoUnitarioBase()`:
  - [ ] Obtener equivalencia de compra del insumo (`ObtenerUnidadesCompraInsumo`)
  - [ ] `unidadesBase = cantidadComprada × (factorN / factorD)`
  - [ ] `costo = costoTotal / unidadesBase`, 4 decimales
  - [ ] Manejar caso `unidadesBase = 0` → retornar error
- [ ] `ConsumoReceta()`:
  - [ ] Si `unidadRecetaAbrev` está vacía → usar unidad base del insumo directamente
  - [ ] Convertir `cantidadReceta` a unidad base
  - [ ] Aplicar `domain.ConsumoBaseConMerma()`
- [ ] Verificar que no hay imports de `database/sql` (solo `ports` e `domain`)

### T-09 🟡 Pruebas del servicio de conversión
**Estimado:** 45 min  
**Archivo:** `internal/core/services/conversion_service_test.go`

- [ ] Mock de `UnidadMedidaRepository` con datos de prueba (kg, g, ml, l)
- [ ] Test `TestConvertir_KgAGramos`: 2.5 kg → 2500 g
- [ ] Test `TestConvertir_CategoriaIncompatible`: kg → ml → error
- [ ] Test `TestCostoUnitarioBase_CajaDeTomate`: 2 cajas 10kg @ $17 → $0.00085/g
- [ ] Test `TestConsumoReceta_ConMerma`: 200 tbsp @ 5% merma → valor en ml
- [ ] Ejecutar `go test ./internal/core/services/... -v`

---

## FASE 6 — Wiring en `cmd/main.go`

### T-10 🟠 Registrar repositorio y servicio
**Estimado:** 20 min  
**Archivo:** `cmd/main.go`

- [ ] Instanciar `sqlite.NewUnidadMedidaRepositorySQLite(db)`
- [ ] Instanciar `services.NewConversionService(unidadRepo)`
- [ ] Llamar `conversionSvc.CargarCache(ctx)` al arrancar
- [ ] Pasar `conversionSvc` al `PedidoService` e `InventarioService` donde aplique
- [ ] Verificar que `go build ./cmd/main.go` compila sin errores

---

## FASE 7 — Integración con Flujo de Pedidos

### T-11 🟡 Actualizar `PedidoService` para usar `ConversionService`
**Estimado:** 40 min  
**Archivo:** `internal/core/services/pedido_service.go`

- [ ] Añadir campo `conversionSvc ports.ConversionService` al struct
- [ ] En `ProcesarPedido()`, por cada `RecetaItem`:
  - [ ] Si `item.UnidadRecetaID == 0` → ruta legacy (`domain.CalcularConsumoTotalConMerma`)
  - [ ] Si `item.UnidadRecetaID != 0` → `conversionSvc.ConsumoReceta()`
- [ ] Verificar rollback atómico si `ErrCategoriaIncompatible` (no descontar stock parcialmente)
- [x] Ejecutar `go test ./...` — todos los tests deben seguir en verde

---

## FASE 8 — Endpoints REST de Unidades (Admin API)

### T-12 🟡 Agregar endpoints en el panel admin
**Estimado:** 60 min  
**Archivos:** `internal/adapters/inbound/admin/routes.go`, `handlers.go`

- [x] `GET  /admin/api/unidades`              → lista todas las unidades (para dropdowns)
- [x] `GET  /admin/api/unidades?categoria=PESO` → filtrar por categoría
- [x] `POST /admin/api/unidades`              → crear unidad personalizada (body: nombre, abrev, categoria, factor_n, factor_d)
- [x] `GET  /admin/api/insumos/{id}/unidades-compra` → unidades de compra de un insumo
- [x] `POST /admin/api/insumos/{id}/unidades-compra` → registrar nueva presentación de compra
- [x] Validar CSRF en todos los POST
- [x] Retornar JSON con estructura consistente (`{"data": ..., "error": null}`)
- [x] Llamar `conversionSvc.InvalidarCache()` tras POST exitoso de nueva unidad

### T-13 🟡 Prueba de integración de endpoints
**Estimado:** 30 min  
**Archivo:** `internal/adapters/inbound/admin/unidades_handler_test.go`

- [x] Test `GET /admin/api/unidades` → retorna 200 + array JSON con ≥ 17 unidades
- [x] Test `POST /admin/api/unidades` con body válido → 201 Created
- [x] Test `POST /admin/api/unidades` con factor_d = 0 → 400 Bad Request
- [x] Test `POST /admin/api/unidades` con categoria inválida → 400 Bad Request

---

## FASE 9 — Verificación Final

### T-14 🔴 Verificación end-to-end
**Estimado:** 20 min

- [x] Ejecutar `go run cmd/main.go` → sin errores de migración ni de arranque
- [x] Confirmar que la cache de unidades se carga al inicio (log: "Unidades de medida cargadas: 18")
- [x] Registrar manualmente un insumo con unidad de compra via API (curl o UI)
- [x] Verificar que un pedido con receta en `tbsp` descuenta correctamente en `ml` del stock
- [x] Verificar rollback: pedido sin stock suficiente → ROLLBACK sin alterar inventario

### T-15 🔴 Suite de pruebas y compilación final
**Estimado:** 10 min

- [x] Ejecutar `go test -v ./...` → 100% PASS, cero regresiones
- [x] Ejecutar `go vet ./...` → sin advertencias
- [x] Compilar: `go build -ldflags="-s -w" -o bin/pos_core.exe ./cmd/main.go`
- [x] Verificar tamaño del binario: `< 15 MB` (11.16 MB)
- [x] Actualizar `MEMORY.md` con el estado de la spec 003

---

## Resumen de Estimaciones

| Fase | Tareas | Estimado |
|------|--------|----------|
| F1 — Base de datos y dominio | T-01, T-02, T-03 | 85 min |
| F2 — Pruebas de dominio | T-04 | 60 min |
| F3 — Puertos | T-05, T-06 | 35 min |
| F4 — Adaptador SQLite | T-07 | 60 min |
| F5 — Servicio de conversión | T-08, T-09 | 120 min |
| F6 — Wiring | T-10 | 20 min |
| F7 — Integración pedidos | T-11 | 40 min |
| F8 — Endpoints REST | T-12, T-13 | 90 min |
| F9 — Verificación final | T-14, T-15 | 30 min |
| **TOTAL** | **15 tareas** | **~8 horas** |

---

> **Orden de ejecución obligatorio:** F1 → F2 → F3 → F4 → F5 → F6 → F7 → F8 → F9  
> Las fases son secuenciales: cada una depende de la anterior.  
> Preguntar antes de modificar tablas existentes (`insumos`, `recetas_escandallo`) o interfaces en `ports/`.
