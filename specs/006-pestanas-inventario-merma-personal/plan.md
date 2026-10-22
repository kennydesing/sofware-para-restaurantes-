# Plan 006 — Arquitectura e Interfaz UI: Módulo de Inventario con Pestañas Superiores e Inserción en Línea (Inline Editing)

**Versión:** 1.0  
**Fecha:** 2026-10-21  
**Estado:** BORRADOR — Pendiente de aprobación  
**Referencia:** [spec.md](file:///c:/Users/admin/Desktop/sofware%20para%20restaurantes/specs/006-pestanas-inventario-merma-personal/spec.md)  
**Stack:** Go 1.22 · SQLite WAL · Tailwind CSS / Vanilla CSS moderno · JavaScript Vanilla ES2023+ · Arquitectura Hexagonal

---

## 1. Visión General de la Arquitectura

El rediseño del módulo de inventario mantiene la separación estricta de responsabilidades bajo la Arquitectura Hexagonal del proyecto:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           FRONTEND (ADMIN SPA)                                 │
│  admin.html  ───►  admin.js (Controlador de Vistas + Máquina de Estados)       │
│  Tablas Inline: [1. Inventario Principal] [2. Mermas] [3. Uso Personal]        │
└────────────────────────────────────────┬────────────────────────────────────────┘
                                         │ JSON / HTTP REST
                                         ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                       ADAPTADOR DE ENTRADA (INBOUND)                            │
│  internal/adapters/inbound/admin/                                               │
│    ├── routes.go               ◄── Registro de rutas (/api/mermas, etc.)        │
│    └── handlers.go             ◄── Handlers HTTP, decodificación y DTOs         │
└────────────────────────────────────────┬────────────────────────────────────────┘
                                         │ Invoca interfaces de dominio
                                         ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         NÚCLEO DE DOMINIO (CORE)                                │
│  internal/core/ports/inbound_ports.go                                           │
│    ├── InventarioService.RegistrarMerma(ctx, MermaRequest)                      │
│    └── InventarioService.RegistrarUsoPersonal(ctx, UsoPersonalRequest)          │
│  internal/core/services/inventario_service.go                                   │
│    └── Lógica de negocio pura, cálculo de merma y consumo base                  │
│  internal/core/domain/                                                          │
│    └── insumo.go, movimiento.go, errors.go                                      │
└────────────────────────────────────────┬────────────────────────────────────────┘
                                         │ Implementa outbound ports
                                         ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                      ADAPTADOR DE SALIDA (OUTBOUND)                             │
│  internal/adapters/outbound/sqlite/                                             │
│    ├── insumo_repo.go          ◄── Actualización de stock y consulta catalogo   │
│    ├── movimiento_repo.go      ◄── Inserción inmutable en libro diario          │
│    └── atomic_debitor.go       ◄── Débitos atómicos transaccionales WAL         │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Frontend / Templates & UI/UX

### 2.1 Estructura HTML y Jerarquía de Componentes

La sección de Inventario (`#tab-inventario` en `admin.html`) se reestructura para incorporar la barra de sub-pestañas superiores segmentadas y tres paneles de contenido independientes:

```html
<!-- Sub-navegación Superior del Módulo de Inventario -->
<div class="inventory-subnav-container mb-6 flex flex-wrap items-center justify-between gap-4 border-b border-slate-700/60 pb-3">
  
  <!-- Segmented Control Tabs (Pestañas Superiores) -->
  <div class="segmented-control inline-flex rounded-xl bg-slate-900/80 p-1.5 border border-slate-700/50 shadow-inner" role="tablist" aria-label="Secciones de Inventario">
    <button type="button" 
            class="subtab-btn active px-4 py-2 rounded-lg text-sm font-semibold transition-all duration-200 flex items-center gap-2 text-white bg-blue-600 shadow-md"
            data-subtab="inventario-principal"
            role="tab"
            aria-selected="true"
            aria-controls="panel-inventario-principal"
            onclick="cambiarSubtabInventario('inventario-principal')">
      <span>📦</span>
      <span>Inventario Principal</span>
      <span id="badge-total-items" class="ml-1.5 px-2 py-0.5 text-xs rounded-full bg-slate-800 text-blue-200 font-mono">0</span>
    </button>

    <button type="button" 
            class="subtab-btn px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 flex items-center gap-2 text-slate-400 hover:text-slate-200 hover:bg-slate-800/50"
            data-subtab="mermas"
            role="tab"
            aria-selected="false"
            aria-controls="panel-mermas"
            onclick="cambiarSubtabInventario('mermas')">
      <span>⚠️</span>
      <span>Mermas (Pérdidas)</span>
      <span id="badge-mermas-hoy" class="ml-1.5 px-2 py-0.5 text-xs rounded-full bg-rose-950/60 text-rose-300 font-mono">0</span>
    </button>

    <button type="button" 
            class="subtab-btn px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 flex items-center gap-2 text-slate-400 hover:text-slate-200 hover:bg-slate-800/50"
            data-subtab="uso-personal"
            role="tab"
            aria-selected="false"
            aria-controls="panel-uso-personal"
            onclick="cambiarSubtabInventario('uso-personal')">
      <span>👨‍🍳</span>
      <span>Uso Personal</span>
      <span id="badge-personal-hoy" class="ml-1.5 px-2 py-0.5 text-xs rounded-full bg-amber-950/60 text-amber-300 font-mono">0</span>
    </button>
  </div>

  <!-- Botones de Acción Rápida Contextual -->
  <div class="subtab-actions flex items-center gap-3">
    <!-- Acción para Pestaña 1 -->
    <div id="actions-inventario-principal" class="subtab-action-group flex items-center gap-2">
      <button class="btn btn-success flex items-center gap-2 font-semibold shadow-lg shadow-emerald-950/40" 
              onclick="agregarFilaNuevaInsumo()">
        <span>+</span> <span>Agregar Producto</span>
      </button>
      <button id="btn-toggle-sonido" class="btn btn-secondary btn-sm p-2 text-base" 
              onclick="toggleAlertaSonido()" title="Activar/Silenciar alerta sonora">🔔</button>
    </div>

    <!-- Acción para Pestaña 2 -->
    <div id="actions-mermas" class="subtab-action-group hidden flex items-center gap-2">
      <button class="btn btn-danger flex items-center gap-2 font-semibold shadow-lg shadow-rose-950/40" 
              onclick="agregarFilaNuevaMerma()">
        <span>⚠️</span> <span>Registrar Merma</span>
      </button>
    </div>

    <!-- Acción para Pestaña 3 -->
    <div id="actions-uso-personal" class="subtab-action-group hidden flex items-center gap-2">
      <button class="btn btn-primary flex items-center gap-2 font-semibold shadow-lg shadow-blue-950/40" 
              onclick="agregarFilaNuevaUsoPersonal()">
        <span>🍽️</span> <span>Registrar Consumo Personal</span>
      </button>
    </div>
  </div>
</div>
```

---

### 2.2 Estructura de las Tres Tablas e Inputs Integrados

#### A. Tabla 1: Inventario Principal (8 Columnas Definitivas)
```html
<div id="panel-inventario-principal" class="subtab-panel" role="tabpanel">
  <div class="table-responsive rounded-xl border border-slate-700/60 shadow-xl overflow-hidden bg-slate-900/60">
    <table id="tabla-insumos-v2" class="w-full text-left text-sm text-slate-200">
      <thead class="bg-slate-800/90 text-xs uppercase tracking-wider text-slate-400 border-b border-slate-700">
        <tr>
          <th class="col-producto p-3.5">Producto</th>
          <th class="col-cantidad p-3.5 text-right">Cantidad</th>
          <th class="col-unidad p-3.5 text-center">Peso / Unidad</th>
          <th class="col-stock-minimo p-3.5 text-right">Stock Mínimo</th>
          <th class="col-costo p-3.5 text-right">Costo ($ / Bs)</th>
          <th class="col-precio p-3.5 text-right">Precio Final ($ / Bs)</th>
          <th class="col-margen p-3.5 text-center">Margen de Ganancias</th>
          <th class="col-ajuste p-3.5 text-center">Ajuste</th>
        </tr>
      </thead>
      <tbody id="tbl-insumos" class="divide-y divide-slate-800">
        <!-- Renderizado dinámico vía JS -->
      </tbody>
    </table>
  </div>
</div>
```

**Estructura de la fila editable inline para Inventario Principal:**
```html
<tr class="fila-nueva fila-inline-editing bg-blue-950/20 border-l-4 border-blue-500 transition-colors" data-insumo-id="nuevo">
  <!-- 1. Producto -->
  <td class="p-2.5">
    <input type="text" class="form-control w-full bg-slate-800 border-slate-600 rounded px-2.5 py-1.5 text-sm font-medium focus:border-blue-500 focus:ring-1 focus:ring-blue-500" placeholder="Nombre del producto..." required />
  </td>
  <!-- 2. Cantidad (Stock) -->
  <td class="p-2.5 text-right">
    <input type="number" step="0.01" class="form-control w-28 text-right bg-slate-800 border-slate-600 rounded px-2 py-1.5 text-sm font-mono" placeholder="0.00" required />
  </td>
  <!-- 3. Peso / Unidad -->
  <td class="p-2.5 text-center">
    <select class="form-control bg-slate-800 border-slate-600 rounded px-2 py-1.5 text-sm font-semibold text-blue-300">
      <option value="g">g (Gramos)</option>
      <option value="ml">ml (Mililitros)</option>
      <option value="u" selected>u (Unidades)</option>
    </select>
  </td>
  <!-- 4. Stock Mínimo -->
  <td class="p-2.5 text-right">
    <input type="number" step="0.01" class="form-control w-24 text-right bg-slate-800 border-slate-600 rounded px-2 py-1.5 text-sm font-mono" placeholder="0.00" required />
  </td>
  <!-- 5. Costo (USD + Dinámico Bs) -->
  <td class="p-2.5 text-right">
    <input type="number" step="0.01" class="form-control w-24 text-right bg-slate-800 border-slate-600 rounded px-2 py-1.5 text-sm font-mono" placeholder="0.00" />
    <div class="costo-bs-inline text-xs text-slate-400 mt-1 font-mono">Bs 0.00</div>
  </td>
  <!-- 6. Precio Final (USD + Dinámico Bs) -->
  <td class="p-2.5 text-right">
    <input type="number" step="0.01" class="form-control w-24 text-right bg-slate-800 border-slate-600 rounded px-2 py-1.5 text-sm font-mono" placeholder="0.00" />
    <div class="precio-bs-inline text-xs text-slate-400 mt-1 font-mono">Bs 0.00</div>
  </td>
  <!-- 7. Margen de Ganancias -->
  <td class="p-2.5 text-center">
    <div class="margen-preview text-xs text-slate-400 font-semibold">—</div>
  </td>
  <!-- 8. Ajuste (Botones inline) -->
  <td class="p-2.5 text-center">
    <div class="flex items-center justify-center gap-1.5">
      <button class="btn btn-sm btn-success px-2.5 py-1 text-xs font-semibold rounded bg-emerald-600 hover:bg-emerald-500 text-white shadow" onclick="guardarNuevaFilaInsumo(this)">💾 Guardar</button>
      <button class="btn btn-sm btn-secondary px-2.5 py-1 text-xs rounded bg-slate-700 hover:bg-slate-600 text-slate-300" onclick="cancelarNuevaFilaInsumo(this)">❌ Cancelar</button>
    </div>
  </td>
</tr>
```

---

#### B. Tabla 2: Mermas (2 Columnas Simplificadas)
```html
<div id="panel-mermas" class="subtab-panel hidden" role="tabpanel">
  <div class="mb-4 p-3 bg-rose-950/30 border border-rose-800/40 rounded-xl text-xs text-rose-200 flex items-center justify-between">
    <div class="flex items-center gap-2">
      <span class="text-base">⚠️</span>
      <span><strong>Registro Exclusivo de Mermas:</strong> Las cantidades registradas se descuentan inmediatamente del inventario general y quedan asentadas en el Libro Diario de Auditoría.</span>
    </div>
    <div class="text-right font-mono" id="stat-resumen-mermas">Total Mermas Hoy: <strong>0.00</strong></div>
  </div>

  <div class="table-responsive rounded-xl border border-slate-700/60 shadow-xl overflow-hidden bg-slate-900/60">
    <table id="tabla-mermas" class="w-full text-left text-sm text-slate-200">
      <thead class="bg-slate-800/90 text-xs uppercase tracking-wider text-slate-400 border-b border-slate-700">
        <tr>
          <th class="p-3.5 w-2/3">1. Producto</th>
          <th class="p-3.5 w-1/3 text-right">2. Cantidad</th>
          <th class="p-3.5 w-28 text-center">Acción</th>
        </tr>
      </thead>
      <tbody id="tbl-mermas" class="divide-y divide-slate-800">
        <!-- Historial y filas inline de merma -->
      </tbody>
    </table>
  </div>
</div>
```

**Fila editable inline para Mermas:**
```html
<tr class="fila-nueva fila-inline-merma bg-rose-950/20 border-l-4 border-rose-500" data-merma-id="nuevo">
  <td class="p-2.5">
    <select class="form-control w-full bg-slate-800 border-slate-600 rounded px-3 py-1.5 text-sm select-insumo-merma" required>
      <option value="">-- Seleccionar Producto del Almacén --</option>
      <!-- Opciones pobladas dinámicamente con stock disponible -->
    </select>
  </td>
  <td class="p-2.5 text-right">
    <div class="inline-flex items-center gap-2 justify-end">
      <input type="number" step="0.01" min="0.01" class="form-control w-32 text-right bg-slate-800 border-slate-600 rounded px-2.5 py-1.5 text-sm font-mono" placeholder="0.00" required />
      <span class="unidad-medida-indicador text-xs font-semibold text-slate-400 w-10 text-left">--</span>
    </div>
  </td>
  <td class="p-2.5 text-center">
    <div class="flex items-center justify-center gap-1.5">
      <button class="btn btn-sm btn-danger px-2.5 py-1 text-xs font-semibold rounded bg-rose-600 hover:bg-rose-500 text-white shadow" onclick="guardarNuevaFilaMerma(this)">💾 Guardar</button>
      <button class="btn btn-sm btn-secondary px-2.5 py-1 text-xs rounded bg-slate-700 hover:bg-slate-600 text-slate-300" onclick="cancelarNuevaFilaMerma(this)">❌ Cancelar</button>
    </div>
  </td>
</tr>
```

---

#### C. Tabla 3: Uso Personal (2 Columnas Estructuradas)
```html
<div id="panel-uso-personal" class="subtab-panel hidden" role="tabpanel">
  <div class="mb-4 p-3 bg-amber-950/30 border border-amber-800/40 rounded-xl text-xs text-amber-200 flex items-center justify-between">
    <div class="flex items-center gap-2">
      <span class="text-base">👨‍🍳</span>
      <span><strong>Alimentación del Personal:</strong> Registre los insumos consumidos para las comidas del equipo. El inventario principal se debitará automáticamente.</span>
    </div>
    <div class="text-right font-mono" id="stat-resumen-personal">Consumo Personal Hoy: <strong>0 ítems</strong></div>
  </div>

  <div class="table-responsive rounded-xl border border-slate-700/60 shadow-xl overflow-hidden bg-slate-900/60">
    <table id="tabla-uso-personal" class="w-full text-left text-sm text-slate-200">
      <thead class="bg-slate-800/90 text-xs uppercase tracking-wider text-slate-400 border-b border-slate-700">
        <tr>
          <th class="p-3.5 w-2/3">1. Producto</th>
          <th class="p-3.5 w-1/3 text-right">2. Cantidad y Peso</th>
          <th class="p-3.5 w-28 text-center">Acción</th>
        </tr>
      </thead>
      <tbody id="tbl-uso-personal" class="divide-y divide-slate-800">
        <!-- Historial y filas inline de uso personal -->
      </tbody>
    </table>
  </div>
</div>
```

**Fila editable inline para Uso Personal:**
```html
<tr class="fila-nueva fila-inline-personal bg-amber-950/20 border-l-4 border-amber-500" data-personal-id="nuevo">
  <td class="p-2.5">
    <select class="form-control w-full bg-slate-800 border-slate-600 rounded px-3 py-1.5 text-sm select-insumo-personal" required>
      <option value="">-- Seleccionar Insumo para Cocina de Personal --</option>
    </select>
  </td>
  <td class="p-2.5 text-right">
    <div class="inline-flex items-center gap-2 justify-end">
      <input type="number" step="0.01" min="0.01" class="form-control w-32 text-right bg-slate-800 border-slate-600 rounded px-2.5 py-1.5 text-sm font-mono" placeholder="0.00" required />
      <span class="unidad-medida-indicador text-xs font-semibold text-slate-400 w-10 text-left">--</span>
    </div>
  </td>
  <td class="p-2.5 text-center">
    <div class="flex items-center justify-center gap-1.5">
      <button class="btn btn-sm btn-primary px-2.5 py-1 text-xs font-semibold rounded bg-amber-600 hover:bg-amber-500 text-white shadow" onclick="guardarNuevaFilaPersonal(this)">💾 Guardar</button>
      <button class="btn btn-sm btn-secondary px-2.5 py-1 text-xs rounded bg-slate-700 hover:bg-slate-600 text-slate-300" onclick="cancelarNuevaFilaPersonal(this)">❌ Cancelar</button>
    </div>
  </td>
</tr>
```

---

### 2.3 Manejo de Estados en JavaScript (Arquitectura de Cliente)

El módulo utiliza un controlador de estados en Vanilla JS modular:

```javascript
// admin.js — Estado y Ciclo de Vida de las Sub-pestañas
const InventoryState = {
  activeSubtab: 'inventario-principal',
  insumosCache: [],
  mermasHistory: [],
  personalHistory: [],
  editingRows: new Set(), // IDs de filas actualmente en edición

  setSubtab(subtabId) {
    if (this.editingRows.size > 0) {
      if (!confirm("Tiene cambios sin guardar en una fila abierta. ¿Desea cambiar de pestaña?")) {
        return;
      }
      this.cancelAllEditing();
    }
    this.activeSubtab = subtabId;
    sessionStorage.setItem('pos_subtab_inventario_activa', subtabId);
    this.renderSubtab();
  },

  renderSubtab() {
    // 1. Actualizar botones de sub-pestaña
    document.querySelectorAll('.subtab-btn').forEach(btn => {
      const isActive = btn.getAttribute('data-subtab') === this.activeSubtab;
      btn.classList.toggle('active', isActive);
      btn.classList.toggle('bg-blue-600', isActive);
      btn.classList.toggle('text-white', isActive);
      btn.classList.toggle('text-slate-400', !isActive);
      btn.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });

    // 2. Alternar visibilidad de paneles
    document.querySelectorAll('.subtab-panel').forEach(panel => {
      panel.classList.add('hidden');
    });
    const activePanel = document.getElementById(`panel-${this.activeSubtab}`);
    if (activePanel) activePanel.classList.remove('hidden');

    // 3. Alternar botones de acción superior
    document.querySelectorAll('.subtab-action-group').forEach(group => group.classList.add('hidden'));
    const activeActions = document.getElementById(`actions-${this.activeSubtab}`);
    if (activeActions) activeActions.classList.remove('hidden');

    // 4. Cargar datos si no están en memoria
    if (this.activeSubtab === 'mermas') cargarHistorialMermas();
    if (this.activeSubtab === 'uso-personal') cargarHistorialUsoPersonal();
  }
};
```

---

## 3. Backend / Adaptadores en Go

### 3.1 Endpoints REST en el Adaptador Inbound Admin

Se implementan y adaptan las siguientes rutas en `internal/adapters/inbound/admin/routes.go`:

| Método | Endpoint | Caso de Uso en Dominio | Descripción |
|---|---|---|---|
| `GET` | `/admin/api/insumos` | `inventarioSvc.ListarInsumos` | Lista de insumos con stock, costo, precio de venta y estado mínimo. |
| `POST` | `/admin/api/insumos` | Inserción directa en `insumoRepo` | Crea nuevo insumo desde la fila inline del Inventario Principal. |
| `PUT` | `/admin/api/insumos/{id}` | Edición en `insumoRepo` | Actualiza nombre, costo, precio, stock mínimo o stock del insumo. |
| `DELETE` | `/admin/api/insumos/{id}` | Eliminación en `insumoRepo` | Elimina insumo (en cascada limpia referencias en modo test). |
| `POST` | `/admin/api/mermas` | `inventarioSvc.RegistrarMerma` | Débito por pérdida operativa y registro de `SALIDA_MERMA`. |
| `GET` | `/admin/api/mermas/historial` | `movimientoRepo.ListarPorTipo("SALIDA_MERMA")` | Historial de mermas con insumo, cantidad, costo y fecha. |
| `POST` | `/admin/api/uso-personal` | `inventarioSvc.RegistrarUsoPersonal` | Débito por alimentación de personal y registro de auditoría. |
| `GET` | `/admin/api/uso-personal/historial` | `movimientoRepo.ListarPorReferenciaPrefix("USO_PERSONAL:")` | Historial de consumo de insumos por el personal. |

### 3.2 DTOs de Entrada y Salida (Package ports)

En `internal/core/ports/inbound_ports.go`:

```go
// UsoPersonalRequest contiene los datos para imputar consumo de personal.
type UsoPersonalRequest struct {
    InsumoID     int     `json:"insumo_id"`
    Cantidad     float64 `json:"cantidad"`       // Magnitud en unidad física
    UnidadMedida string  `json:"unidad_medida"`  // Opcional: si difiere de unidad base
    Detalle      string  `json:"detalle"`        // "Comida de turno", "Desayuno personal", etc.
}

// HistorialSalidaDTO estructura la respuesta de lectura para Mermas y Uso Personal.
type HistorialSalidaDTO struct {
    ID            int       `json:"id"`
    InsumoID      int       `json:"insumo_id"`
    InsumoNombre  string    `json:"insumo_nombre"`
    Cantidad      float64   `json:"cantidad"`
    UnidadBase    string    `json:"unidad_base"`
    CostoTotal    float64   `json:"costo_total"`
    Referencia    string    `json:"referencia"`
    CreatedAt     time.Time `json:"created_at"`
}
```

### 3.3 Casos de Uso en el Servicio de Inventario (`InventarioService`)

Ampliación de `internal/core/ports/inbound_ports.go`:
```go
type InventarioService interface {
    // Métodos existentes...
    RegistrarCompra(ctx context.Context, req CompraRequest) (*domain.Insumo, error)
    RegistrarMerma(ctx context.Context, req MermaRequest) (*domain.MovimientoInventario, error)
    AjustarStock(ctx context.Context, req AjusteRequest) (*domain.MovimientoInventario, error)
    ListarInsumos(ctx context.Context) ([]*domain.Insumo, error)
    
    // NUEVO: RegistrarUsoPersonal
    RegistrarUsoPersonal(ctx context.Context, req UsoPersonalRequest) (*domain.MovimientoInventario, error)
    
    // NUEVO: Consultas de auditoría específica
    ListarHistorialMermas(ctx context.Context, limite int) ([]*HistorialSalidaDTO, error)
    ListarHistorialUsoPersonal(ctx context.Context, limite int) ([]*HistorialSalidaDTO, error)
}
```

### 3.4 Estrategia de Auditoría y Persistencia SQLite

Para respetar la restricción `CHECK` de `movimientos_inventario`:
```sql
CHECK (tipo IN ('ENTRADA_COMPRA', 'SALIDA_VENTA', 'SALIDA_MERMA', 'AJUSTE_MANUAL'))
```
Se adoptan dos opciones técnicas:

- **Estrategia Recomendada (Cero alteraciones de DDL en migrations):**
  - Mermas se guardan con `tipo: SALIDA_MERMA` y `referencia: "MERMA: <motivo>"`.
  - Uso Personal se registra con `tipo: SALIDA_MERMA` y `referencia: "USO_PERSONAL: <detalle>"`.
  - Esto garantiza **100% de retrocompatibilidad** con el esquema SQLite actual sin requerir una migración destructiva sobre la tabla `movimientos_inventario` (que en SQLite exigiría recrear la tabla con índices y FKs).
  
- **Estrategia Alternativa (Migración Formal):**
  - Si se prefiere un tipo explícito `SALIDA_USO_PERSONAL`, se planifica la migración recreando la tabla o actualizando el check. La especificación documenta ambas y recomienda la Estrategia A para máxima estabilidad y despliegue sin riesgo.

### 3.5 Transaccionalidad Atómica del Débito

Toda salida de stock por Merma o Uso Personal ejecuta el siguiente flujo en `InventarioServiceImpl`:

```go
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
        return nil, fmt.Errorf("error al convertir unidad: %w", err)
    }

    // Validación de stock: no permitir salidas que superen el disponible
    if insumo.Stock < cantidadBase {
        return nil, domain.ErrStockInsuficiente
    }

    nuevoStock := insumo.Stock - cantidadBase
    if err := s.insumoRepo.UpdateStockAndCosto(ctx, insumo.ID, nuevoStock, insumo.CostoPromedio); err != nil {
        return nil, fmt.Errorf("error al actualizar stock: %w", err)
    }
    insumo.Stock = nuevoStock

    detalle := req.Detalle
    if detalle == "" {
        detalle = "Consumo de personal de cocina/sala"
    }

    mov := &domain.MovimientoInventario{
        InsumoID:      insumo.ID,
        Tipo:          domain.SalidaMerma, // Estrategia A
        Cantidad:      cantidadBase,
        CostoUnitario: insumo.CostoPromedio,
        Referencia:    fmt.Sprintf("USO_PERSONAL: %s", detalle),
    }

    if err := s.movimientoRepo.Create(ctx, mov); err != nil {
        return nil, fmt.Errorf("error al asentar movimiento: %w", err)
    }

    // Notificación reactiva si cae bajo mínimo
    if insumo.EstaBajoMinimo() && s.notifier != nil {
        _ = s.notifier.NotificarStockBajo(ctx, insumo)
    }

    return mov, nil
}
```

---

## 4. Diseño UI/UX y Sistema de Tokens

### 4.1 Paleta Semántica
- **Inventario Principal:** Acento Azul (`#3b82f6` / `rgb(59 130 246)`). Representa el catálogo maestro y la estabilidad de almacén.
- **Mermas:** Acento Rojo/Rosa (`#ef4444` / `rgb(239 68 68)`). Denota alerta, pérdida y descarte.
- **Uso Personal:** Acento Ámbar/Dorado (`#f59e0b` / `rgb(245 154 11)`). Representa la cocina interna, alimentación y cuidado del equipo.

### 4.2 Estados de la Fila Editable
```
[ Modo Lectura (Hover sutil) ]
       │
       ├─ Clic "+ Agregar" ───► [ Modo Inserción (Borde acento izquierdo, inputs activos, autofocus) ]
       ├─ Clic "✏️ Editar"   ───► [ Modo Edición (Inputs precargados con valor actual) ]
       │
       ├── Tecla [Escape] / "❌ Cancelar" ───► [ Fila restaurada / Fila temporal eliminada ]
       └── Tecla [Enter]  / "💾 Guardar"   ───► [ Estado "Guardando..." (Disabled + Spinner) ]
                                                       │
                                      ┌────────────────┴────────────────┐
                                      ▼                                 ▼
                              [ Éxito: Toast verde              [ Error: Toast rojo
                               + animación fade-in ]             + foco en campo inválido ]
```

---

## 5. Medidas de Rendimiento y Seguridad

1. **Rendimiento UI:** Renderizado eficiente de tablas con fragmentos de documento (`DocumentFragment`) en JavaScript para evitar reflujos innecesarios del DOM (*layout thrashing*).
2. **Protección CSRF:** Toda mutación HTTP (`POST`, `PUT`, `DELETE`) incluye el header `X-CSRF-Token` extraído de `<meta name="csrf-token">`.
3. **Control de Concurrencia:** SQLite WAL permite lectores concurrentes sin bloquear escrituras, previniendo cuellos de botella durante picos de venta en el POS.
