# Tasks 005 — Desglose de Tareas Secuenciales: Empaques y Presentaciones de Cervezas y Bebidas

**Versión:** 1.0  
**Fecha:** 2026-10-20  
**Estado:** BORRADOR — Pendiente de Aprobación  
**Referencia:** `specs/005-empaques-cervezas-personalizados/spec.md` + `plan.md`  
**Ejecución:** Cada tarea es atómica, enfocada y verificable. Ningún código de producción se modificará hasta la aprobación formal de esta especificación.

---

## Matriz Resumen de Tareas

| ID | Bloque | Descripción | Archivos Principales | Dependencias |
|---|---|---|---|---|
| **TASK-000** | Bloque 0 | Aprobación de spec.md, plan.md y tasks.md | `specs/005-*/` | Ninguna |
| **TASK-001** | Bloque 1 | Migración SQL `000006_empaques_cervezas_bebidas` | `migrations/000006_*.sql` | TASK-000 |
| **TASK-002** | Bloque 2 | Entidades de Dominio (`Insumo`, `TipoEmpaque`) | `internal/core/domain/insumo.go` | TASK-001 |
| **TASK-003** | Bloque 2 | Pruebas Unitarias de Conversión en Dominio | `internal/core/domain/insumo_test.go` | TASK-002 |
| **TASK-004** | Bloque 3 | Repositorio SQLite: Adaptar Queries y Scanners | `internal/adapters/outbound/sqlite/insumo_repo.go` | TASK-002 |
| **TASK-005** | Bloque 4 | Lógica en `InventarioService` (Conversión Compras) | `internal/core/services/inventario_service.go` | TASK-004 |
| **TASK-006** | Bloque 5 | DTOs y Handlers HTTP en Adaptador Admin | `internal/adapters/inbound/admin/handlers.go` | TASK-005 |
| **TASK-007** | Bloque 6 | Maquetación HTML de Empaques en Modales | `internal/adapters/inbound/admin/templates/admin.html` | TASK-006 |
| **TASK-008** | Bloque 6 | Estilos CSS para Empaques y Custom Pack | `internal/adapters/inbound/admin/templates/admin.css` | TASK-007 |
| **TASK-009** | Bloque 6 | Lógica JavaScript Reactiva y Preview Dinámico | `internal/adapters/inbound/admin/static/admin.js` | TASK-008 |
| **TASK-010** | Bloque 7 | Pruebas de Integración y Débito Atómico en POS | `cmd/main.go`, `internal/core/services/` | TASK-009 |
| **TASK-011** | Bloque 8 | Verificación Final de Compilación y Memoria | `MEMORY.md`, Binario `pos_core.exe` | TASK-010 |

---

## BLOQUE 0 — Aprobación y Pre-requisitos

### TASK-000 · Aprobación Formal de Especificación 005
- **Responsable:** Product Owner / Arquitecto de Software
- **Entregables:** Aprobación explícita de `spec.md`, `plan.md` y `tasks.md`.
- **Criterio de Salida:** Confirmación sin objeciones para dar inicio a la creación de código funcional.

---

## BLOQUE 1 — Base de Datos y Migraciones (SQLite WAL)

### TASK-001 · Crear Migración `000006_empaques_cervezas_bebidas.up.sql` y `.down.sql`
- **Archivos:**
  - `migrations/000006_empaques_cervezas_bebidas.up.sql`
  - `migrations/000006_empaques_cervezas_bebidas.down.sql`
- **Descripción:** Agregar columnas `empaque_tipo`, `empaque_nombre`, `unidades_por_empaque`, `contenido_ml_por_unidad` y `es_retornable` a la tabla `insumos`, con restricciones CHECK y sus correspondientes índices SQLite.
- **Verificación:**
  - El motor de migración de `cmd/main.go` aplica la versión 6 sin errores en modo WAL.
  - Comprobar esquema con:
    ```sql
    PRAGMA table_info(insumos);
    ```

---

## BLOQUE 2 — Capa de Dominio (Hexagonal Core)

### TASK-002 · Extender `domain.Insumo` y Enums de Empaque
- **Archivo:** `internal/core/domain/insumo.go`
- **Descripción:**
  - Declarar tipo `TipoEmpaque` con las constantes: `EmpaqueUnitario`, `EmpaqueGavera36`, `EmpaqueCaja24Bot`, `EmpaqueCaja24Lata`, `EmpaqueSixPack6`, `EmpaqueCaja12Art`, `EmpaquePersonalizado`.
  - Agregar campos `EmpaqueTipo`, `EmpaqueNombre`, `UnidadesPorEmpaque`, `ContenidoMlPorUnidad`, `EsRetornable` en el struct `Insumo`.
  - Implementar método `ConvertirEmpaquesABase(cantidadEmpaques float64) float64`.
  - Implementar validaciones puras: `unidades_por_empaque >= 1` y `contenido_ml >= 0`.
- **Restricción:** Cero dependencias externas (`database/sql`, frameworks HTTP).

### TASK-003 · Pruebas Unitarias de Dominio
- **Archivo:** `internal/core/domain/insumo_test.go`
- **Descripción:**
  - Test de conversión de gavera de 36: 1 gavera = 36 unidades, 3 gaveras = 108 unidades.
  - Test de caja de 24 latas: 2 cajas = 48 unidades.
  - Test de Custom Pack de 18 botellas: 5 packs = 90 unidades.
  - Test de validación: rechazar unidades <= 0 en empaque.
- **Verificación:**
  ```bash
  go test -v ./internal/core/domain/...
  ```

---

## BLOQUE 3 — Adaptadores Outbound (SQLite)

### TASK-004 · Actualizar Repositorio SQLite de Insumos
- **Archivo:** `internal/adapters/outbound/sqlite/insumo_repo.go`
- **Descripción:**
  - Actualizar `Create(ctx, insumo)` para persistir las nuevas columnas de empaque.
  - Actualizar `Update(ctx, insumo)` para permitir la edición de la configuración de empaque.
  - Actualizar métodos de lectura `GetByID` y `GetAll` para mapear los nuevos campos desde las filas SQLite.

### TASK-005 · Actualizar `GetAllEnriquecido()` y DTO de Salida
- **Archivo:** `internal/adapters/outbound/sqlite/insumo_repo.go`
- **Descripción:**
  - Modificar la consulta SQL enriquecida que calcula mermas acumuladas para incluir los campos de empaque (`i.empaque_tipo`, `i.empaque_nombre`, `i.unidades_por_empaque`, `i.contenido_ml_por_unidad`, `i.es_retornable`).
  - Actualizar el struct `InsumoEnriquecido` en `internal/core/ports/outbound_ports.go`.
- **Verificación:**
  ```bash
  go test -v ./internal/adapters/outbound/sqlite/...
  ```

---

## BLOQUE 4 — Servicios de Dominio (Casos de Uso)

### TASK-006 · Actualizar `InventarioService.RegistrarCompra` para Empaques
- **Archivo:** `internal/core/services/inventario_service.go`
- **Descripción:**
  - Si el insumo tiene configurado un empaque (`unidades_por_empaque > 1`) y la compra se registra en unidad empaque (ej. "Gavera", "Caja", o la unidad configurada), convertir automáticamente:
    $$\text{CantidadBase} = \text{CantidadComprada} \times \text{Insumo.UnidadesPorEmpaque}$$
  - Calcular el costo unitario por botella base:
    $$\text{CostoUnitarioBase} = \frac{\text{CostoTotalCompra}}{\text{CantidadBase}}$$
  - Recalcular el Costo Promedio Ponderado con base en la fórmula de dominio existente.
  - Registrar el movimiento en el libro diario con la descripción explícita de las cajas recibidas.
- **Verificación:**
  ```bash
  go test -v ./internal/core/services/...
  ```

---

## BLOQUE 5 — Adaptadores Inbound (HTTP REST)

### TASK-007 · Actualizar Handlers de Insumos en Panel Admin
- **Archivo:** `internal/adapters/inbound/admin/handlers.go`
- **Descripción:**
  - Actualizar `InsumoRequestDTO` con los campos de empaque.
  - Actualizar `handleInsumosAPI` (POST) para validar y propagar la configuración de empaques predefinidos o personalizados.
  - Actualizar `handleInsumoByIDAPI` (PUT) para permitir modificar la configuración de empaque.
  - Mapear correctamente en `InsumoResponseDTO` hacia el cliente.
- **Verificación:**
  - Endpoint `/admin/api/insumos` retorna JSON con todos los atributos de empaque.

---

## BLOQUE 6 — Interfaz Frontend Admin (UI/UX)

### TASK-008 · Maquetar Sección de Empaques en `admin.html`
- **Archivo:** `internal/adapters/inbound/admin/templates/admin.html`
- **Descripción:**
  - Insertar en `#modal-gestionar-inventario` la tarjeta `.empaque-config-card` con el selector `#prod-empaque-tipo` (opciones predefinidas y opción personalizada).
  - Insertar el contenedor dinámico `#custom-pack-fields` con los campos de nombre, unidades, ml y retornabilidad.
  - Insertar la tarjeta de previsualización `#costeo-empaque-preview`.
  - Replicar campos correspondientes en `#modal-editar-insumo` para edición.

### TASK-009 · Incorporar Estilos Visuales en `admin.css`
- **Archivo:** `internal/adapters/inbound/admin/templates/admin.css` (y `static/admin.css`)
- **Descripción:**
  - Añadir reglas CSS para `.empaque-config-card`, `.custom-pack-panel`, `.costeo-empaque-box`, `.costeo-grid`, `.badge-retornable` y animaciones `@keyframes fadeInDown`.
  - Asegurar adaptabilidad responsiva en pantallas móviles ($\le 640\text{ px}$).

### TASK-010 · Implementar Lógica Dinámica en `admin.js`
- **Archivo:** `internal/adapters/inbound/admin/static/admin.js` (y `templates/admin.js`)
- **Descripción:**
  - Implementar `alCambiarTipoEmpaque()`: mostrar/ocultar `#custom-pack-fields` y preseleccionar unidad base `u`.
  - Implementar `recalcularCosteoEmpaque()`: cálculo reactivo de costo por botella, margen unitario (%) y ganancia total por caja completa ante cambios en `#prod-costo-usd` o `#prod-venta-usd`.
  - Integrar lectura y envío de datos de empaque en `guardarProductoInventario()` y `guardarEditarInsumo()`.
  - Añadir badge de empaque / retornabilidad en la tabla principal de insumos (`renderizarInsumos`).

---

## BLOQUE 7 — Pruebas End-to-End y Débito Atómico en Ventas POS

### TASK-011 · Demostración y Validación E2E del Ciclo Completo
- **Archivos:** `cmd/main.go` / Scripts de verificación
- **Flujo de Prueba:**
  1. **Creación:** Registrar insumo *"Polar Pilsen 222ml"* con empaque `GAVERA_36` (36 botellas, $222\text{ ml}$, retornable).
  2. **Compra Inicial:** Entrada de $2\text{ gaveras}$ a $\$18.00$ por gavera ($\$36.00$ total).
     - *Resultado Esperado:* Stock $= 72\text{ u}$, Costo Promedio $= \$0.50$ por botella.
  3. **Custom Pack:** Registrar *"Artesanal Premium"* con `PERSONALIZADO` ($18\text{ unidades}$ de $330\text{ ml}$).
     - Entrada de $1\text{ caja}$ a $\$27.00$.
     - *Resultado Esperado:* Stock $= 18\text{ u}$, Costo Promedio $= \$1.50$ por botella.
  4. **Venta POS:** Crear pedido con $2\text{ Polar Pilsen}$ ($2\text{ u}$) y comanda enviada a cocina.
     - *Resultado Esperado:* Stock remanente de Polar Pilsen decrementa a exactamente $70\text{ u}$.
  5. **Rollback ante Quiebre:** Intentar vender $80\text{ botellas}$ de Polar Pilsen.
     - *Resultado Esperado:* Error `ErrStockInsuficiente`, transacción abortada y stock permanece inalterado en $70\text{ u}$.

---

## BLOQUE 8 — Cierre y Actualización de Memoria

### TASK-012 · Actualizar `MEMORY.md` y Compilar Binario
- **Archivo:** `MEMORY.md`
- **Descripción:**
  - Resumir las decisiones arquitectónicas tomadas para la gestión de empaques y cervezas.
  - Registrar la migración `000006` en el historial de base de datos.
  - Compilar binario de producción optimizado:
    ```bash
    go build -ldflags="-s -w" -o bin/pos_core.exe ./cmd/main.go
    ```
  - Verificar que el tamaño del binario se mantenga liviano ($< 15\text{ MB}$) y las pruebas pasen al $100\%$.
