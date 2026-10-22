# Tasks 006 — Desglose de Tareas Secuenciales: Pestañas de Inventario e Inserción en Línea

**Versión:** 1.0  
**Fecha:** 2026-10-21  
**Estado:** BORRADOR — Pendiente de aprobación  
**Referencias:** [spec.md](file:///c:/Users/admin/Desktop/sofware%20para%20restaurantes/specs/006-pestanas-inventario-merma-personal/spec.md) · [plan.md](file:///c:/Users/admin/Desktop/sofware%20para%20restaurantes/specs/006-pestanas-inventario-merma-personal/plan.md)

---

## Principios de Ejecución y Reglas de Trabajo
- **Cero cambios destructivos sin aprobación:** Las tareas se ejecutan en orden estricto de capas (Dominio → Puertos → Adaptadores Backend → Endpoints HTTP → Templates HTML → Lógica JS).
- **No alterar el código funcional** hasta que la presente especificación sea aprobada formalmente.
- Cada tarea es atómica, verificable y acompañada de su comando de comprobación (`go test`, `curl`, o verificación visual en navegador).
- Preservar al 100% los tests verdes existentes (`go test ./...`) en cada paso.

---

## FASE 0 — Aprobación y Acuerdos de Arquitectura

### TASK-001 · Aprobación formal de la Especificación 006
- **Responsable:** Arquitecto de Software / Tech Lead / Usuario
- **Entregable:** Aprobación explícita de `spec.md`, `plan.md` y `tasks.md`.
- **Decisiones Clave a Ratificar:**
  1. Uso de la **Estrategia A** de auditoría (sin alterar el DDL de `movimientos_inventario` en SQLite, utilizando `referencia: "USO_PERSONAL: ..."` y `tipo: SALIDA_MERMA` para total retrocompatibilidad).
  2. Adopción de las **8 columnas definitivas** en el Inventario Principal: *Producto*, *Cantidad*, *Peso/Unidad*, *Stock Mínimo*, *Costo*, *Precio Final*, *Margen de Ganancias*, *Ajuste*.
  3. Adopción del cuadro simplificado de **2 columnas** para Mermas (*Producto*, *Cantidad*) y **2 columnas estructuradas** para Uso Personal (*Producto*, *Cantidad y Peso*).
  4. Flujo 100% **Inline Editing** con eliminación definitiva de cualquier modal flotante en inventario.

---

## FASE 1 — Backend: Dominio, Puertos y Casos de Uso (Core)

### TASK-002 · Extensión de DTOs e Interfaces en Puertos de Entrada
- **Archivo:** `internal/core/ports/inbound_ports.go`
- **Acción:**
  - Definir `UsoPersonalRequest` con campos: `InsumoID`, `Cantidad`, `UnidadMedida`, `Detalle`.
  - Definir `HistorialSalidaDTO` para lectura agregada de pérdidas y consumo de personal.
  - Añadir métodos al puerto `InventarioService`:
    ```go
    RegistrarUsoPersonal(ctx context.Context, req UsoPersonalRequest) (*domain.MovimientoInventario, error)
    ListarHistorialMermas(ctx context.Context, limite int) ([]*HistorialSalidaDTO, error)
    ListarHistorialUsoPersonal(ctx context.Context, limite int) ([]*HistorialSalidaDTO, error)
    ```
- **Criterio de Aceptación:** El código compila sin errores de sintaxis (`go vet ./internal/core/...`).

### TASK-003 · Implementación del Caso de Uso `RegistrarUsoPersonal` en Core Service
- **Archivo:** `internal/core/services/inventario_service.go`
- **Acción:**
  - Implementar la lógica de negocio para debitar stock por concepto de alimentación del personal:
    1. Validar `Cantidad > 0`.
    2. Convertir a unidad base mínima (`domain.ConvertirAUnidadBase`).
    3. Validar `Stock >= CantidadBase` (retornar `domain.ErrStockInsuficiente` si falta stock).
    4. Actualizar stock en `insumoRepo.UpdateStockAndCosto`.
    5. Crear movimiento inmutable con `referencia = "USO_PERSONAL: " + detalle`.
    6. Disparar notificación si `insumo.EstaBajoMinimo()`.
- **Criterio de Aceptación:** Test unitario que simula débito exitoso y rechazo por stock insuficiente.

### TASK-004 · Pruebas Unitarias del Núcleo de Inventario
- **Archivo:** `internal/core/services/inventario_service_test.go`
- **Acción:**
  - Añadir suite de tests para `RegistrarUsoPersonal` y `RegistrarMerma`:
    - Caso 1: Salida válida debita stock exacto.
    - Caso 2: Cantidad 0 o negativa retorna `ErrCantidadInvalida`.
    - Caso 3: Cantidad mayor al stock físico retorna `ErrStockInsuficiente`.
    - Caso 4: Registro audita costo unitario vigente del insumo.
- **Verificación:** `go test -v ./internal/core/services -run TestInventario` debe pasar 100% verde.

---

## FASE 2 — Backend: Adaptadores Outbound SQLite

### TASK-005 · Métodos de Consulta de Historial en Repositorios SQLite
- **Archivo:** `internal/adapters/outbound/sqlite/movimiento_repo.go` (o nuevo helper en adaptador)
- **Acción:**
  - Implementar consultas SQL optimizadas con `LIMIT`:
    ```sql
    -- Historial de Mermas
    SELECT m.id, m.insumo_id, i.nombre, m.cantidad, i.unidad_base,
           (m.cantidad * m.costo_unitario) AS costo_total, m.referencia, m.created_at
    FROM movimientos_inventario m
    JOIN insumos i ON i.id = m.insumo_id
    WHERE m.tipo = 'SALIDA_MERMA' AND m.referencia NOT LIKE 'USO_PERSONAL:%'
    ORDER BY m.created_at DESC LIMIT ?;

    -- Historial de Uso Personal
    SELECT m.id, m.insumo_id, i.nombre, m.cantidad, i.unidad_base,
           (m.cantidad * m.costo_unitario) AS costo_total, m.referencia, m.created_at
    FROM movimientos_inventario m
    JOIN insumos i ON i.id = m.insumo_id
    WHERE m.tipo = 'SALIDA_MERMA' AND m.referencia LIKE 'USO_PERSONAL:%'
    ORDER BY m.created_at DESC LIMIT ?;
    ```
- **Criterio de Aceptación:** Índices sobre `movimientos_inventario(tipo, created_at)` utilizados eficientemente (`EXPLAIN QUERY PLAN`).

---

## FASE 3 — Backend: Rutas y Handlers HTTP en el Servidor Admin

### TASK-006 · Endpoints HTTP para Mermas y Uso Personal
- **Archivos:** `internal/adapters/inbound/admin/routes.go` y `internal/adapters/inbound/admin/handlers.go`
- **Acción:**
  - Registrar nuevas rutas protegidas por middleware RBAC:
    - `POST /admin/api/mermas` (existente, asegurar retorno estructurado).
    - `GET  /admin/api/mermas/historial` (retorna JSON con lista de pérdidas).
    - `POST /admin/api/uso-personal` (procesa `UsoPersonalRequest`).
    - `GET  /admin/api/uso-personal/historial` (retorna JSON con consumos de personal).
  - Validar tokens CSRF y serialización de JSON con manejo de errores semánticos (400 Bad Request, 500 Internal Error).
- **Verificación:** Ejecutar peticiones curl de prueba contra endpoints de test.

---

## FASE 4 — Frontend: Estructura HTML y Diseño Visual

### TASK-007 · Maquetación de la Sub-navegación Superior (Sub-tabs)
- **Archivo:** `internal/adapters/inbound/admin/templates/admin.html`
- **Acción:**
  - Sustituir la cabecera simple de `#tab-inventario` por el contenedor de control segmentado con las 3 sub-pestañas:
    - `[📦 Inventario Principal]`
    - `[⚠️ Mermas]`
    - `[👨‍🍳 Uso Personal]`
  - Incluir contadores/badges de items en cada botón de pestaña.
  - Implementar botones de acción contextual (`+ Agregar Producto`, `+ Registrar Merma`, `+ Registrar Consumo Personal`).
- **Criterio de Aceptación:** La barra responde con diseño fluido en resoluciones de 320 px a 1920 px.

### TASK-008 · Maquetación de las Tres Tablas de Inventario
- **Archivo:** `internal/adapters/inbound/admin/templates/admin.html`
- **Acción:**
  - **Panel 1 (`#panel-inventario-principal`):** Tabla con las 8 columnas exactas (*Producto*, *Cantidad*, *Peso/Unidad*, *Stock Mínimo*, *Costo*, *Precio Final*, *Margen de Ganancias*, *Ajuste*).
  - **Panel 2 (`#panel-mermas`):** Tabla simplificada de 2 columnas (*Producto*, *Cantidad*) + columna de acciones/fecha.
  - **Panel 3 (`#panel-uso-personal`):** Tabla estructurada de 2 columnas (*Producto*, *Cantidad y Peso*) + columna de acciones/fecha.
- **Criterio de Aceptación:** Estructura limpia y semántica con IDs únicos para cada componente.

### TASK-009 · Estilos CSS y Tokens para Filas Inline y Sub-tabs
- **Archivo:** `internal/adapters/inbound/admin/static/admin.css`
- **Acción:**
  - Estilos para el control segmentado (`.segmented-control`, `.subtab-btn`, `.subtab-btn.active`).
  - Clases para filas en edición inline (`.fila-nueva`, `.fila-inline-editing`, `.fila-inline-merma`, `.fila-inline-personal`).
  - Animaciones sutiles para transiciones de pestañas (`fadeIn`, pulse de guardado).
- **Criterio de Aceptación:** No hay conflictos visuales con el tema oscuro/claro existente.

---

## FASE 5 — Frontend: Lógica JavaScript, Control de Vistas e Inline Editing

### TASK-010 · Controlador de Sub-pestañas y Persistencia de Sesión
- **Archivo:** `internal/adapters/inbound/admin/static/admin.js`
- **Acción:**
  - Función `cambiarSubtabInventario(subtabId)`:
    - Alterna paneles con clase `.hidden`.
    - Actualiza atributos `aria-selected` y estados visuales de los botones.
    - Persiste pestaña activa en `sessionStorage.setItem('pos_subtab_inventario_activa', subtabId)`.
    - Al abrir la pestaña de Inventario, lee `sessionStorage` y restaura la sub-pestaña seleccionada.
- **Criterio de Aceptación:** Al recargar la página o volver desde otra sección, se mantiene en la misma sub-pestaña.

### TASK-011 · Inserción y Edición en Línea de Inventario Principal (8 Columnas)
- **Archivo:** `internal/adapters/inbound/admin/static/admin.js`
- **Acción:**
  - Actualizar `agregarFilaNuevaInsumo()` y `renderizarFilaEditableHTML()` para mapear las 8 columnas completas.
  - Atajos de teclado: `[Enter]` guarda la fila; `[Escape]` cancela.
  - Recálculo en vivo de Margen (% y monto) y de valores en Bolívares (Costo Bs y Precio Bs) ante eventos `input`.
  - Enlace con `POST /admin/api/insumos` y actualización optimista de la tabla sin refrescar la página.
- **Criterio de Aceptación:** Se puede registrar un producto completo en menos de 5 segundos solo usando el teclado.

### TASK-012 · Inserción en Línea de Mermas (2 Columnas)
- **Archivo:** `internal/adapters/inbound/admin/static/admin.js`
- **Acción:**
  - Función `agregarFilaNuevaMerma()`:
    - Inserta fila al inicio de `#tbl-mermas` con selector de productos del catálogo.
    - Al seleccionar un insumo, muestra automáticamente su stock actual y unidad base mínima.
  - Función `guardarNuevaFilaMerma()`:
    - Envía `POST /admin/api/mermas`.
    - Actualiza el historial de mermas y descuenta el stock en el caché local de insumos para que la Pestaña 1 se actualice de inmediato.
- **Criterio de Aceptación:** Al guardar una merma, la cantidad se descuenta del stock visible en la Pestaña 1 sin recargar.

### TASK-013 · Inserción en Línea de Uso Personal (2 Columnas)
- **Archivo:** `internal/adapters/inbound/admin/static/admin.js`
- **Acción:**
  - Función `agregarFilaNuevaUsoPersonal()`:
    - Inserta fila al inicio de `#tbl-uso-personal` con selector de productos y campo numérico de cantidad y peso.
  - Función `guardarNuevaFilaPersonal()`:
    - Envía `POST /admin/api/uso-personal`.
    - Actualiza el historial de uso personal y descuenta el stock en el catálogo del Inventario Principal.
- **Criterio de Aceptación:** Descuento automático confirmado en el catálogo principal con feedback tipo Toast.

---

## FASE 6 — Verificación End-to-End, Compilación y Memoria

### TASK-014 · Suite de Pruebas y Análisis Estático
- **Acción:**
  - Ejecutar `go test -v ./...` y verificar 100% de tests aprobados.
  - Ejecutar `go vet ./...` para descartar advertencias de concurrencia o tipos.
- **Criterio de Aceptación:** Cero errores de compilación y cero linter warnings.

### TASK-015 · Compilación de Producción y Verificación de Tamaño de Binario
- **Acción:**
  - Ejecutar: `go build -ldflags="-s -w" -o bin/pos_core.exe ./cmd/main.go`
  - Validar que el binario resultante sea ligero (< 15 MB).
- **Criterio de Aceptación:** El ejecutable se genera correctamente en `bin/pos_core.exe`.

### TASK-016 · Actualización de `MEMORY.md`
- **Archivo:** `MEMORY.md`
- **Acción:**
  - Documentar la especificación 006 completada.
  - Registrar decisiones arquitectónicas (Estrategia de auditoría A, estructura de 8 columnas y flujos inline).
- **Criterio de Aceptación:** `MEMORY.md` actualizado en menos de 50 líneas.
