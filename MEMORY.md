# MEMORY.md — POS Core & Inventario Restaurante
Memoria del proyecto entre sesiones. Máximo ~50 líneas.

## Estado actual
- ✅ Especificación `001-tres-interfaces-pos` y `002-rbac-multirol` implementadas al 100%.
- ✅ Autenticación perimetral Argon2id, tokens opacos de sesión (256 bits), cookies HttpOnly/SameSite=Strict.
- ✅ Matriz RBAC por middleware (Admin, Gerente, Cocinero, Mesero, Cajero) con CSRF, Rate Limiting y Security Headers.
- ✅ Repositorios SQLite WAL de alto rendimiento y resistencia a apagones.
- ✅ Interfaz Cliente (`/?mesa=1`), KDS (`/kds/`) y Admin (`/admin/`) con assets estáticos embebidos correctamente enrutados.
- ✅ Gestión de Insumos en Admin: edición e inserción **inline** directamente en la tabla. Botón `+ Agregar Producto` crea una fila editable al inicio con inputs integrados; botón ✏️ convierte filas existentes al mismo modo. El modal de registro de producto fue eliminado.
- ✅ Endpoints PUT/DELETE en `/admin/api/insumos/{id}` funcionales.
- ✅ Integración BCV: tasas USD y EUR sincronizadas automáticamente. Widget navbar. Ajuste manual offline.
- ✅ Spec 003 (Unidades de Medida) y Spec 004 (Tabla Insumos v2 + Alertas + Tasa): 100% implementadas.
- ✅ Spec 005 (Empaques y Presentaciones de Cervezas y Bebidas): columnas de empaque en tabla `insumos` desde `000001_init_schema.up.sql`; lógica de dominio preservada. La UI de empaque se mantuvo disponible para futura reactivación; el flujo principal de inventario ahora usa edición inline.
- ✅ **Migraciones Idempotentes (2026-10-20)**: Sistema `schema_migrations` en `db.go` via `RunNamedMigration()`. Las columnas nuevas (`precio_venta_usd`, campos de empaque, `unidad_receta_id`) se crean directamente en `000001_init_schema.up.sql` para evitar errores de `duplicate column name` en desarrollo.
- ✅ `go test ./...` 100% verde. Binario `bin/pos_core.exe` ~11.2 MB (< 15 MB).
- ✅ **Especificación `006-pestanas-inventario-merma-personal` implementada al 100%**: módulo de inventario rediseñado con 3 sub-pestañas superiores (`Inventario Principal`, `Mermas`, `Uso Personal`), todas con edición inline y sin modales. Estado de sub-pestaña persistido en `sessionStorage`.
- ✅ **Columna Merma removida del cuadro principal**: la tabla `Inventario Principal` ahora muestra 7 columnas (`Producto, Stock Almacén, Mínimo Requerido, Costo Prod., P. Venta, Margen, Ajustes`). El registro de mermas queda exclusivamente en la sub-pestaña `Mermas`.
- ✅ **Columnas Cantidad y Presentación / Medida implementadas**: la tabla `Inventario Principal` muestra `Cantidad` (número de presentaciones físicas) y `Presentación / Medida` (contenido total en unidad legible: kg, L, g, ml, u). Se agregaron al esquema `presentacion_nombre` y `cantidad_base_por_presentacion`; la UI permite editar la presentación y el peso/presentación de forma inline.
- ✅ **Empaque compuesto (cerveza)**: en edición inline el primer input inferior muestra el **total de unidades** (cantidad × unidades por caja), editable manualmente. El sistema calcula automáticamente el total en ml y muestra el desglose, ej: `72 unid. × 222 ml`. La columna Cantidad muestra el total de unidades como desglose.
- ✅ **Mínimo requerido en unidades para empaques compuestos**: cuando un producto tiene `unidades_por_empaque > 1` y `contenido_ml > 0`, el input de Mínimo Requerido se escribe en **unidades** (ej: 18 cervezas) y el backend lo convierte a ml internamente. La tabla muestra `18 unidades` con el equivalente en ml abajo.
- ✅ Backend Spec 006: `RegistrarUsoPersonal`, `ListarHistorialMermas`, `ListarHistorialUsoPersonal` en core + SQLite; endpoints `POST /admin/api/uso-personal`, `GET /admin/api/mermas/historial`, `GET /admin/api/uso-personal/historial`.
- ✅ Auditoría Estrategia A: uso personal almacenado como `SALIDA_MERMA` con referencia `USO_PERSONAL: ...`; sin alterar DDL de `movimientos_inventario`.
- ✅ Tests unitarios para `RegistrarUsoPersonal` y `RegistrarMerma` (débito, cantidad inválida, stock insuficiente, conversión de unidad).
- ✅ **BCV Live Scraping**: DolarAPIProvider consulta BCV con fallback a DolarAPI. Background ticker cada 10 minutos.

## Decisiones (y por qué)
- Arquitectura Hexagonal estricta: `internal/core/` desacoplado de HTTP y SQLite.
- **Migraciones idempotentes**: `RunNamedMigration(ctx, filename, sql)` crea tabla `schema_migrations` (PK = filename), verifica si fue aplicada y la omite. Resuelve crash al reiniciar con `ALTER TABLE ... ADD COLUMN` que ya existe.
- Eliminación de insumos: handler no bloquea por stock > 0 (modo pruebas). Repositorio Delete elimina dependencias (`insumo_unidades_compra`, `recetas_escandallo`, `movimientos_inventario`) en transacción atómica.
- Edición inline de insumos: eliminado el modal de registro. Ahora la inserción y edición ocurren directamente en la tabla con inputs integrados, cálculo de margen y Bs en tiempo real, y botones Guardar/Cancelar por fila.
- Auditoría Estrategia A (Spec 006): uso personal se registra como `SALIDA_MERMA` con prefijo `USO_PERSONAL:` en la referencia, evitando migraciones destructivas sobre `movimientos_inventario` en SQLite.
- Sub-pestañas de inventario: estado activo persistido en `sessionStorage` (`pos_subtab_inventario_activa`) para mantener contexto al navegar entre pestañas principales.
- Mermas y uso personal usan tablas inline de 2 columnas con débito atómico; el stock del inventario principal se actualiza en caché local tras guardar, sin recargar toda la página.
- RegistrarMerma ahora valida stock insuficiente (corrección de RF-05).

## Aprendizajes y errores a evitar
- `fs.Sub()`: importar `"io/fs"` directamente.
- Archivos estáticos: siempre aplicar `http.StripPrefix()` en mux raíz y sub-muxes para evitar 404.
- Insumos: para producción volver a exigir stock == 0 antes de DELETE.
- **ALTER TABLE idempotencia**: SQLite no soporta `ADD COLUMN IF NOT EXISTS`. Las columnas se añaden en `000001_init_schema.up.sql` y las migraciones 000004/000005/000006 quedan documentadas como históricas (comentadas) para evitar `duplicate column name`.
- **querySelector**: paréntesis de cierre FUERA del string del selector.
- **API Go JSON**: los structs Go serializan con mayúscula inicial; normalizar en JS.
- **JS inline editing**: los handlers `onclick="func(this)"` deben estar en botones con `this` bien posicionado; siempre usar `closest('tr')` para recuperar la fila.

## Próximos pasos
- ✅ Verificación end-to-end de `/admin/` Inventario completada.
- ✅ Configuración MCP GitHub (`.agents/mcp_config.json`, `mcp_config.json`, `.gitignore`, `GITHUB_BACKUP_MCP.md`).
- ✅ Respaldo en GitHub completado en https://github.com/kennydesing/sofware-para-restaurantes- (rama `main`).
- Integración física con impresoras térmicas ESC/POS de comanda en cocina.
- Reportes de ventas y movimientos de inventario exportables.




