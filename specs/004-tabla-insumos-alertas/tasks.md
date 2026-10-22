# Tasks 004 — Desglose de Tareas Secuenciales

**Versión:** 1.0  
**Fecha:** 2026-10-17  
**Estado:** BORRADOR — Pendiente de aprobación  
**Referencia:** specs/004-tabla-insumos-alertas/spec.md + plan.md

---

## Principios de Ejecución

- Cada tarea es atómica, enfocada y verificable de forma independiente.
- El orden es estricto: las tareas de Backend preceden al Frontend que las consume.
- Ningún cambio de esquema SQL sin migración numerada y aprobada.
- Al finalizar cada bloque ejecutar: `go test ./...` y confirmar 100% verde.
- Actualizar MEMORY.md al completar la implementación completa.

---

## BLOQUE 0 — Aprobación y Pre-requisitos

### TASK-000 · Aprobación de spec y plan

**Responsable:** Arquitecto / Product Owner  
**Dependencias:** ninguna  
**Entregable:** Aprobación explícita de spec.md y plan.md antes de iniciar cualquier código.

Puntos que requieren aprobación antes de implementar:
1. Agregar migración 000005 (columna precio_venta_usd en insumos).
2. Ampliar la interfaz InsumoRepository con GetAllEnriquecido().
3. Decidir si InsumoEnriquecido vive en ports/ o se usa un DTO solo en el adaptador inbound.

---

## BLOQUE 1 — Base de Datos y Dominio (Backend)

### TASK-001 · Migración SQL 000005 — precio_venta_usd en insumos

**Archivo:** migrations/000005_precio_venta_insumo.up.sql  
**Descripción:** Agregar la columna precio_venta_usd REAL DEFAULT 0.0 a la tabla insumos.

```sql
-- migrations/000005_precio_venta_insumo.up.sql
ALTER TABLE insumos ADD COLUMN precio_venta_usd REAL DEFAULT 0.0;
```

```sql
-- migrations/000005_precio_venta_insumo.down.sql
-- SQLite ≥ 3.35: DROP COLUMN soportado directamente.
-- Para versiones anteriores: recrear tabla sin la columna.
ALTER TABLE insumos DROP COLUMN precio_venta_usd;
```

**Verificación:**
- Ejecutar la migración manualmente o via el bootstrap de `cmd/main.go`.
- Confirmar con: SELECT sql FROM sqlite_master WHERE name='insumos';
- El campo precio_venta_usd debe aparecer con DEFAULT 0.0.

---

### TASK-002 · Puerto de Salida — InsumoEnriquecido + GetAllEnriquecido()

**Archivo:** internal/core/ports/outbound_ports.go  
**Descripción:** Agregar el struct InsumoEnriquecido y el método GetAllEnriquecido al puerto InsumoRepository.

Cambios:
```go
// InsumoEnriquecido extiende domain.Insumo con campos calculados de la BD.
// No pertenece al dominio puro; vive en ports como DTO de capa de aplicación.
type InsumoEnriquecido struct {
    domain.Insumo
    PrecioVentaUSD float64  // USD, default 0.0
    MermaAcumulada float64  // suma de SALIDA_MERMA en unidad base
}
```

Método adicional en InsumoRepository:
```go
GetAllEnriquecido(ctx context.Context) ([]*InsumoEnriquecido, error)
```

**Verificación:** `go build ./...` sin errores de compilación.

---

### TASK-003 · Adaptador SQLite — Implementar GetAllEnriquecido()

**Archivo:** internal/adapters/outbound/sqlite/insumo_repo.go  
**Descripción:** Implementar GetAllEnriquecido() usando LEFT JOIN con movimientos_inventario para calcular merma_acumulada.

SQL a usar:
```sql
SELECT
    i.id,
    i.nombre,
    i.unidad_base,
    i.costo_promedio,
    COALESCE(i.precio_venta_usd, 0.0)    AS precio_venta_usd,
    i.stock_minimo,
    i.stock,
    COALESCE(SUM(CASE WHEN m.tipo_movimiento = 'SALIDA_MERMA'
                      THEN m.cantidad ELSE 0 END), 0.0) AS merma_acumulada
FROM insumos i
LEFT JOIN movimientos_inventario m ON m.insumo_id = i.id
GROUP BY i.id
ORDER BY i.nombre ASC;
```

**Verificación:**
- `go test ./internal/adapters/outbound/sqlite/... -v` sin errores.
- `go build ./...` limpio.

---

### TASK-004 · Handler HTTP — Actualizar handleGetInsumos con DTO

**Archivo:** internal/adapters/inbound/admin/handlers.go  
**Descripción:**
1. Definir InsumoResponseDTO (id, nombre, unidad_base, costo_promedio, precio_venta_usd, stock_minimo, stock, merma_acumulada).
2. Modificar handleGetInsumos para llamar a InsumoRepo.GetAllEnriquecido().
3. Mapear []*InsumoEnriquecido a []InsumoResponseDTO y serializar como JSON.

**Verificación:**
- `go test ./internal/adapters/inbound/admin/... -v` sin errores.
- Curl manual: `curl -b cookies.txt http://localhost:8080/admin/api/insumos`
  - La respuesta debe incluir los campos precio_venta_usd y merma_acumulada.
  - Registros existentes deben tener precio_venta_usd = 0.0 y merma_acumulada = 0.0 o el valor real.

---

## BLOQUE 2 — Frontend CSS (Estilos Base Primero)

### TASK-005 · CSS — Estilos de la Tabla v2

**Archivo:** internal/adapters/inbound/admin/static/admin.css  
**Descripción:** Agregar al final del archivo todos los estilos del plan §3.5.

Selectores a implementar en este orden:
1. .row-critica (fondo + borde izquierdo rojo)
2. .badge, .badge-critical, .badge-ok
3. @keyframes pulse-critical
4. .usd-value, .bs-value (doble línea de moneda)
5. .margen-badge, .margen-verde, .margen-ambar, .margen-rojo, .margen-monto
6. .ajustes-btns, .btn-icon, .btn-icon-danger
7. .unidad-tag
8. #banner-autoplay
9. .valor-nd
10. @media max-width:768px → .table-insumos min-width:900px

**Verificación:** Abrir el panel admin en el navegador y confirmar que no hay errores de CSS en la consola.

---

## BLOQUE 3 — Frontend HTML

### TASK-006 · HTML — Reemplazar tabla de insumos por la versión de 8 columnas

**Archivo:** internal/adapters/inbound/admin/templates/admin.html  
**Descripción:** En la sección #tab-inventario:
1. Agregar el botón #btn-toggle-sonido al header de sección.
2. Agregar el div#banner-autoplay (display:none por defecto).
3. Cambiar el id de la tabla de tbl-insumos a tabla-insumos-v2 (o mantener el mismo tbody id=tbl-insumos para compatibilidad con JS existente).
4. Reemplazar los <th> actuales por los 8 nuevos: Producto, Stock Almacén, Stock Mínimo, Costo Prod., P. Venta, Margen, Merma, Ajustes.
5. Actualizar colspan de la fila de carga de 8 a 8.

**Verificación:** El markup renderiza correctamente en el navegador sin errores de template Go.

---

## BLOQUE 4 — Frontend JS (Funcionalidad Core)

### TASK-007 · JS — Funciones auxiliares de formateo

**Archivo:** internal/adapters/inbound/admin/static/admin.js  
**Descripción:** Implementar/verificar las funciones auxiliares:
- formatStock(cantidad, unidad): formatea con separador de miles y la unidad.
  - Ej: formatStock(2000, 'ml') → "2,000 ml"
  - Ej: formatStock(15.5, 'kg') → "15.50 kg"
  - Ej: formatStock(48, 'u')   → "48 u"
- formatBs(monto): formatea en formato venezolano con 2 decimales.
  - Ej: formatBs(46500) → "46,500.00"
- fmtUSD(monto): formatea USD con 2 decimales.
  - Ej: fmtUSD(3.5) → "3.50"
- escHtml(str): escapa caracteres HTML para prevenir XSS.

**Verificación:** Pruebas manuales en la consola del navegador.

---

### TASK-008 · JS — renderizarInsumos() v2 con doble moneda y margen

**Archivo:** internal/adapters/inbound/admin/static/admin.js  
**Descripción:** Reemplazar la función renderizarInsumos existente por la versión del plan §3.2.

La nueva función debe:
1. Aceptar (insumos, tasaUSD).
2. Calcular Bs de costo y venta: USD × tasaUSD.
3. Calcular margenPct y margenMonto si precio_venta_usd > 0.
4. Asignar clase .row-critica y badge si stock <= stock_minimo.
5. Generar el HTML de las 8 columnas con los atributos data-usd en los span.bs-value.
6. Actualizar #stat-stock-bajo en el Dashboard.
7. Llamar dispararAlertaSonora() si hay al menos un ítem crítico.

**Verificación:**
- Cargar la pestaña Inventario y confirmar que la tabla muestra las 8 columnas.
- Verificar que los valores Bs están correctamente calculados (comparar manualmente con tasa actual del widget).
- Confirmar que las filas críticas se resaltan en rojo y muestran el badge ⚠️.

---

### TASK-009 · JS — Evento tasaActualizada y recálculo masivo de Bs

**Archivo:** internal/adapters/inbound/admin/static/admin.js  
**Descripción:**
1. Agregar la variable tasaUSDActual (global en el módulo JS) que se inicializa al cargar los insumos.
2. Registrar listener de CustomEvent('tasaActualizada') que llame a recalcularValoresBs(nuevaTasa).
3. recalcularValoresBs() itera sobre todos los document.querySelectorAll('.bs-value[data-usd]') y actualiza su textContent.
4. Modificar sincronizarTasasBCV() y abrirModalTasaManual() para emitir el evento CustomEvent('tasaActualizada') con la nueva tasa tras una actualización exitosa.

**Verificación:**
- Sincronizar BCV desde el navbar y confirmar que todos los valores Bs de la tabla se actualizan sin recargar la página.
- Hacer un ajuste manual de tasa y verificar el mismo comportamiento.

---

### TASK-010 · JS — Sistema de Alerta Sonora con Web Audio API

**Archivo:** internal/adapters/inbound/admin/static/admin.js  
**Descripción:** Implementar las funciones del plan §3.4:
1. Variable alertaSonidoActiva (desde localStorage).
2. Variable paginaInteractuada con listener en primer click.
3. Variable alarmaDisparada como debounce de 30 s.
4. toggleAlertaSonido() con persistencia en localStorage y actualización del icono del botón.
5. inicializarToggleSonido() que lee localStorage al cargar la página.
6. dispararAlertaSonora() con Web Audio API (oscilador sine 880 Hz, 0.4 s, fade-out).
7. mostrarBannerAutoplay() para política de autoplay.

**Verificación:**
- Con un insumo en stock crítico, cargar la página y verificar que suena la alarma.
- Silenciar con el botón 🔕 y recargar: el sonido no debe reproducirse pero la alerta visual sí.
- Recargar nuevamente: el botón debe mostrar 🔕 (estado persistido en localStorage).
- Activar nuevamente con 🔔 y verificar que el sonido regresa.

---

## BLOQUE 5 — Integración y Pruebas

### TASK-011 · Prueba de Integración End-to-End

**Descripción:** Flujo completo de verificación manual:

1. Arrancar: `go run cmd/main.go`
2. Login como Admin en http://localhost:8080
3. Navegar a Inventario & Compras → verificar tabla de 8 columnas.
4. Verificar que insumos con stock <= stock_minimo muestran .row-critica + badge ⚠️.
5. Verificar doble línea USD/Bs en columnas Costo Prod. y P. Venta.
6. Sincronizar BCV desde el navbar → verificar que los valores Bs de la tabla se actualizan sin recarga.
7. Abrir la consola del navegador → confirmar que no hay errores JS.
8. Probar el toggle de sonido (🔔 / 🔕) y verificar persistencia tras recarga.
9. Registrar una merma desde el modal correspondiente → verificar que la columna Merma se actualiza al recargar los insumos.
10. Verificar que el contador "Alertas Stock Bajo" en el Dashboard es coherente con la tabla.

---

### TASK-012 · Prueba de Regresión Go

**Descripción:** Ejecutar la suite completa de tests y el build de producción.

```bash
go test -v ./...
go build -ldflags="-s -w" -o bin/pos_core.exe ./cmd/main.go
ls -lh bin/pos_core.exe   # debe ser < 15 MB
```

**Criterios de aceptación:**
- 100% de tests en verde.
- Sin errores de compilación.
- Binario < 15 MB.
- go vet ./... sin advertencias.

---

### TASK-013 · Prueba de Acceso por Rol (RBAC)

**Descripción:** Verificar que el RBAC existente sigue funcionando correctamente.

| Rol | Acceso a Inventario | Puede ver tabla | Puede ajustar |
|-----|---------------------|-----------------|---------------|
| Admin | ✅ | ✅ | ✅ |
| Gerente | ✅ | ✅ | ✅ |
| Cajero | ❌ (redirige) | — | — |
| Mesero | ❌ | — | — |
| Cocinero | ❌ | — | — |

**Verificación:** Login con cada rol y confirmar comportamiento esperado.

---

### TASK-014 · Actualizar MEMORY.md

**Archivo:** MEMORY.md  
**Descripción:** Agregar al estado actual:

```markdown
- ✅ Spec 004 (Tabla Insumos v2 + Alertas + Tasa): implementada al 100%.
  - Migración 000005: columna precio_venta_usd en tabla insumos.
  - GetAllEnriquecido(): LEFT JOIN con movimientos_inventario para merma_acumulada.
  - Frontend: tabla de 8 columnas con dual-currency USD/Bs, margen automático, alerta visual .row-critica y alerta sonora Web Audio API con toggle localStorage.
  - Evento CustomEvent('tasaActualizada') para recálculo masivo de Bs sin recarga.
```

---

## Resumen de Secuencia

```
TASK-000 (Aprobación)
    │
    ├── TASK-001 (Migración SQL)
    │       └── TASK-002 (Puerto GetAllEnriquecido)
    │               └── TASK-003 (SQLite Adapter)
    │                       └── TASK-004 (Handler HTTP + DTO)
    │
    ├── TASK-005 (CSS estilos tabla v2)
    │       └── TASK-006 (HTML tabla 8 columnas)
    │               ├── TASK-007 (JS helpers formateo)
    │               ├── TASK-008 (JS renderizarInsumos v2)
    │               ├── TASK-009 (JS evento tasaActualizada)
    │               └── TASK-010 (JS alerta sonora Web Audio)
    │
    └── TASK-011 (E2E manual)
            ├── TASK-012 (Regresión Go: go test + build)
            ├── TASK-013 (RBAC verification)
            └── TASK-014 (Actualizar MEMORY.md)
```

---

## Estimación de Esfuerzo

| Tarea | Esfuerzo | Riesgo |
|-------|----------|--------|
| TASK-001 Migración SQL | 15 min | Bajo |
| TASK-002 Puerto | 20 min | Bajo |
| TASK-003 SQLite adapter | 30 min | Medio (SQL con JOIN) |
| TASK-004 Handler + DTO | 25 min | Bajo |
| TASK-005 CSS | 30 min | Bajo |
| TASK-006 HTML | 20 min | Bajo |
| TASK-007 JS helpers | 20 min | Bajo |
| TASK-008 JS renderizar | 45 min | Medio (muchos campos) |
| TASK-009 JS evento tasa | 20 min | Bajo |
| TASK-010 JS sonido | 25 min | Bajo (Web Audio API simple) |
| TASK-011 E2E manual | 30 min | Bajo |
| TASK-012 Regresión | 10 min | Bajo |
| TASK-013 RBAC | 10 min | Bajo |
| TASK-014 MEMORY.md | 5 min | Bajo |
| **TOTAL** | **~5.5 h** | |
