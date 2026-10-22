# Plan de Arquitectura — 001: Separación de Interfaces POS

**Estado:** Draft  
**Base:** AGENTS.md — Arquitectura Hexagonal Estricta

---

## 1. Principios Rectores

1. **Core aislado:** `internal/core/` no importa `net/http`, `database/sql` o drivers.
2. **Single Backend:** Un único binario Go sirve las tres interfaces.
3. **Sin frameworks web:** Solo `net/http` + `html/template` de la stdlib.
4. **SQLite WAL + AtomicDebitor:** Transacciones atómicas inmutables para inventario.
5. **Adaptadores de entrada por rol:** Cada vista es un adaptador inbound independiente.

---

## 2. Arquitectura Propuesta

```
cmd/main.go
├── internal/adapters/inbound/
│   ├── shared/           (middleware, JSON helpers, errors HTTP)
│   ├── client/           (vista QR móvil — cliente)
│   │   ├── routes.go     (router: /?mesa={n} → HTML)
│   │   ├── handlers.go   (listar platos, crear pedido)
│   │   └── templates/    (index.html mobile-first)
│   ├── kitchen/          (vista KDS — chef)
│   │   ├── routes.go     (router: /kds → HTML)
│   │   ├── handlers.go   (listar pedidos, cambiar estado)
│   │   └── templates/    (kds.html Kanban + SSE)
│   ├── admin/            (vista dashboard — admin)
│   │   ├── routes.go     (router: /admin/* → HTML/JSON)
│   │   ├── handlers.go   (CRUD insumos, platos, pedidos, auditoría)
│   │   └── templates/    (dashboard.html, forms, tables)
│   └── api/              (API gateway compartida)
│       └── router.go     (sub-routers por rol → handlers de cada vista)
├── internal/adapters/outbound/
│   ├── sqlite/           (SIN CAMBIOS — repos, atomic_debitor, db)
│   ├── printer/          (consola de cocina — reutilizado por KDS)
│   └── notifier/         (logs — reutilizado)
├── internal/core/
│   ├── domain/           (SIN CAMBIOS)
│   ├── ports/            (SIN CAMBIOS — puertos de entrada/salida)
│   └── services/         (SIN CAMBIOS — casos de uso)
└── migrations/           (SIN CAMBIOS)
```

### 2.1 Patrón de Rutas (Single Server, Multi-Views)

```go
// Router principal (api/router.go)
mux := http.NewServeMux()
mux.Handle("/", shared.WithMiddleware(client.NewHandler(inventario, pedido), client.LoggedRoutes()))
mux.Handle("/kds", kitchen.NewHandler(pedido, insumo))
mux.Handle("/admin/", admin.NewHandler(inventario, pedido, insumoRepo, movimientoRepo, platoRepo))
mux.HandleFunc("/api/", apiGateway.handleAPI)  // REST compartida
mux.HandleFunc("/ws/kds", kdsStream)            // SSE para tiempo real KDS
mux.HandleFunc("/ws/client", clientStream)      // SSE para cliente
```

### 2.2 Comunicación en Tiempo Real (SSE)

**No WebSockets** — Usamos Server-Sent Events (SSE) de la stdlib para:
- KDS: recibe notificaciones de nuevas órdenes y cambios de estado.
- Cliente: recibe el estado de su pedido (PENDIENTE → EN_PREPARACIÓN → LISTO).

```go
// Patrón SSE (stdlib net/http)
func (h *KitchenHandler) stream(w http.ResponseWriter, r *http.Request) {
    flusher, _ := w.(http.Flusher)
    w.Header().Set("Content-Type", "text/event-stream")
    // ... broadcast via channel interno
}
```

### 2.3 Flujo de Pedido (End-to-End)

```
[Cliente QR] → POST /api/pedidos (mesa=5, items=[...])
  → PedidoService.CrearPedido() → estado=PENDIENTE
  → Publica evento: order.created → canal SSE
  ↓
[Cliente KDS] ← SSE: orden aparece en columna PENDIENTE
  → Chef hace click: POST /api/pedidos/{id}/estado (EN_PREPARACIÓN)
  → Publica evento: order.status.updated → canal SSE
  → Debita inventario (AtomicDebitor) en estado LISTO
  ↓
[Cliente QR] ← SSE: estado cambia a EN_PREPARACIÓN → LISTO
  → Muestra notificación visual
```

### 2.4 Consistencia de Inventario

El débito de inventario se mantiene en el `AtomicDebtorSQLite` (SIN CAMBIOS).  
El KDS, al marcar un pedido como **LISTO**, llama a `ProcesarPedido()` que ejecuta el débito atómico.

---

## 3. Impacto en Componentes Existentes

| Componente | Acción |
|-----------|--------|
| `internal/core/*` | **Sin cambios** — lógica de negocio pura preservada |
| `migrations/*` | **Sin cambios** — esquema actual es suficiente |
| `internal/adapters/outbound/*` | **Sin cambios** — repositorios y adaptadores reutilizados |
| `internal/adapters/inbound/web/` | **Reemplazar** por `client/`, `kitchen/`, `admin/` + `api/` + `shared/` |
| `cmd/main.go` | Modularizar: extraer seeding a función separada |

---

## 4. Dependencias (cero adicionales)

| Tecnología | Fuente |
|-----------|--------|
| `net/http` | Go stdlib |
| `html/template` | Go stdlib |
| `encoding/json` | Go stdlib |
| `embed` | Go stdlib (para templates estáticos) |
| `go-sqlite3` | Ya en `go.mod` |

**No se añaden dependencias nuevas a `go.mod`.**
