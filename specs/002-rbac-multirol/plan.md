# Plan de Arquitectura 002 — RBAC Multirol

**Proyecto:** POS Core — Inventario y Punto de Venta para Restaurantes  
**Vinculado a:** `specs/002-rbac-multirol/spec.md`  
**Estado:** BORRADOR — Fase Futura

---

## 1. Visión General de la Arquitectura

El sistema RBAC se integra en la arquitectura hexagonal existente siguiendo el mismo patrón de capas sin romper ningún contrato actual. La capa `core` permanece 100% aislada; el middleware de seguridad vive exclusivamente en los adaptadores de entrada.

```
┌─────────────────────────────────────────────────────────────────┐
│                    ADAPTADORES DE ENTRADA                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  shared/middleware/                                       │   │
│  │    auth_middleware.go   ← extrae token, valida sesión     │   │
│  │    rbac_middleware.go   ← verifica rol contra ruta        │   │
│  │    security_headers.go  ← headers HTTP de seguridad       │   │
│  │    csrf.go              ← validación CSRF                 │   │
│  │    rate_limiter.go      ← límite de intentos de login     │   │
│  └──────────────────────────────────────────────────────────┘   │
│                          ↓                                       │
│  ┌────────┐ ┌─────────┐ ┌──────┐ ┌─────────┐ ┌─────────────┐  │
│  │ /login │ │ /admin/ │ │/kds/ │ │  /api/  │ │     /       │  │
│  └────────┘ └─────────┘ └──────┘ └─────────┘ └─────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────────┐
│                      CORE (sin cambios)                          │
│  domain/ │ ports/ │ services/   ← InventarioService, PedidoSvc  │
│  + NUEVO: AuthService (port inbound)                             │
│  + NUEVO: SessionService (port inbound)                          │
│  + NUEVO: UsuarioRepository (port outbound)                      │
│  + NUEVO: SesionRepository (port outbound)                       │
│  + NUEVO: AuditoriaRepository (port outbound)                    │
└─────────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────────┐
│                   ADAPTADORES DE SALIDA                          │
│  sqlite/usuario_repo.go   ← CRUD tabla usuarios                  │
│  sqlite/sesion_repo.go    ← CRUD tabla sesiones                  │
│  sqlite/auditoria_repo.go ← INSERT-only tabla auditoria_acciones │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Diseño de Base de Datos

### 2.1 Migración UP — `migrations/000002_auth.up.sql`

```sql
-- ==============================================================================
-- Migration: 000002_auth.up.sql
-- Sistema de Autenticación RBAC para POS Restaurante
-- Argon2id para hashes, sesiones persistentes, auditoría inmutable
-- ==============================================================================

-- ==============================================================================
-- 1. TABLA: USUARIOS
-- Almacena credenciales y rol de cada usuario del sistema.
-- password_hash: formato Argon2id PHC string ($argon2id$v=19$...).
-- ==============================================================================
CREATE TABLE IF NOT EXISTS usuarios (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre       TEXT    NOT NULL,
    email        TEXT    NOT NULL,
    password_hash TEXT   NOT NULL,
    rol          TEXT    NOT NULL CHECK (rol IN ('admin','gerente','cocinero','mesero','cajero')),
    activo       INTEGER NOT NULL DEFAULT 1 CHECK (activo IN (0, 1)),
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_usuarios_email UNIQUE (email)
);

CREATE INDEX IF NOT EXISTS idx_usuarios_email  ON usuarios(email);
CREATE INDEX IF NOT EXISTS idx_usuarios_rol    ON usuarios(rol);
CREATE INDEX IF NOT EXISTS idx_usuarios_activo ON usuarios(activo);

-- ==============================================================================
-- 2. TABLA: SESIONES
-- Token opaco de 64 hex-chars (32 bytes crypto/rand).
-- Una sesión por fila. Limpieza periódica por job interno (goroutine).
-- ==============================================================================
CREATE TABLE IF NOT EXISTS sesiones (
    token       TEXT    PRIMARY KEY,           -- 64 hex chars, 256 bits
    usuario_id  INTEGER NOT NULL,
    ip_origen   TEXT    NOT NULL DEFAULT '',
    user_agent  TEXT    NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at  DATETIME NOT NULL,
    FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sesiones_usuario  ON sesiones(usuario_id);
CREATE INDEX IF NOT EXISTS idx_sesiones_expires  ON sesiones(expires_at);

-- ==============================================================================
-- 3. TABLA: AUDITORIA_ACCIONES
-- Registro inmutable de eventos críticos del sistema (INSERT-only).
-- No se proveerán UPDATE ni DELETE desde ninguna capa de la aplicación.
-- ==============================================================================
CREATE TABLE IF NOT EXISTS auditoria_acciones (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    usuario_id   INTEGER,                       -- NULL si es acción del sistema
    accion       TEXT    NOT NULL CHECK (accion IN (
                     'LOGIN', 'LOGOUT', 'LOGIN_FALLIDO',
                     'CREAR_USUARIO', 'DESACTIVAR_USUARIO', 'MODIFICAR_ROL',
                     'CAMBIO_ESTADO_PEDIDO', 'COBRO_PEDIDO', 'CANCELAR_PEDIDO',
                     'REGISTRAR_COMPRA', 'REGISTRAR_MERMA', 'AJUSTE_MANUAL',
                     'CONTEO_FISICO'
                 )),
    entidad_tipo TEXT    NOT NULL DEFAULT '',   -- 'pedido', 'usuario', 'insumo', ''
    entidad_id   INTEGER,                       -- ID del recurso afectado (nullable)
    detalle      TEXT    NOT NULL DEFAULT '',   -- contexto libre (estado anterior→nuevo, monto, etc.)
    ip_origen    TEXT    NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_auditoria_usuario   ON auditoria_acciones(usuario_id);
CREATE INDEX IF NOT EXISTS idx_auditoria_accion    ON auditoria_acciones(accion);
CREATE INDEX IF NOT EXISTS idx_auditoria_created   ON auditoria_acciones(created_at);
CREATE INDEX IF NOT EXISTS idx_auditoria_entidad   ON auditoria_acciones(entidad_tipo, entidad_id);
```

### 2.2 Migración DOWN — `migrations/000002_auth.down.sql`

```sql
-- ==============================================================================
-- Migration: 000002_auth.down.sql
-- Rollback completo de la migración RBAC
-- ==============================================================================
DROP INDEX IF EXISTS idx_auditoria_entidad;
DROP INDEX IF EXISTS idx_auditoria_created;
DROP INDEX IF EXISTS idx_auditoria_accion;
DROP INDEX IF EXISTS idx_auditoria_usuario;
DROP TABLE IF EXISTS auditoria_acciones;

DROP INDEX IF EXISTS idx_sesiones_expires;
DROP INDEX IF EXISTS idx_sesiones_usuario;
DROP TABLE IF EXISTS sesiones;

DROP INDEX IF EXISTS idx_usuarios_activo;
DROP INDEX IF EXISTS idx_usuarios_rol;
DROP INDEX IF EXISTS idx_usuarios_email;
DROP TABLE IF EXISTS usuarios;
```

---

## 3. Core Domain — Nuevas Entidades

### 3.1 `internal/core/domain/usuario.go`

```go
package domain

import "time"

// Rol representa el conjunto de permisos de un usuario en el sistema POS.
type Rol string

const (
    RolAdmin    Rol = "admin"
    RolGerente  Rol = "gerente"
    RolCocinero Rol = "cocinero"
    RolMesero   Rol = "mesero"
    RolCajero   Rol = "cajero"
)

// RolesValidos es el conjunto cerrado de roles permitidos en el sistema.
var RolesValidos = map[Rol]bool{
    RolAdmin: true, RolGerente: true,
    RolCocinero: true, RolMesero: true, RolCajero: true,
}

// Usuario es la entidad pura de dominio para un operador del sistema.
// No contiene lógica de persistencia ni de HTTP.
type Usuario struct {
    ID           int
    Nombre       string
    Email        string
    PasswordHash string // Formato PHC Argon2id — NUNCA se expone en respuestas HTTP
    Rol          Rol
    Activo       bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// EsActivo valida que la cuenta puede iniciar sesión.
func (u *Usuario) EsActivo() bool { return u.Activo }

// TieneRol comprueba si el usuario posee exactamente el rol indicado.
func (u *Usuario) TieneRol(rol Rol) bool { return u.Rol == rol }

// PuedeAcceder verifica si el usuario tiene alguno de los roles permitidos.
func (u *Usuario) PuedeAcceder(rolesPermitidos []Rol) bool {
    for _, r := range rolesPermitidos {
        if u.Rol == r {
            return true
        }
    }
    return false
}
```

### 3.2 `internal/core/domain/sesion.go`

```go
package domain

import "time"

// Sesion representa una sesión autenticada activa en el sistema.
type Sesion struct {
    Token     string    // 64 hex chars — 32 bytes de crypto/rand
    UsuarioID int
    IPOrigen  string
    UserAgent string
    CreatedAt time.Time
    ExpiresAt time.Time
}

// EstaVigente comprueba si la sesión no ha expirado.
func (s *Sesion) EstaVigente() bool {
    return time.Now().Before(s.ExpiresAt)
}
```

### 3.3 `internal/core/domain/auditoria.go`

```go
package domain

import "time"

// AccionAuditoria define el catálogo cerrado de eventos auditables.
type AccionAuditoria string

const (
    AccionLogin              AccionAuditoria = "LOGIN"
    AccionLogout             AccionAuditoria = "LOGOUT"
    AccionLoginFallido       AccionAuditoria = "LOGIN_FALLIDO"
    AccionCrearUsuario       AccionAuditoria = "CREAR_USUARIO"
    AccionDesactivarUsuario  AccionAuditoria = "DESACTIVAR_USUARIO"
    AccionModificarRol       AccionAuditoria = "MODIFICAR_ROL"
    AccionCambioEstadoPedido AccionAuditoria = "CAMBIO_ESTADO_PEDIDO"
    AccionCobroPedido        AccionAuditoria = "COBRO_PEDIDO"
    AccionCancelarPedido     AccionAuditoria = "CANCELAR_PEDIDO"
    AccionRegistrarCompra    AccionAuditoria = "REGISTRAR_COMPRA"
    AccionRegistrarMerma     AccionAuditoria = "REGISTRAR_MERMA"
    AccionAjusteManual       AccionAuditoria = "AJUSTE_MANUAL"
    AccionConteoFisico       AccionAuditoria = "CONTEO_FISICO"
)

// EventoAuditoria es el registro inmutable de una acción crítica del sistema.
type EventoAuditoria struct {
    ID          int
    UsuarioID   *int            // nil si es acción del sistema
    Accion      AccionAuditoria
    EntidadTipo string          // "pedido", "usuario", "insumo"
    EntidadID   *int            // nil si no aplica
    Detalle     string          // contexto libre: "PENDIENTE→EN_COCINA", "$1500.00", etc.
    IPOrigen    string
    CreatedAt   time.Time
}
```

---

## 4. Core Ports — Nuevas Interfaces

### 4.1 Inbound Ports — `internal/core/ports/auth_ports.go`

```go
package ports

import (
    "context"
    "pos-inventory/internal/core/domain"
)

// ---- DTOs de Entrada ----

// LoginRequest contiene las credenciales enviadas por el usuario.
type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"` // contraseña en texto plano (solo en tránsito)
    IPOrigen string `json:"-"`        // extraído del request HTTP, no del body
}

// CrearUsuarioRequest contiene los datos para registrar un nuevo operador.
type CrearUsuarioRequest struct {
    Nombre   string      `json:"nombre"`
    Email    string      `json:"email"`
    Password string      `json:"password"` // texto plano — se hashea antes de persistir
    Rol      domain.Rol  `json:"rol"`
}

// ModificarUsuarioRequest permite cambiar el rol o el estado activo de un usuario.
type ModificarUsuarioRequest struct {
    Rol    *domain.Rol `json:"rol,omitempty"`
    Activo *bool       `json:"activo,omitempty"`
}

// ---- Inbound Ports ----

// AuthService define los casos de uso de autenticación y gestión de usuarios.
// Implementado en internal/core/services/auth_service.go
type AuthService interface {
    // Login verifica credenciales y crea una sesión persistente.
    // Retorna la sesión creada (con token) o error (ErrCredencialesInvalidas, ErrCuentaInactiva).
    Login(ctx context.Context, req LoginRequest) (*domain.Sesion, error)

    // Logout invalida la sesión asociada al token.
    Logout(ctx context.Context, token string) error

    // ValidarSesion verifica que el token sea válido y no esté expirado.
    // Retorna el Usuario completo (para poblar el contexto HTTP) o error.
    ValidarSesion(ctx context.Context, token string) (*domain.Usuario, error)

    // CrearUsuario registra un nuevo usuario. Solo el rol admin puede invocar este caso de uso.
    CrearUsuario(ctx context.Context, req CrearUsuarioRequest) (*domain.Usuario, error)

    // ModificarUsuario actualiza el rol o estado activo de un usuario existente.
    ModificarUsuario(ctx context.Context, usuarioID int, req ModificarUsuarioRequest) (*domain.Usuario, error)

    // ListarUsuarios devuelve todos los usuarios del sistema (sin hash de contraseña).
    ListarUsuarios(ctx context.Context) ([]*domain.Usuario, error)

    // LimpiarSesionesExpiradas elimina tokens cuyo expires_at ha vencido.
    LimpiarSesionesExpiradas(ctx context.Context) (int64, error)
}

// AuditoriaService define los casos de uso para el registro y consulta de eventos de auditoría.
type AuditoriaService interface {
    // RegistrarEvento inserta un evento inmutable en el log de auditoría.
    RegistrarEvento(ctx context.Context, evento *domain.EventoAuditoria) error

    // ListarEventos devuelve eventos filtrados por rango de fecha y usuario, paginados.
    ListarEventos(ctx context.Context, filtro FiltroAuditoria) ([]*domain.EventoAuditoria, error)
}

// FiltroAuditoria encapsula los parámetros de búsqueda para el visor de auditoría.
type FiltroAuditoria struct {
    UsuarioID *int
    Desde     string // formato "2006-01-02"
    Hasta     string // formato "2006-01-02"
    Pagina    int    // 1-indexed
    PorPagina int    // default 50
}
```

### 4.2 Outbound Ports — `internal/core/ports/auth_outbound_ports.go`

```go
package ports

import (
    "context"
    "pos-inventory/internal/core/domain"
)

// UsuarioRepository define el contrato de persistencia para usuarios.
// Implementado en internal/adapters/outbound/sqlite/usuario_repo.go
type UsuarioRepository interface {
    GetByID(ctx context.Context, id int) (*domain.Usuario, error)
    GetByEmail(ctx context.Context, email string) (*domain.Usuario, error)
    GetAll(ctx context.Context) ([]*domain.Usuario, error)
    Create(ctx context.Context, u *domain.Usuario) error
    Update(ctx context.Context, u *domain.Usuario) error // solo para rol/activo — nunca para hash
}

// SesionRepository define el contrato de persistencia para sesiones activas.
// Implementado en internal/adapters/outbound/sqlite/sesion_repo.go
type SesionRepository interface {
    Create(ctx context.Context, s *domain.Sesion) error
    GetByToken(ctx context.Context, token string) (*domain.Sesion, error)
    DeleteByToken(ctx context.Context, token string) error
    DeleteByUsuarioID(ctx context.Context, usuarioID int) error   // para SINGLE_SESSION
    DeleteExpired(ctx context.Context) (int64, error)             // limpieza periódica
}

// AuditoriaRepository define el contrato INSERT-only para el log de auditoría.
// Implementado en internal/adapters/outbound/sqlite/auditoria_repo.go
// IMPORTANTE: No se define Update ni Delete — violación de RNF-05.
type AuditoriaRepository interface {
    Create(ctx context.Context, e *domain.EventoAuditoria) error
    GetFiltered(ctx context.Context, filtro FiltroAuditoria) ([]*domain.EventoAuditoria, error)
}
```

---

## 5. Adaptadores de Entrada — Middleware

### 5.1 `internal/adapters/inbound/shared/middleware/auth_middleware.go`

Responsabilidad: interceptar TODAS las rutas protegidas, extraer la cookie `pos_session`, validar el token vía `AuthService.ValidarSesion`, e inyectar `*domain.Usuario` en el contexto HTTP.

```go
// Pseudocódigo — estructura del middleware

func AuthMiddleware(authSvc ports.AuthService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("pos_session")
            if err != nil {
                redirectOrUnauthorized(w, r) // HTML→/login | JSON→401
                return
            }
            usuario, err := authSvc.ValidarSesion(r.Context(), cookie.Value)
            if err != nil {
                clearCookie(w)
                redirectOrUnauthorized(w, r)
                return
            }
            // Inyectar usuario en contexto para handlers y RBAC downstream
            ctx := context.WithValue(r.Context(), ctxKeyUsuario, usuario)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### 5.2 `internal/adapters/inbound/shared/middleware/rbac_middleware.go`

Responsabilidad: verificar que el usuario en contexto posea uno de los roles permitidos para la ruta.

```go
// Pseudocódigo — estructura del middleware

func RequireRole(rolesPermitidos ...domain.Rol) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            usuario := UsuarioFromContext(r.Context())
            if usuario == nil || !usuario.PuedeAcceder(rolesPermitidos) {
                forbiddenOrJSON(w, r) // HTML→403.html | JSON→{"error":"Acceso denegado"}
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Uso en cmd/main.go al registrar rutas:
// mux.Handle("/admin/usuarios", AuthMiddleware(authSvc)(RequireRole(domain.RolAdmin)(usuariosHandler)))
// mux.Handle("/kds/", AuthMiddleware(authSvc)(RequireRole(domain.RolAdmin, domain.RolCocinero)(kdsAdapter)))
```

### 5.3 Composición de Middlewares en `cmd/main.go`

```
Solicitud HTTP
    │
    ▼
SecurityHeaders   ← siempre, en todas las rutas
    │
    ▼
RateLimiter       ← solo en POST /login
    │
    ▼
AuthMiddleware    ← rutas protegidas (todo excepto /login, /static/*)
    │
    ▼
CSRF              ← POST|PUT|DELETE con cookie presente
    │
    ▼
RequireRole(...)  ← por ruta específica
    │
    ▼
Handler           ← lógica de negocio existente (sin cambios)
```

---

## 6. Adaptadores de Salida — Repositorios SQLite

### 6.1 `internal/adapters/outbound/sqlite/usuario_repo.go`

- `GetByEmail`: query con índice `idx_usuarios_email`. Retorna `ErrNotFound` si no existe.
- `Create`: inserta con hash ya computado. Retorna `ErrEmailDuplicado` si viola unique.
- `Update`: solo actualiza columnas `rol`, `activo`, `updated_at`. Nunca toca `password_hash`.

### 6.2 `internal/adapters/outbound/sqlite/sesion_repo.go`

- `Create`: inserta token + usuario_id + expires_at.
- `GetByToken`: query `WHERE token = ? AND expires_at > datetime('now')`. Si expira → `ErrSesionExpirada`.
- `DeleteExpired`: `DELETE FROM sesiones WHERE expires_at < datetime('now')`. Retorna filas afectadas.

### 6.3 `internal/adapters/outbound/sqlite/auditoria_repo.go`

- `Create`: solo `INSERT INTO auditoria_acciones (...)`. Nunca UPDATE/DELETE.
- `GetFiltered`: query con filtros opcionales de `usuario_id`, rango de fechas y paginación `LIMIT ? OFFSET ?`.

---

## 7. Adaptador de Entrada — Login Handler

Nuevo adaptador independiente en `internal/adapters/inbound/auth/`:

```
internal/adapters/inbound/auth/
    handler.go        ← HandleLogin, HandleLogout, HandleGetLogin (formulario)
    templates/
        login.html    ← formulario de login con diseño coherente al POS
```

**Flujo de HandleLogin (POST /login):**

```
1. Parsear form: email, password
2. Extraer IP de r.RemoteAddr (o X-Forwarded-For en producción)
3. RateLimiter.Check(ip) → 429 si excede límite
4. authSvc.Login(ctx, LoginRequest{Email, Password, IP})
   ├── UsuarioRepo.GetByEmail → ErrNotFound → 401 + retardo 300ms
   ├── argon2.CompareHashAndPassword → fallo → 401 + retardo 300ms
   │   + AuditoriaRepo.Create(AccionLoginFallido)
   ├── usuario.Activo == false → 403
   └── OK:
       ├── generar token (32 bytes crypto/rand → hex)
       ├── SesionRepo.Create(token, usuario.ID, ip, expires=+8h)
       ├── AuditoriaRepo.Create(AccionLogin)
       └── Set-Cookie: pos_session={token}; HttpOnly; SameSite=Strict
5. Redirigir según rol:
   admin/gerente/cajero → /admin/
   cocinero             → /kds/
   mesero               → /?mesa=1
```

---

## 8. Interfaz de Usuario — Login y Renderizado Condicional

### 8.1 Página de Login (`/login`)

- Formulario simple: email + contraseña + botón "Entrar".
- Mensajes de error inline (sin revelar causa específica).
- Compatible con el diseño visual existente del POS.
- Sin JavaScript complejo — HTML nativo + CSS.

### 8.2 Barra de Navegación Condicional

Todas las plantillas base existentes (`admin.html`, `kds.html`) reciben una variable `UsuarioActual *domain.Usuario` en el `TemplateData`. Con Go templates se condiciona el menú:

```html
{{if eq .UsuarioActual.Rol "admin"}}
    <a href="/admin/usuarios">Usuarios</a>
    <a href="/admin/config">Configuración</a>
{{end}}

{{if or (eq .UsuarioActual.Rol "admin") (eq .UsuarioActual.Rol "gerente")}}
    <a href="/admin/inventario">Inventario</a>
    <a href="/admin/auditoria">Auditoría</a>
{{end}}
```

### 8.3 Redirección por Rol tras Login

| Rol | Redirección |
|---|---|
| `admin` | `/admin/` |
| `gerente` | `/admin/` |
| `cajero` | `/admin/pedidos` |
| `cocinero` | `/kds/` |
| `mesero` | `/?mesa=1` (primera mesa disponible) |

---

## 9. Errores de Dominio Nuevos

Añadir a `internal/core/domain/errors.go`:

```go
var (
    ErrCredencialesInvalidas = errors.New("credenciales inválidas")
    ErrCuentaInactiva        = errors.New("cuenta desactivada")
    ErrSesionExpirada        = errors.New("sesión expirada")
    ErrSesionInvalida        = errors.New("sesión inválida o no encontrada")
    ErrEmailDuplicado        = errors.New("el email ya está registrado")
    ErrRolInvalido           = errors.New("rol no reconocido")
    ErrAccesoDenegado        = errors.New("acceso denegado: rol insuficiente")
    ErrPasswordInsegura      = errors.New("la contraseña no cumple los requisitos mínimos")
)
```

---

## 10. Dependencia Nueva — `golang.org/x/crypto`

Argon2id requiere añadir `golang.org/x/crypto` a `go.mod`:

```
golang.org/x/crypto v0.27.0
```

> ⚠️ **Requiere aprobación previa según AGENTS.md** antes de ejecutar `go get golang.org/x/crypto`.  
> Es la única dependencia nueva. No hay frameworks web, no hay ORM.

---

## 11. Variables de Entorno Nuevas

| Variable | Tipo | Default | Descripción |
|---|---|---|---|
| `SESSION_TTL_HOURS` | int | `8` | Duración de sesión en horas |
| `SESSION_CLEANUP_INTERVAL` | int | `60` | Minutos entre limpiezas de sesiones expiradas |
| `SINGLE_SESSION` | bool | `false` | Si true, invalida sesiones anteriores al hacer login |
| `RATE_LIMIT_MAX_ATTEMPTS` | int | `5` | Intentos fallidos antes del bloqueo por IP |
| `RATE_LIMIT_WINDOW_MINUTES` | int | `15` | Ventana de tiempo para rate limiting |
| `ADMIN_BOOTSTRAP_EMAIL` | string | — | Email del admin inicial (seeder del primer arranque) |
| `ADMIN_BOOTSTRAP_PASSWORD` | string | — | Contraseña del admin inicial (no se guarda en texto plano) |
