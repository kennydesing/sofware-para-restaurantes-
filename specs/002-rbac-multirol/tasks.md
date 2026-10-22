# Tareas Secuenciales 002 — RBAC Multirol

**Proyecto:** POS Core — Inventario y Punto de Venta para Restaurantes  
**Vinculado a:** `specs/002-rbac-multirol/spec.md` y `plan.md`  
**Estimación Total:** ~11–14 días de desarrollo  
**Prerrequisito:** MVP `spec/001` compilando al 100% con `go build`, `go vet` y `go test` en verde.

---

## Principio de Ejecución

> **Una tarea a la vez. Verificar con `go build` y `go vet` tras cada tarea.**  
> No comenzar la siguiente hasta que la anterior compile limpio y los tests pasen.

---

## Fase 0 — Preparación (Día 0 · 0.5 días)

### TAREA-00 · Aprobar dependencia nueva
**Archivo afectado:** `go.mod`, `go.sum`  
**Estimación:** 0.5 días  
**Prioridad:** BLOQUEANTE — sin esto no hay Argon2id

- [x] Revisar licencia de `golang.org/x/crypto` (BSD-3 ✅ compatible).
- [x] Obtener aprobación del equipo (requerida por `AGENTS.md` para nuevas dependencias).
- [x] Ejecutar `go get golang.org/x/crypto@v0.27.0`.
- [x] Confirmar que `go mod tidy` pasa sin errores.
- [x] Verificar: `go build ./...` y `go vet ./...` sin advertencias.

**Criterio de aceptación:** `go.mod` incluye `golang.org/x/crypto`, `go build` pasa.

---

## Fase 1 — Base de Datos (Días 1–2 · 1.5 días)

### TAREA-01 · Crear migración SQL de autenticación
**Archivos nuevos:**
- `migrations/000002_auth.up.sql`
- `migrations/000002_auth.down.sql`

**Estimación:** 0.5 días  
**Prioridad:** Alta — bloquea todas las fases siguientes

- [x] Crear `000002_auth.up.sql` con las tablas `usuarios`, `sesiones` y `auditoria_acciones` (ver `plan.md §2`).
- [x] Crear `000002_auth.down.sql` con los `DROP TABLE` en orden correcto (respetando FKs).
- [x] Revisar CHECKs de `rol IN (...)`, índices y constraints UNIQUE.
- [x] Aplicar la migración en la DB de desarrollo: `sqlite3 data/pos.db < migrations/000002_auth.up.sql`.
- [x] Verificar con `.schema` en sqlite3 que las 3 tablas y sus índices existen.
- [x] Validar rollback: `sqlite3 data/pos.db < migrations/000002_auth.down.sql` limpia todo sin error.

**Criterio de aceptación:** Las 3 tablas existen en SQLite con constraints e índices correctos.

---

### TAREA-02 · Integrar migración en el bootstrap de la aplicación
**Archivo afectado:** `internal/adapters/outbound/sqlite/db.go`

**Estimación:** 1 día  
**Prioridad:** Alta

- [x] Añadir la ejecución de `000002_auth.up.sql` embebida (via `embed.FS`) en la función de inicialización de la DB, después de `000001_init_schema.up.sql`.
- [x] Usar `CREATE TABLE IF NOT EXISTS` para idempotencia (igual que migration 001).
- [x] Verificar que al arrancar con `go run cmd/main.go` las tablas se crean automáticamente en un DB vacío.
- [x] Verificar idempotencia: segunda ejecución no produce error ni duplicación.

**Criterio de aceptación:** `go run cmd/main.go` arranca sin errores con las nuevas tablas ya presentes.

---

## Fase 2 — Core Domain (Días 2–3 · 1 día)

### TAREA-03 · Crear entidades de dominio
**Archivos nuevos:**
- `internal/core/domain/usuario.go`
- `internal/core/domain/sesion.go`
- `internal/core/domain/auditoria.go`

**Estimación:** 0.5 días  
**Prioridad:** Alta — bloquea ports y services

- [x] Implementar `usuario.go` con tipo `Rol`, constantes de roles, struct `Usuario`, métodos `EsActivo()`, `TieneRol()` y `PuedeAcceder()`.
- [x] Implementar `sesion.go` con struct `Sesion` y método `EstaVigente()`.
- [x] Implementar `auditoria.go` con tipo `AccionAuditoria`, constantes y struct `EventoAuditoria`.
- [x] Añadir errores nuevos a `domain/errors.go`: `ErrCredencialesInvalidas`, `ErrCuentaInactiva`, `ErrSesionExpirada`, `ErrSesionInvalida`, `ErrEmailDuplicado`, `ErrRolInvalido`, `ErrAccesoDenegado`, `ErrPasswordInsegura`.
- [x] Verificar que ningún archivo en `domain/` importa paquetes externos (solo `errors`, `time`).
- [x] `go vet ./internal/core/domain/...` sin advertencias.

**Criterio de aceptación:** Dominio compila limpio, cero imports de `database/sql` o `net/http`.

---

### TAREA-04 · Escribir tests unitarios de dominio
**Archivos nuevos:**
- `internal/core/domain/usuario_test.go`

**Estimación:** 0.5 días  
**Prioridad:** Media

- [x] Test: `PuedeAcceder` retorna true cuando el rol está en la lista, false cuando no.
- [x] Test: `EsActivo()` con usuario activo e inactivo.
- [x] Test: `EstaVigente()` con sesión futura y pasada.
- [x] `go test -v ./internal/core/domain/...` 100% verde.

**Criterio de aceptación:** `go test ./internal/core/domain/...` pasa con 0 fallos.

---

## Fase 3 — Ports (Día 3 · 0.5 días)

### TAREA-05 · Definir puertos de autenticación
**Archivos nuevos:**
- `internal/core/ports/auth_ports.go` (inbound: `AuthService`, `AuditoriaService`)
- `internal/core/ports/auth_outbound_ports.go` (outbound: `UsuarioRepository`, `SesionRepository`, `AuditoriaRepository`)

**Estimación:** 0.5 días  
**Prioridad:** Alta — bloquea services y adaptadores

- [x] Implementar DTOs de entrada: `LoginRequest`, `CrearUsuarioRequest`, `ModificarUsuarioRequest`, `FiltroAuditoria`.
- [x] Definir interfaz `AuthService` con métodos: `Login`, `Logout`, `ValidarSesion`, `CrearUsuario`, `ModificarUsuario`, `ListarUsuarios`, `LimpiarSesionesExpiradas`.
- [x] Definir interfaz `AuditoriaService` con métodos: `RegistrarEvento`, `ListarEventos`.
- [x] Definir interfaces outbound: `UsuarioRepository`, `SesionRepository`, `AuditoriaRepository`.
- [x] Verificar que ningún puerto importa `database/sql`, `net/http` o drivers.
- [x] `go build ./internal/core/ports/...` limpio.

**Criterio de aceptación:** Puertos compilan sin errores, 0 dependencias externas.

---

## Fase 4 — Servicios de Dominio (Días 4–6 · 2 días)

### TAREA-06 · Implementar AuthService
**Archivo nuevo:** `internal/core/services/auth_service.go`

**Estimación:** 1.5 días  
**Prioridad:** Alta

- [x] Implementar struct `authService` con dependencias inyectadas: `UsuarioRepository`, `SesionRepository`, `AuditoriaRepository`.
- [x] **Login:**
  - Llamar `UsuarioRepository.GetByEmail` → `ErrCredencialesInvalidas` si no existe (con retardo 300ms).
  - Llamar `argon2.CompareHashAndPassword` → `ErrCredencialesInvalidas` si no coincide (con retardo 300ms).
  - Verificar `usuario.Activo` → `ErrCuentaInactiva` si está desactivado.
  - Generar token con `crypto/rand`.
  - Crear sesión con TTL desde `SESSION_TTL_HOURS` (env var, default 8h).
  - Llamar `SesionRepository.Create`.
  - Registrar `AccionLogin` en auditoría.
  - Retornar `*domain.Sesion`.
- [x] **ValidarSesion:** `SesionRepository.GetByToken` → verificar `EstaVigente()` → retornar usuario completo.
- [x] **Logout:** `SesionRepository.DeleteByToken` + registrar `AccionLogout`.
- [x] **CrearUsuario:** validar política de contraseña → hashear con `argon2.GenerateFromPassword` → `UsuarioRepository.Create` → registrar `AccionCrearUsuario`.
- [x] **ModificarUsuario:** `UsuarioRepository.Update` → registrar `AccionModificarRol` o `AccionDesactivarUsuario`.
- [x] **LimpiarSesionesExpiradas:** `SesionRepository.DeleteExpired` → log de filas eliminadas.
- [x] Asegurarse de que `authService` implementa la interfaz `ports.AuthService` con `var _ ports.AuthService = (*authService)(nil)`.

**Criterio de aceptación:** Compilación limpia. Test de integración básico de login/logout pasa.

---

### TAREA-07 · Implementar AuditoriaService
**Archivo nuevo:** `internal/core/services/auditoria_service.go`

**Estimación:** 0.5 días  
**Prioridad:** Media

- [x] Implementar `RegistrarEvento`: simplemente delega a `AuditoriaRepository.Create`. No falla silenciosamente — propaga errores.
- [x] Implementar `ListarEventos`: construye el filtro SQL desde `FiltroAuditoria` y delega a `AuditoriaRepository.GetFiltered`.
- [x] `var _ ports.AuditoriaService = (*auditoriaService)(nil)` para verificación en tiempo de compilación.

**Criterio de aceptación:** Compila. Tests unitarios con mock del repositorio pasan.

---

## Fase 5 — Adaptadores de Salida SQLite (Días 6–8 · 2 días)

### TAREA-08 · Repositorio de Usuarios SQLite
**Archivo nuevo:** `internal/adapters/outbound/sqlite/usuario_repo.go`

**Estimación:** 0.5 días

- [x] `GetByEmail`: `SELECT * FROM usuarios WHERE email = ?`. Mapear fila a `domain.Usuario`. Retornar `domain.ErrNotFound` si no existe.
- [x] `GetByID`: `SELECT * FROM usuarios WHERE id = ?`.
- [x] `GetAll`: `SELECT id, nombre, email, rol, activo, created_at, updated_at FROM usuarios ORDER BY created_at DESC`. **Nunca seleccionar `password_hash`.**
- [x] `Create`: `INSERT INTO usuarios (nombre, email, password_hash, rol, activo, ...)`. Detectar violación UNIQUE → retornar `domain.ErrEmailDuplicado`.
- [x] `Update`: `UPDATE usuarios SET rol=?, activo=?, updated_at=? WHERE id=?`. **No incluir `password_hash` en el UPDATE.**

**Criterio de aceptación:** `go test ./internal/adapters/outbound/sqlite/...` con DB en memoria pasa.

---

### TAREA-09 · Repositorio de Sesiones SQLite
**Archivo nuevo:** `internal/adapters/outbound/sqlite/sesion_repo.go`

**Estimación:** 0.5 días

- [x] `Create`: `INSERT INTO sesiones (token, usuario_id, ip_origen, user_agent, expires_at)`.
- [x] `GetByToken`: `SELECT * FROM sesiones WHERE token = ?`. Si no existe → `domain.ErrSesionInvalida`. La validación de expiración la hace el servicio (no el repo).
- [x] `DeleteByToken`: `DELETE FROM sesiones WHERE token = ?`.
- [x] `DeleteByUsuarioID`: `DELETE FROM sesiones WHERE usuario_id = ?` (para SINGLE_SESSION).
- [x] `DeleteExpired`: `DELETE FROM sesiones WHERE expires_at < datetime('now')`. Retornar `RowsAffected`.

**Criterio de aceptación:** Tests con SQLite en memoria pasan.

---

### TAREA-10 · Repositorio de Auditoría SQLite
**Archivo nuevo:** `internal/adapters/outbound/sqlite/auditoria_repo.go`

**Estimación:** 1 día

- [x] `Create`: `INSERT INTO auditoria_acciones (usuario_id, accion, entidad_tipo, entidad_id, detalle, ip_origen)`.
- [x] `GetFiltered`: construir query dinámico con `WHERE` opcionales para `usuario_id`, `created_at BETWEEN ? AND ?`, más `LIMIT ? OFFSET ?`.
- [x] Implementar helper `offsetFromPage(pagina, porPagina int) int`.
- [x] Verificar que en ningún momento se genera una query `UPDATE` o `DELETE` en este repositorio.

**Criterio de aceptación:** Tests de filtrado paginado con DB en memoria pasan.

---

## Fase 6 — Adaptadores de Entrada (Días 8–11 · 3 días)

### TAREA-11 · Login Handler
**Archivos nuevos:**
- `internal/adapters/inbound/auth/handler.go`
- `internal/adapters/inbound/auth/templates/login.html`

**Estimación:** 1 día

- [x] `HandleGetLogin`: renderizar `login.html`. Si ya hay sesión válida, redirigir a la ruta del rol.
- [x] `HandlePostLogin`: parsear form → rate limiter → `authSvc.Login` → set cookie `pos_session` (HttpOnly, SameSite=Strict) → redirigir por rol.
- [x] `HandleLogout`: `authSvc.Logout(token)` → clear cookie → redirigir a `/login`.
- [x] Diseñar `login.html` coherente con el diseño visual del POS (mismos fonts, colores, dark mode).
- [x] Incluir campo CSRF oculto en el formulario de login.
- [x] Registrar rutas en `cmd/main.go`: `GET /login`, `POST /login`, `POST /logout`.

**Criterio de aceptación:** Navegar a `http://localhost:8080/login` muestra el formulario. Login exitoso redirige al área correcta.

---

### TAREA-12 · Middleware de Autenticación
**Archivo nuevo:** `internal/adapters/inbound/shared/middleware/auth_middleware.go`

**Estimación:** 0.5 días

- [x] Implementar `AuthMiddleware(authSvc)` como descrito en `plan.md §5.1`.
- [x] Distinguir entre peticiones de navegador (HTML) y API (JSON) para la respuesta de error.
- [x] Implementar `UsuarioFromContext(ctx)` helper.
- [x] Tests unitarios con `httptest` verificando redirección y 401.

**Criterio de aceptación:** Tests unitarios del middleware pasan. Ruta sin sesión es redirigida.

---

### TAREA-13 · Middleware RBAC
**Archivo nuevo:** `internal/adapters/inbound/shared/middleware/rbac_middleware.go`

**Estimación:** 0.5 días

- [x] Implementar `RequireRole(roles ...domain.Rol)` como descrito en `plan.md §5.2`.
- [x] Tests: admin puede, mesero no puede en ruta de admin → 403. Cocinero puede en `/kds/`.
- [x] Aplicar `RequireRole` en el registro de TODAS las rutas en `cmd/main.go` según la matriz de §3.2 de `spec.md`.

**Criterio de aceptación:** Tests de RBAC pasan. Un usuario con rol incorrecto recibe 403 en rutas restringidas.

---

### TAREA-14 · Middlewares de seguridad transversal
**Archivos nuevos:**
- `internal/adapters/inbound/shared/middleware/security_headers.go`
- `internal/adapters/inbound/shared/middleware/csrf.go`
- `internal/adapters/inbound/shared/middleware/rate_limiter.go`

**Estimación:** 1 día

- [x] **SecurityHeaders**: añadir `X-Content-Type-Options`, `X-Frame-Options`, `Content-Security-Policy`, `Referrer-Policy` en cada respuesta. Aplicar como primer middleware global.
- [x] **CSRF**: generar token CSRF vinculado a la sesión (HMAC-SHA256 del token de sesión + salt). Validar en `POST|PUT|DELETE`. Exponer helper de template para `{{csrfField}}`.
- [x] **RateLimiter**: contador en memoria por IP (mapa con `sync.RWMutex`). Si supera `RATE_LIMIT_MAX_ATTEMPTS` en `RATE_LIMIT_WINDOW_MINUTES` → 429. Limpiar entradas expiradas en goroutine background.
- [x] Tests unitarios para cada middleware.

**Criterio de aceptación:** Headers de seguridad presentes en todas las respuestas. CSRF bloquea requests sin token. Rate limiter devuelve 429 tras 5 intentos.

---

## Fase 7 — Integración en UI Existente (Días 11–12 · 1 día)

### TAREA-15 · Renderizado condicional de navegación
**Archivos afectados:**
- `internal/adapters/inbound/admin/templates/admin.html`
- `internal/adapters/inbound/kitchen/templates/kds.html`
- Todos los handlers que renderizan plantillas (`admin/handlers.go`, etc.)

**Estimación:** 1 día

- [x] Añadir campo `UsuarioActual *domain.Usuario` a los structs de datos de plantilla existentes.
- [x] Extraer `UsuarioActual` del contexto HTTP en todos los handlers que renderizan HTML.
- [x] Actualizar `admin.html` con bloques `{{if eq .UsuarioActual.Rol "admin"}}...{{end}}` para ocultar menús de usuarios y configuración a no-admins.
- [x] Actualizar `kds.html` con la información del cocinero autenticado.
- [x] Añadir botón "Cerrar Sesión" (POST /logout + CSRF token) en el header de todas las interfaces.
- [x] Verificar que el renderizado es correcto para cada rol manualmente.

**Criterio de aceptación:** La barra de navegación muestra solo las opciones permitidas según el rol del usuario autenticado.

---

## Fase 8 — Admin de Usuarios (Día 12 · 1 día)

### TAREA-16 · Pantalla de gestión de usuarios (solo Admin)
**Archivos nuevos o afectados:**
- `internal/adapters/inbound/admin/handlers.go` (añadir handlers de usuarios)
- Nuevas plantillas: `admin/templates/usuarios.html`

**Estimación:** 1 día

- [x] `GET /admin/usuarios`: listar todos los usuarios (sin hash). Renderizar tabla con rol, estado activo, fecha creación.
- [x] `GET /admin/usuarios/nuevo`: formulario de creación con campos nombre, email, contraseña, rol.
- [x] `POST /admin/usuarios`: crear usuario vía `authSvc.CrearUsuario`. Redirigir a lista con mensaje de éxito.
- [x] `PUT /admin/usuarios/:id` (o `POST /admin/usuarios/:id/modificar`): cambiar rol o desactivar.
- [x] Proteger todas estas rutas con `RequireRole(domain.RolAdmin)`.
- [x] Verificar que el hash no aparece en ningún template ni respuesta JSON.

**Criterio de aceptación:** Admin puede crear, ver y desactivar usuarios. Gerente recibe 403 en estas rutas.

---

## Fase 9 — Seeder del Admin Inicial (Día 12 · 0.5 días)

### TAREA-17 · Bootstrap del usuario administrador inicial
**Archivo afectado:** `internal/adapters/outbound/sqlite/seeder.go` (o nuevo `internal/adapters/outbound/sqlite/auth_seeder.go`)

**Estimación:** 0.5 días

- [x] Al iniciar la aplicación, si `usuarios` está vacía y `ADMIN_BOOTSTRAP_EMAIL` y `ADMIN_BOOTSTRAP_PASSWORD` están configuradas en el entorno:
  - Llamar `authSvc.CrearUsuario` con esos datos y rol `admin`.
  - Loggear: `"Usuario administrador inicial creado: {email}"`.
- [x] Si ya existe al menos un usuario, omitir el seeder silenciosamente.
- [x] Verificar que la contraseña no se loggea en ningún punto.

**Criterio de aceptación:** Primera ejecución con variables de entorno crea el admin. Segunda ejecución no duplica.

---

## Fase 10 — Limpieza de Sesiones (Background Goroutine)

### TAREA-18 · Job de limpieza de sesiones expiradas
**Archivo afectado:** `cmd/main.go`

**Estimación:** 0.5 días

- [x] En `cmd/main.go`, tras inicializar los servicios, lanzar una goroutine que ejecuta `authSvc.LimpiarSesionesExpiradas(ctx)` cada `SESSION_CLEANUP_INTERVAL` minutos.
- [x] Usar `time.NewTicker` con el contexto del servidor para cancelación limpia al apagar.
- [x] Loggear el número de sesiones eliminadas en cada ciclo.

**Criterio de aceptación:** Las sesiones expiradas se eliminan automáticamente. No hay goroutine leak al apagar el servidor.

---

## Fase 11 — Pruebas de Seguridad (Días 13–14 · 1.5 días)

### TAREA-19 · Tests de integración de autenticación
**Archivos nuevos:**
- `internal/adapters/inbound/auth/handler_test.go`
- `internal/core/services/auth_service_test.go`

**Estimación:** 1 día

- [x] Test: login exitoso → cookie presente → redirige al área del rol.
- [x] Test: login fallido → 401 → sin cookie.
- [x] Test: token inválido → redirige a /login.
- [x] Test: token expirado → redirige a /login?expired=1.
- [x] Test: rol insuficiente → 403.
- [x] Test: rate limiter → 429 tras N intentos fallidos.
- [x] Test: CSRF ausente → 403 en POST.
- [x] Test: audit log se crea en login exitoso y fallido.
- [x] `go test -v -race ./...` sin data races.

**Criterio de aceptación:** Todos los tests de seguridad pasan con `go test -race`.

---

### TAREA-20 · Validación final y build de producción
**Estimación:** 0.5 días

- [x] `go build ./...` — cero errores.
- [x] `go vet ./...` — cero advertencias.
- [x] `go test -v -race ./...` — 100% verde.
- [x] Compilar binario: `go build -ldflags="-s -w" -o bin/pos_core.exe ./cmd/main.go`.
- [x] Verificar tamaño del binario < 15MB (límite del proyecto).
- [x] Arrancar servidor y probar manualmente los 5 flujos de rol.
- [x] Actualizar `MEMORY.md` con el estado de la Spec 002.

**Criterio de aceptación:** Binario < 15MB. Todos los tests verdes. MEMORY.md actualizado.

---

## Resumen de Estimaciones

| Fase | Tareas | Estimación |
|---|---|---|
| Fase 0 — Preparación | TAREA-00 | 0.5 días |
| Fase 1 — Base de Datos | TAREA-01, TAREA-02 | 1.5 días |
| Fase 2 — Core Domain | TAREA-03, TAREA-04 | 1.0 día |
| Fase 3 — Ports | TAREA-05 | 0.5 días |
| Fase 4 — Servicios | TAREA-06, TAREA-07 | 2.0 días |
| Fase 5 — Adaptadores SQLite | TAREA-08, TAREA-09, TAREA-10 | 2.0 días |
| Fase 6 — Adaptadores Entrada | TAREA-11, TAREA-12, TAREA-13, TAREA-14 | 3.0 días |
| Fase 7 — UI Existente | TAREA-15 | 1.0 día |
| Fase 8 — Admin Usuarios | TAREA-16 | 1.0 día |
| Fase 9 — Seeder | TAREA-17 | 0.5 días |
| Fase 10 — Background Job | TAREA-18 | 0.5 días |
| Fase 11 — Pruebas | TAREA-19, TAREA-20 | 1.5 días |
| **TOTAL** | **20 tareas** | **~14.5 días** |

---

## Diagrama de Dependencias entre Tareas

```
TAREA-00 (crypto dep)
    │
    ▼
TAREA-01 (SQL migration)
    │
    ▼
TAREA-02 (bootstrap DB)
    │
    ▼
TAREA-03 (domain entities)
    │
    ├──▶ TAREA-04 (domain tests)
    │
    ▼
TAREA-05 (ports)
    │
    ├──▶ TAREA-06 (AuthService)
    │        │
    │        └──▶ TAREA-07 (AuditoriaService)
    │
    └──▶ TAREA-08, TAREA-09, TAREA-10 (repos SQLite)
              │
              ▼
         TAREA-11 (login handler)
              │
         TAREA-12 (auth middleware)
              │
         TAREA-13 (rbac middleware)
              │
         TAREA-14 (security middlewares)
              │
              ▼
         TAREA-15 (UI condicional) ──▶ TAREA-16 (admin usuarios)
              │
         TAREA-17 (seeder admin)
              │
         TAREA-18 (cleanup job)
              │
              ▼
         TAREA-19 (tests integración)
              │
              ▼
         TAREA-20 (build producción)
```
