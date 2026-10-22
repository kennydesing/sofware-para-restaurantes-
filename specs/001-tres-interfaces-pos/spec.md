# Especificación Técnica — 001: Separación de Interfaces POS en Tres Vistas

**Versión:** 1.0  
**Estado:** Draft  
**Fecha:** 2026-10-10  
**Autor:** Equipo de Ingeniería  
**Base:** AGENTS.md — POS Core & Inventario por Escandallo para Restaurantes

---

## 1. Introducción

### 1.1 Propósito

Esta especificación detalla la planificación y desglose técnico para separar la interfaz web monolítica actual en tres vistas independientes, manteniendo un backend único en Go bajo estricta Arquitectura Hexagonal. Cada vista atiende a un rol específico del restaurante: cliente (QR por mesa), cocina (KDS) y administración (dashboard maestro).

### 1.2 Alcance

Incluye:
- Interface de Cliente (QR por Mesa): menú móvil y envío de comandas.
- Interface de Cocina (KDS): tablero de control de órdenes en tiempo real con actualización de estados y débito atómico de inventario.
- Panel de Administración: gestión de inventario, menú, auditoría y reportes.

No incluye:
- Cambios en la lógica de dominio (`internal/core/domain/`).
- Modificaciones a la estructura de tablas migracionales sin revisión previa.
- Adición de frameworks web externos (Gin, Chi, Fiber, GORM).

### 1.3 Definiciones

| Término | Definición |
|---------|-----------|
| **Escandallo** | Ficha técnica que define la cantidad de cada insumo necesaria por plato, incluyendo porcentaje de merma. |
| **Merma** | Pérdida de materia prima durante la preparación (corte, cocción, error). |
| **KDS** | Kitchen Display System — Sistema de visualización de cocina. |
| **Débito Atómico** | Transacción SQLite (BEGIN/COMMIT/ROLLBACK) que descuenta stock de insumos y registra movimientos en una sola operación. |
| **WAL** | Write-Ahead Logging — modo de journal de SQLite para lecturas concurrentes y resistencia a apagones. |

---

## 2. Descripción General del Sistema

### 2.1 Arquitectura Actual

```
cmd/main.go
├── internal/adapters/inbound/web/     (interfaz monolítica actual)
│   ├── server.go                      (router net/http + static embed)
│   ├── handlers.go                    (API handlers)
│   └── static/                        (index.html, styles.css, app.js)
├── internal/adapters/outbound/sqlite/ (repositorios SQLite)
├── internal/adapters/outbound/printer/ (consola)
├── internal/adapters/outbound/notifier/ (logs)
├── internal/core/ports/               (puertos de entrada/salida)
├── internal/core/services/            (casos de uso)
└── internal/core/domain/              (entidades puras)
```

### 2.2 Arquitectura Objetivo

```
cmd/main.go
├── internal/adapters/inbound/client/    (vista QR móvil - cliente)
├── internal/adapters/inbound/kitchen/   (vista KDS - cocina)
├── internal/adapters/inbound/admin/     (vista dashboard - admin)
├── internal/adapters/inbound/api/       (API gateway compartida - router único)
├── internal/adapters/outbound/sqlite/   (repositorios - SIN CAMBIOS)
├── internal/core/ports/                 (puertos - SIN CAMBIOS)
├── internal/core/services/              (casos de uso - SIN CAMBIOS)
└── internal/core/domain/                (entidades - SIN CAMBIOS)
```

**Principio:** El backend (`internal/core/`) permanece completamente aislado de cualquier detalle de transporte (HTTP, WebSocket, colas). Los adaptadores de entrada (`client`, `kitchen`, `admin`) dependen únicamente de los puertos (`ports.InventarioService`, `ports.PedidoService`, etc.), nunca del core directamente.

### 2.3 Restricciones Técnicas

| Restricción | Detalle |
|------------|---------|
| **Hexagonal** | `internal/core/` no importa `net/http`, `database/sql` o drivers. |
| **SQLite WAL** | PRAGMA `journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout=5000`. |
| **AtomicDebitor** | Transacción explícita con `*sql.Tx`: verificación previa → débito → auditoría → commit. |
| **Sin frameworks web** | Solo `net/http` stdlib. Cero dependencias externas. |
| **CGO** | Requerido para `go-sqlite3`. |
| **Sin ORM** | Uso exclusivo de `database/sql` nativo. |

---

## 3. Requisitos Funcionales (Formato EARS)

### 3.1 RF-01 — Interface de Cliente (QR por Mesa)

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-01.01 | Universal | El [sistema] shal permitir al cliente visualizar el menú de platos activos al acceder a `/?mesa={n}` |
| RF-01.02 | State-driven | Mientras [el cliente está en la vista de menú] el [sistema] shal mostrar los platos con nombre, precio y descripción |
| RF-01.03 | Universal | El [sistema] shal permitir al cliente agregar platos al carrito con cantidad especificada |
| RF-01.04 | Event-driven | Si [el cliente confirma el carrito], el [sistema] shal crear un Pedido con estado PENDIENTE asociado a su mesa ID |
| RF-01.05 | Universal | El [sistema] shal mostrar al cliente un ticket de confirmación con número de pedido y total |
| RF-01.06 | Universal | El [sistema] shal permitir al cliente ver el estado de su pedido en tiempo real (PENDIENTE → EN_PREPARACIÓN → LISTO) |
| RF-01.07 | Optional | Si [no hay platos activos], el [sistema] shal mostrar un mensaje "Menú no disponible" |
| RF-01.08 | Universal | El [sistema] shal generar códigos QR para las 10 mesas (IDs 1-10) |
| RF-01.09 | Universal | Cada [código QR] shal codificar el ID de mesa como parámetro de consulta (`/?mesa={n}`) |
| RF-01.10 | Universal | El [sistema] shal proveer una página de referencia (`/qr`) con todos los 10 QR para imprimir |
| RF-01.11 | Universal | El [sistema] shal validar que el `mesa_id` recibido esté en el rango 1-10 antes de crear el pedido |

### 3.2 RF-02 — Interface de Cocina (KDS)

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-02.01 | Universal | El [sistema] shal mostrar al chef un tablero con todas las órdenes en estado PENDIENTE |
| RF-02.02 | State-driven | Mientras [un pedido está PENDIENTE] el [sistema] shal mostrarlo en la primera columna del KDS |
| RF-02.03 | Universal | El [sistema] shal permitir al chef mover un pedido a "EN_PREPARACIÓN" al hacer clic en él |
| RF-02.04 | Universal | El [sistema] shal permitir al chef mover un pedido a "LISTO" cuando la preparación está completa |
| RF-02.05 | Event-driven | Si [el chef marca un pedido como LISTO], el [sistema] shal notificar al cliente de forma visual |
| RF-02.06 | Universal | El [sistema] shal mostrar por cada orden: número de pedido, mesa, platos con cantidades, hora de creación y tiempo de espera |
| RF-02.07 | Unwanted-behavior | Si [el chef intenta avanzar un pedido que no está en PENDIENTE], el [sistema] shal mostrar un error y no permitir la transición |
| RF-02.08 | Universal | El [sistema] shal mostrar todas las órdenes de las 10 mesas en el KDS, ordenadas por tiempo de creación |
| RF-02.09 | Universal | El [KDS] shal desplegar el número de mesa asociado a cada orden en la tarjeta de la orden |

### 3.3 RF-03 — Panel de Administración

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-03.01 | Universal | El [sistema] shal permitir al administrador crear y listar insumos con unidad base y stock mínimo |
| RF-03.02 | Universal | El [sistema] shal permitir al administrador registrar compras con conversión automática (kg→g, L→ml, docena→u) |
| RF-03.03 | Universal | El [sistema] shal permitir al administrador crear platos con precio de venta y ficha técnica (escandallo con merma) |
| RF-03.04 | Universal | El [sistema] shal mostrar el libro diario de auditoría con todos los movimientos de inventario agrupados por insumo |
| RF-03.05 | Universal | El [sistema] shal proveer de auditoría de stock (teórico vs físico) con descuadre valorizado |
| RF-03.06 | Universal | El [sistema] shal permitir al administrador ajustar stock manualmente tras auditoría física |
| RF-03.07 | Universal | El [sistema] shal generar alertas visuales cuando un insumo cae por debajo de su stock mínimo |

### 3.4 RF-04 — Consistencia de Inventario (Transaccional)

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-04.01 | Universal | El [sistema] shal debitar stock de insumos de forma atómica al procesar un pedido, usando SQLite `*sql.Tx` |
| RF-04.02 | Unwanted-behavior | Si [un insumo no tiene stock suficiente para cubrir el escandallo del pedido], el [sistema] shal hacer ROLLBACK total de la transacción y retornar error `ErrStockInsuficiente` |
| RF-04.03 | Universal | El [sistema] shal registrar obligatoriamente un movimiento `SALIDA_VENTA` por cada insumo consumido |
| RF-04.04 | Universal | El [sistema] shal aplicar la fórmula de merma: `consumo_real = cantidad_requerida × (1 + porcentaje_merma/100) × cantidad_platos` |

---

## 4. Especificación de Datos

### 4.1 Entidades (sin cambios)

Las entidades del dominio permanecen idénticas: `Insumo`, `RecetaItem`, `Plato`, `Pedido`, `DetallePedido`, `MovimientoInventario`, `Mesa`, `EstadoPedido`, `TipoMovimiento`.

### 4.2 Nuevas Transferencias de Estado (WebSocket)

| Evento | Trigger | Payload | Destinatarios |
|--------|---------|---------|--------------|
| `order.created` | Cliente crea pedido | `{pedido_id, mesa_id, detalles, total}` | KDS, Admin |
| `order.status.updated` | Chef cambia estado | `{pedido_id, estado}` | Cliente, Admin |
| `order.processed` | Débito atómico OK | `{pedido_id, insumos_debitados}` | Admin |
| `stock.alert` | Stock bajo mínimo | `{insumo_id, nombre, stock}` | Admin |

---

## 5. Especificación de Interfaces

### 5.1 Interface de Cliente (QR por Mesa)

- **URL:** `http://localhost:8080/?mesa={n}`
- **Tecnología:** HTML + CSS (mobile-first) + vanilla JS
- **Funcionalidades:**
  - Visualización del menú de platos activos
  - Carrito de pedidos con contador de cantidad
  - Confirmación de pedido
  - Seguimiento en tiempo real del estado

### 5.2 Interface de Cocina (KDS)

- **URL:** `http://localhost:8080/kds`
- **Tecnología:** HTML + CSS (grid Kanban) + vanilla JS + Server-Sent Events (SSE)
- **Columnas Kanban:** PENDIENTE → EN_PREPARACIÓN → LISTO
- **Funcionalidades:**
  - Tablero Kanban de órdenes
  - Drag & drop entre columnas
  - Timer de espera por orden
  - Notificación visual al cliente cuando su orden está LISTA

### 5.3 Panel de Administración

- **URL:** `http://localhost:8080/admin`
- **Tecnologización:** HTML + CSS (dashboard) + vanilla JS
- **Secciones:**
  - Dashboard: tarjetas de stock, alertas
  - Inventario: CRUD de insumos, compras, mermas, ajustes
  - Menú: CRUD de platos, fichas técnicas
  - Órdenes: historial de pedidos con filtros
  - Auditoría: libro diario de movimientos

---

## 6. Consideraciones de Seguridad

| Requisito | Detalle |
|-----------|---------|
| SC-01 | El [admin] shal acceder al panel mediante autenticación básica HTTP o token |
| SC-02 | El [cliente] shal crear pedidos sin autenticación, pero con validación de mesa ID (1-99) |
| SC-03 | El [chef] shal acceder al KDS sin autenticación (red interna de cocina) |
| SC-04 | El [sistema] shal validar todos los inputs del cliente para prevenir SQL injection (usando prepared statements) |

---

## 7. Consideraciones de Rendimiento

| Requisito | Detalle |
|---------------------|---------|
| PF-01 | El [sistema] shal atender ≤100 clientes concurrentes simultáneos |
| PF-02 | El [KDS] shal actualizar el tablero en ≤2 segundos tras cambio de estado |
| PF-03 | El [backend] shal mantener CGO_ENABLED=1 para go-sqlite3 con SQLite WAL |
