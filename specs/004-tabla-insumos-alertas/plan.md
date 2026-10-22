# Plan 004 — Arquitectura e Interfaz UI

**Versión:** 1.0  
**Fecha:** 2026-10-17  
**Estado:** BORRADOR — Pendiente de aprobación  
**Referencia:** specs/004-tabla-insumos-alertas/spec.md

---

## 1. Visión General de la Arquitectura

La implementación sigue el patrón hexagonal existente. Todos los cambios respetan las capas:

```
Frontend (admin.html + admin.js + admin.css)
    │  HTTP REST
    ▼
internal/adapters/inbound/admin/
    │  handlers.go           ← agregar campo merma_acumulada en response + DTO
    │  templates/admin.html  ← nueva tabla 8 columnas + banner autoplay + btn toggle
    │  static/admin.js       ← renderizado dual USD/Bs, alarma sonora, evento tasaActualizada
    │  static/admin.css      ← estilos .row-critica, .badge-critical, .badge-ok, @keyframes pulse
    ▼
internal/adapters/outbound/sqlite/
    │  insumo_repo.go        ← GetAllEnriquecido() con LEFT JOIN merma
    ▼
migrations/
    └── 000005_precio_venta_insumo.up.sql  ← NUEVO (requiere aprobación)
```

---

## 2. Backend / Adaptadores Go

### 2.1 Migración SQL — 000005_precio_venta_insumo.up.sql

> ⚠️ Requiere aprobación antes de escribir o aplicar.

```sql
-- Migración 000005: precio de venta base en USD por insumo.
ALTER TABLE insumos ADD COLUMN precio_venta_usd REAL DEFAULT 0.0;
```

### 2.2 SQL — GetAllEnriquecido()

Consulta SQL para el método GetAllEnriquecido en insumo_repo.go:

```sql
SELECT
    i.id,
    i.nombre,
    i.unidad_base,
    i.costo_promedio,
    COALESCE(i.precio_venta_usd, 0.0)                                          AS precio_venta_usd,
    i.stock_minimo,
    i.stock,
    COALESCE(SUM(CASE WHEN m.tipo_movimiento = 'SALIDA_MERMA'
                      THEN m.cantidad ELSE 0 END), 0.0)                         AS merma_acumulada
FROM insumos i
LEFT JOIN movimientos_inventario m ON m.insumo_id = i.id
GROUP BY i.id
ORDER BY i.nombre ASC;
```

### 2.3 DTO de Respuesta (adaptador inbound — sin contaminar el dominio)

```go
// internal/adapters/inbound/admin/handlers.go

// InsumoResponseDTO es el DTO de salida para la vista admin.
// Extiende los datos de dominio con campos calculados sin modificar el core.
type InsumoResponseDTO struct {
    ID             int     `json:"id"`
    Nombre         string  `json:"nombre"`
    UnidadBase     string  `json:"unidad_base"`
    CostoPromedio  float64 `json:"costo_promedio"`
    PrecioVentaUSD float64 `json:"precio_venta_usd"`
    StockMinimo    float64 `json:"stock_minimo"`
    Stock          float64 `json:"stock"`
    MermaAcumulada float64 `json:"merma_acumulada"`
}
```

### 2.4 Puerto de Salida (outbound_ports.go)

> ⚠️ Requiere aprobación: modificar InsumoRepository.

Agregar método al puerto:
```go
// GetAllEnriquecido retorna todos los insumos con merma acumulada y precio de venta.
// Implementado en adapters/outbound/sqlite/insumo_repo.go.
GetAllEnriquecido(ctx context.Context) ([]*InsumoEnriquecido, error)
```

Donde InsumoEnriquecido (en paquete ports, no domain):
```go
type InsumoEnriquecido struct {
    domain.Insumo
    PrecioVentaUSD float64
    MermaAcumulada float64
}
```

---

## 3. Frontend / Templates

### 3.1 HTML — Nueva Tabla de 8 Columnas (admin.html)

Estructura del header de la sección con toggle de sonido:

```html
<div class="section-header">
  <h2 class="section-title">📦 Catálogo de Insumos & Entradas</h2>
  <div style="display:flex; gap:0.5rem; flex-wrap:wrap; align-items:center;">
    <button class="btn btn-success"    onclick="abrirModalCompra()">📥 + Registrar Entrada</button>
    <button class="btn btn-warning"    onclick="abrirModalMerma()">⚠️ - Registrar Merma</button>
    <button class="btn btn-secondary"  onclick="abrirModalAjuste()">⚙ Ajuste Directo</button>
    <button class="btn"                onclick="abrirModalNuevoInsumo()">📦 + Nuevo Insumo</button>
    <!-- NUEVO: Toggle alerta sonora -->
    <button id="btn-toggle-sonido" class="btn btn-secondary btn-sm"
            onclick="toggleAlertaSonido()" title="Activar/Silenciar alerta sonora"
            style="font-size:1.1rem; padding:4px 10px;">🔔</button>
  </div>
</div>

<!-- Banner para política de autoplay -->
<div id="banner-autoplay" role="alert">
  🔔 Hay ítems en stock crítico. Haz clic en cualquier parte para activar la alerta sonora.
</div>

<div class="table-responsive">
  <table id="tabla-insumos-v2" class="table-insumos">
    <thead>
      <tr>
        <th class="col-producto">Producto</th>
        <th class="col-stock-almacen">Stock Almacén</th>
        <th class="col-stock-minimo">Stock Mínimo</th>
        <th class="col-costo">Costo Prod.</th>
        <th class="col-venta">P. Venta</th>
        <th class="col-margen">Margen</th>
        <th class="col-merma">Merma</th>
        <th class="col-ajustes">Ajustes</th>
      </tr>
    </thead>
    <tbody id="tbl-insumos">
      <tr><td colspan="8" class="empty-state">Cargando insumos...</td></tr>
    </tbody>
  </table>
</div>
```

### 3.2 JS — renderizarInsumos() v2

La función nueva acepta (insumos, tasaUSD) y genera HTML para las 8 columnas.
Funciones auxiliares necesarias:
- formatStock(cantidad, unidad)  → "2,000 ml" | "15.00 kg" | "48 u"
- formatBs(monto)                → "46,500.00" (separador de miles coma)
- fmtUSD(monto)                  → "3.50"
- escHtml(str)                   → escapa caracteres HTML

Lógica por fila:
1. Clasificar esCritico = stock <= stock_minimo
2. Calcular Bs de costo = costo_promedio * tasaUSD
3. Calcular Bs de venta = precio_venta_usd * tasaUSD
4. Calcular margenPct = ((venta - costo) / venta) * 100 si venta > 0
5. Aplicar class "row-critica" y badge "⚠️ Stock Crítico" si es crítico
6. Emitir dispararAlertaSonora() si hayStockCritico

### 3.3 JS — Evento tasaActualizada para recálculo masivo de Bs

```javascript
// Escucha global del evento de nueva tasa
document.addEventListener('tasaActualizada', function(e) {
  var tasa = e.detail.tasa;
  document.querySelectorAll('.bs-value[data-usd]').forEach(function(el) {
    var usd = parseFloat(el.getAttribute('data-usd')) || 0;
    el.textContent = 'Bs ' + formatBs(usd * tasa);
  });
});

// sincronizarTasasBCV() emite el evento tras respuesta exitosa:
document.dispatchEvent(new CustomEvent('tasaActualizada', { detail: { tasa: nuevaTasaUSD } }));
```

### 3.4 JS — Alerta Sonora con Web Audio API (sin archivos externos)

```javascript
var alertaSonidoActiva = localStorage.getItem('pos_alerta_sonido') !== 'false';
var paginaInteractuada = false;
document.addEventListener('click', function() { paginaInteractuada = true; }, { once: true });

function toggleAlertaSonido() {
  alertaSonidoActiva = !alertaSonidoActiva;
  localStorage.setItem('pos_alerta_sonido', String(alertaSonidoActiva));
  var btn = document.getElementById('btn-toggle-sonido');
  btn.textContent = alertaSonidoActiva ? '🔔' : '🔕';
}

var alarmaDisparada = false;
function dispararAlertaSonora() {
  if (alarmaDisparada || !alertaSonidoActiva) return;
  alarmaDisparada = true;
  setTimeout(function() { alarmaDisparada = false; }, 30000);
  if (!paginaInteractuada) { mostrarBannerAutoplay(); return; }
  try {
    var ctx  = new (window.AudioContext || window.webkitAudioContext)();
    var osc  = ctx.createOscillator();
    var gain = ctx.createGain();
    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.type = 'sine';
    osc.frequency.setValueAtTime(880, ctx.currentTime);
    gain.gain.setValueAtTime(0.4, ctx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.4);
    osc.start(ctx.currentTime);
    osc.stop(ctx.currentTime + 0.4);
    osc.onended = function() { ctx.close(); };
  } catch(err) { console.warn('[POS] Alerta sonora no disponible:', err); }
}
```

### 3.5 CSS — Estilos para la Tabla v2 (admin.css)

Nuevos selectores a agregar:
- .row-critica            → fondo rgba(239,68,68,0.08) + borde izquierdo rojo 3px
- .badge-critical         → rojo con animación pulse-critical
- .badge-ok               → verde suave
- @keyframes pulse-critical → glow 2s infinite
- .usd-value              → texto bold, color primario
- .bs-value               → texto pequeño verde muted
- .margen-badge           → pill coloreado según escala
- .margen-verde/.margen-ambar/.margen-rojo
- .margen-monto           → texto muted debajo del badge
- .ajustes-btns           → flex row con gap 4px
- .btn-icon               → botón icono hover state
- .btn-icon-danger        → hover rojo
- .unidad-tag             → chip de unidad base
- #banner-autoplay        → alerta amarilla display:none por defecto
- .valor-nd               → muted italic para "—"
- @media max-width:768px  → table-insumos min-width:900px

---

## 4. Flujo de Datos Completo

```
Browser: GET /admin/api/insumos
    → Go handler llama a InsumoRepo.GetAllEnriquecido()
    → SQL: SELECT + LEFT JOIN movimientos_inventario GROUP BY i.id
    → Response: JSON array de InsumoResponseDTO

admin.js: cargarInsumos()
    → GET /api/tasas (tasa USD actual)
    → renderizarInsumos(insumos, tasaUSD)
        → por cada insumo: formatear, calcular Bs, margen, clasificar fila
        → actualizar DOM: tbody#tbl-insumos
        → actualizar #stat-stock-bajo en Dashboard
        → si hayStockCritico: dispararAlertaSonora()

Cambio de tasa (BCV sync / manual):
    → POST /api/tasas/sincronizar o PUT /api/tasas/manual
    → JS emite CustomEvent('tasaActualizada', { detail: { tasa: X } })
    → Listener: actualiza todos los span.bs-value[data-usd] en el DOM
```

---

## 5. Archivos a Modificar

| Archivo | Cambio |
|---------|--------|
| migrations/000005_precio_venta_insumo.up.sql | NUEVO (requiere aprobación) |
| internal/core/ports/outbound_ports.go | Agregar GetAllEnriquecido + InsumoEnriquecido (requiere aprobación) |
| internal/adapters/outbound/sqlite/insumo_repo.go | Implementar GetAllEnriquecido() |
| internal/adapters/inbound/admin/handlers.go | Usar GetAllEnriquecido + DTO de respuesta |
| internal/adapters/inbound/admin/templates/admin.html | Nueva tabla 8 col + banner + btn toggle |
| internal/adapters/inbound/admin/static/admin.js | renderizarInsumos v2 + sonido + evento tasa |
| internal/adapters/inbound/admin/static/admin.css | Todos los estilos nuevos del plan §3.5 |

---

## 6. Criterios de No Regresión

- go test ./... debe seguir 100% verde.
- Tests existentes unidades_handler_test.go y tasa_cambio_test.go no deben verse afectados.
- Binario final < 15 MB.
- Interfaces /cliente y /kds sin cambios.
