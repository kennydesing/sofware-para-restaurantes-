# AGENTS.md — POS Core & Inventario por Escandallo para Restaurantes
Núcleo de backend en Go para la gestión de inventario crítico por escandallo (gramos/ml), control de mermas y punto de venta (POS) offline-first con SQLite WAL de alto rendimiento y resistencia a apagones.

## Stack y estructura
- Go 1.22+ estándar, SQLite (`github.com/mattn/go-sqlite3`) en modo WAL (`PRAGMA synchronous = NORMAL`, `busy_timeout = 5000`).
- Estructura Hexagonal Estricta:
  - `cmd/main.go`: Punto de entrada y demo end-to-end.
  - `migrations/000001_init_schema.up.sql`: Esquema DDL optimizado con FKs, restricciones CHECK e índices.
  - `internal/core/domain/`: Entidades puras (`insumo.go`, `receta.go`, `pedido.go`, `movimiento.go`, `plato.go`, `errors.go`). Lógica pura de negocio sin dependencias externas.
  - `internal/core/ports/`: Interfaces de entrada (`inbound_ports.go`) y salida (`outbound_ports.go`). Cero dependencias de `database/sql` o drivers.
  - `internal/core/services/`: Casos de uso (`inventario_service.go`, `pedido_service.go`).
  - `internal/adapters/outbound/sqlite/`: Adaptador SQLite (`atomic_debitor.go`, `db.go`, repositorios).
  - `internal/adapters/outbound/printer/` y `notifier/`: Adaptadores de consola y logs.

## Comandos
- Instalar dependencias: `go mod tidy`
- Ejecutar demostración: `go run cmd/main.go`
- Compilar binario de producción optimizado: `go build -ldflags="-s -w" -o bin/pos_core.exe ./cmd/main.go`
- Ejecutar pruebas: `go test -v ./...`
- Linter y análisis estático: `go vet ./...`

## Convenciones
- Textos de interfaz y comentarios en español técnico.
- Estilo: Go idiomático (`gofmt`), nombres descriptivos alineados con el dominio de restauración (`Insumo`, `RecetaItem`, `MovimientoInventario`).
- Patrones obligatorios:
  - Capa `core` 100% aislada: Prohibido importar `database/sql` o frameworks externos en `internal/core/`.
  - Transaccionalidad abstracta: Operaciones multi-tabla encapsuladas en adaptadores concretos (ej. `AtomicDebitor`) sin exponer `*sql.Tx` a los puertos.

## Datos y Reglas de dominio / trampas conocidas
- **Unidades Base Mínimas**: Todos los insumos se guardan en gramos (`g`), mililitros (`ml`) o unidades (`u`). La conversión desde compras ($kg \rightarrow g$, $L \rightarrow ml$, $docena \rightarrow u$) se realiza antes de persistir.
- **Fórmula de Merma en Escandallo**:
  $$\text{Consumo Real} = \text{Cantidad Requerida} \times \left(1 + \frac{\text{Porcentaje Merma}}{100}\right) \times \text{Cantidad Platos}$$
- **Costo Promedio Ponderado**: Si `stock_actual` $\le 0$, el nuevo costo adopta el costo unitario de la compra para evitar divisiones entre cero o costos inconsistentes.
- **Inmutabilidad y Auditoría**: Toda alteración de stock (ENTRADA_COMPRA, SALIDA_VENTA, SALIDA_MERMA, AJUSTE_MANUAL) debe generar un registro obligatorio en `movimientos_inventario`.
- **Atomicidad y Rollback**: El débito de inventario al vender es atómico. Si falta gramaje de un solo insumo, la transacción completa hace ROLLBACK y retorna `ErrStockInsuficiente`.

## Memoria
- Al empezar cualquier tarea, lee `MEMORY.md` para conocer el estado del proyecto y las decisiones tomadas.
- Al terminar una tarea, actualízalo: estado actual, decisiones importantes (con su porqué) y errores a evitar.
- Mantenlo breve (máximo ~50 líneas): resume o elimina lo que ya no aporte.
- Si algo se convierte en una regla permanente, propón moverlo a `AGENTS.md` en lugar de dejarlo en la memoria.
- No guardes nunca datos sensibles (claves, tokens, datos personales).

## Forma de trabajar
- Haz solo lo que se pide: no añadas funcionalidades o endpoints innecesarios por tu cuenta.
- Planifica cualquier cambio en el esquema SQL primero en `migrations/`.
- Cambios pequeños, limpios y enfocados. Al terminar, explica brevemente qué cambiaste.

## Límites
- ✅ Siempre: mantener la compatibilidad con el modo WAL de SQLite, garantizar la transaccionalidad atómica y actualizar `MEMORY.md` al terminar cada tarea.
- ⚠️ Pregunta antes: modificar la estructura de las tablas en `migrations/`, agregar dependencias a `go.mod` o alterar interfaces en `ports/`.
- 🚫 Nunca: incluir código basura/dead code, agregar servidores/routers HTTP (Gin, Chi, Fiber) o endpoints REST, ni usar ORMs pesados (GORM).

## Verificación
- Ejecutar `go run cmd/main.go` y confirmar que el flujo end-to-end corra sin errores.
- Validar que al intentar procesar un pedido sin stock suficiente se dispare el rollback sin alterar la base de datos.
- Compilar con `go build -ldflags="-s -w" -o bin/pos_core.exe ./cmd/main.go` y comprobar que el ejecutable sea liviano (< 15MB).