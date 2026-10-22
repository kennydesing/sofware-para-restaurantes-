# Desglose de Tareas — 001: Separación de Interfaces POS

**Metodología:** 5 días de trabajo estimado  
**Orden:** Secuencial (backend → kitchen → client → admin → integración)  
**Estado Actual:** ✅ 100% Completado

---

## Fase 0: Infrastructure (½ día)

| ID | Tarea | Estado | Prioridad |
|----|-------|--------|-----------|
| T0.1 | Crear directorios `internal/adapters/inbound/{shared,client,kitchen,admin,api}/` | ✅ Completado | Alta |
| T0.2 | Mover y renombrar `web/` → `api/` (shared router + helpers) | ✅ Completado | Alta |
| T0.3 | Extraer seeding de `cmd/main.go` a `internal/adapters/outbound/sqlite/seeder.go` | ✅ Completado | Media |

---

## Fase 1: Backend Compartido — API Gateway (½ día)

| ID | Tarea | Estado | Prioridad |
|----|-------|--------|-----------|
| T1.1 | `internal/adapters/inbound/api/router.go`: HTTP mux con sub-routers por rol | ✅ Completado | Alta |
| T1.2 | `shared/middleware.go`: logging, recovery, CORS para SSE | ✅ Completado | Alta |
| T1.3 | `shared/helpers.go`: writeJSON, writeError, parseID | ✅ Completado | Alta |
| T1.4 | Mantener endpoints CRUD existentes (insumos, platos, pedidos, compras, mermas) | ✅ Completado | Alta |

---

## Fase 2: Interface de Cocina — KDS (1 día)

| ID | Tarea | Estado | Prioridad |
|----|-------|--------|-----------|
| T2.1 | `kitchen/routes.go`: router para `/kds` | ✅ Completado | Alta |
| T2.2 | `kitchen/handlers.go`: GET `/kds` (HTML), GET `/api/kds/orders` (JSON), POST `/api/pedidos/{id}/estado` | ✅ Completado | Alta |
| T2.3 | `kitchen/templates/kds.html`: layout Kanban (3 columnas: PENDIENTE, EN_PREPARACIÓN, LISTO) | ✅ Completado | Alta |
| T2.4 | `kitchen/templates/kds.css`: estilos Kanban responsive | ✅ Completado | Media |
| T2.5 | `kitchen/handlers.go`: SSE `/ws/kds` — broadcast de órdenes nuevas y cambios de estado | ✅ Completado | Alta |
| T2.6 | `kitchen/orders_store.go`: canal interno + goroutine de broadcast | ✅ Completado | Alta |
| T2.7 | Integrar `printer.ImprimirComanda` al crear pedido en KDS | ✅ Completado | Media |

---

## Fase 3: Interface de Cliente — QR por Mesa (1 día)

| ID | Tarea | Estado | Prioridad |
|----|-------|--------|-----------|
| T3.1 | `client/routes.go`: router para `/` con query param `?mesa={n}` | ✅ Completado | Alta |
| T3.2 | `client/handlers.go`: GET `/` → renderizar menú con `html/template` filtrado por mesa | ✅ Completado | Alta |
| T3.3 | `client/templates/menu.html`: menú móvil-first (grid de platos, carrito slide-up) | ✅ Completado | Alta |
| T3.4 | `client/handlers.go`: POST `/api/pedidos` desde cliente (validación mesa 1-10) | ✅ Completado | Alta |
| T3.5 | `client/templates/app.js`: carrito, checkout, toast de confirmación | ✅ Completado | Alta |
| T3.6 | `client/handlers.go`: SSE `/ws/client?mesa={n}` — seguimiento de estado en tiempo real | ✅ Completado | Alta |
| T3.7 | `client/templates/status.html`: overlay de estado del pedido (PENDIENTE → LISTO) | ✅ Completado | Media |
| T3.8 | `client/handlers.go`: GET `/qr` → página con 10 códigos QR (mesas 1-10) para imprimir | ✅ Completado | Alta |
| T3.9 | `client/templates/qr.html`: grid de 10 QR generados con `go-qrcode` o `<img>` por URL | ✅ Completado | Alta |
| T3.10 | `client/handlers.go`: Validar `mesa_id` ∈ [1,10] al crear pedido | ✅ Completado | Alta |

---

## Fase 4: Panel de Administración (1.5 días)

| ID | Tarea | Estado | Prioridad |
|----|-------|--------|-----------|
| T4.1 | `admin/routes.go`: router para `/admin` y subpaths (`/admin/insumos`, `/admin/platos`, etc.) | ✅ Completado | Alta |
| T4.2 | `admin/handlers.go`: GET `/admin` → dashboard con stats de stock | ✅ Completado | Alta |
| T4.3 | `admin/templates/dashboard.html`: cards (insumos, platos, alertas) + tabla de stock | ✅ Completado | Alta |
| T4.4 | `admin/handlers.go`: CRUD insumos (GET list, POST create) | ✅ Completado | Alta |
| T4.5 | `admin/templates/inventory.html`: tabla con botones de compra/merma/ajuste | ✅ Completado | Alta |
| T4.6 | `admin/handlers.go`: CRUD platos + recetas (escandallo con merma) | ✅ Completado | Alta |
| T4.7 | `admin/templates/menu.html`: tabla de platos con formulario de escandallo | ✅ Completado | Alta |
| T4.8 | `admin/handlers.go`: historial de pedidos con filtros por mesa/estado/fecha | ✅ Completado | Alta |
| T4.9 | `admin/templates/orders.html`: tabla extendida de pedidos | ✅ Completado | Media |
| T4.10 | `admin/handlers.go`: libro diario de auditoría (movimientos por insumo) | ✅ Completado | Alta |
| T4.11 | `admin/templates/audit.html`: tabla de movimientos + filtros | ✅ Completado | Media |
| T4.12 | Auth básica para `/admin/*` (HTTP Basic Auth, configuración via env `ADMIN_TOKEN`) | ✅ Completado | Media |

---

## Fase 5: Integración y Verificación (½ día)

| ID | Tarea | Estado | Prioridad |
|----|-------|--------|-----------|
| T5.1 | Actualizar `cmd/main.go`: wiring de los 3 routers + SSE channels | ✅ Completado | Alta |
| T5.2 | `go vet ./...` + `go build` con CGO_ENABLED=1 | ✅ Completado | Alta |
| T5.3 | Test end-to-end: crear pedido desde QR → ver en KDS → procesar → ver en admin | ✅ Completado | Alta |
| T5.4 | Test rollback: crear pedido grande → verificar stock intacto | ✅ Completado | Alta |
| T5.5 | Test SSE: estado cambia en tiempo real en cliente y KDS | ✅ Completado | Alta |
| T5.6 | Actualizar `MEMORY.md` con decisiones finales | ✅ Completado | Baja |

---

## Resumen de Entregables

| Fase | Entregable | Archivos clave |
|------|-----------|----------------|
| F0 | Infrastructure | `inbound/{shared,api}/`, `sqlite/seeder.go` |
| F1 | API Gateway | `api/router.go`, `shared/helpers.go` |
| F2 | KDS | `kitchen/{routes,handlers}.go`, `kitchen/templates/` |
| F3 | Cliente QR | `client/{routes,handlers}.go`, `client/templates/` |
| F4 | Admin | `admin/{routes,handlers}.go`, `admin/templates/` |
| F5 | Integración | `cmd/main.go` actualizado |
