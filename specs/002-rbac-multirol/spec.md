# Especificación 002 — Sistema de Autenticación y Control de Acceso Basado en Roles (RBAC)

**Proyecto:** POS Core — Inventario y Punto de Venta para Restaurantes  
**Versión:** 1.0.0  
**Estado:** BORRADOR — Fase Futura (no modifica el MVP actual)  
**Fecha:** 2026-10-12  

---

## 1. Alcance

Esta especificación define los requisitos funcionales y de seguridad para introducir autenticación multiusuario, gestión de sesiones persistentes y control de acceso basado en roles (RBAC) en el sistema POS. El alcance cubre exclusivamente la capa de seguridad perimetral; **no altera la lógica de negocio de inventario ni el motor de escandallo existente.**

El sistema actual (`spec/001`) expone sus interfaces sin autenticación. Esta especificación define la capa de protección que se añade delante de todas las interfaces existentes (`/`, `/kds/`, `/admin/`, `/api/`).

---

## 2. Glosario

| Término | Definición |
|---|---|
| **Usuario** | Persona registrada en el sistema con email, contraseña y rol asignado |
| **Rol** | Conjunto de permisos predefinido que determina qué rutas y acciones puede ejecutar un usuario |
| **Sesión** | Token opaco de 256 bits asociado a un usuario autenticado, con TTL configurable |
| **Middleware** | Función HTTP que intercepta la solicitud antes de llegar al handler y valida la identidad y el rol |
| **Auditoría** | Registro inmutable de acciones críticas: quién ejecutó qué acción, sobre qué entidad y cuándo |
| **RBAC** | Role-Based Access Control — modelo de autorización basado en roles, no en usuarios individuales |
| **EARS** | Easy Approach to Requirements Syntax — sintaxis estructurada para redacción de requisitos |

---

## 3. Roles y Matriz de Acceso

### 3.1 Definición de Roles

| Rol | Código | Descripción |
|---|---|---|
| Propietario/Admin | `admin` | Control total del sistema |
| Gerente | `gerente` | Operaciones diarias completas, sin configuración |
| Cocinero | `cocinero` | Exclusivo al KDS — vista y actualización de estados |
| Mesero | `mesero` | Toma de pedidos en interfaz de mesa |
| Cajero | `cajero` | Cobros, cierre de caja, sin inventario ni configuración |

### 3.2 Matriz de Permisos por Ruta

| Ruta | admin | gerente | cajero | mesero | cocinero |
|---|:---:|:---:|:---:|:---:|:---:|
| `GET /` (cliente QR + mesa) | ✅ | ✅ | ✅ | ✅ | ❌ |
| `POST /api/pedidos` | ✅ | ✅ | ✅ | ✅ | ❌ |
| `GET /kds/` | ✅ | ✅ | ❌ | ❌ | ✅ |
| `PUT /api/pedidos/:id/estado` (EN_COCINA→LISTO) | ✅ | ✅ | ❌ | ❌ | ✅ |
| `GET /admin/` | ✅ | ✅ | ✅* | ❌ | ❌ |
| `GET /admin/inventario` | ✅ | ✅ | ❌ | ❌ | ❌ |
| `POST /api/inventario/compra` | ✅ | ✅ | ❌ | ❌ | ❌ |
| `POST /api/inventario/merma` | ✅ | ✅ | ❌ | ❌ | ❌ |
| `GET /admin/pedidos` (todos) | ✅ | ✅ | ✅ | ❌ | ❌ |
| `POST /api/caja/cierre` | ✅ | ✅ | ✅ | ❌ | ❌ |
| `GET /admin/auditoria` | ✅ | ✅ | ❌ | ❌ | ❌ |
| `GET /admin/usuarios` | ✅ | ❌ | ❌ | ❌ | ❌ |
| `POST /admin/usuarios` | ✅ | ❌ | ❌ | ❌ | ❌ |
| `GET /admin/config` | ✅ | ❌ | ❌ | ❌ | ❌ |

> `*` Cajero solo accede a sub-rutas de pedidos y caja dentro de `/admin/`.

---

## 4. Requisitos Funcionales (Sintaxis EARS)

### RF-01 — Registro y Autenticación de Usuarios

**RF-01.1** — Registro de usuario (solo Admin)  
`WHEN` el usuario con rol `admin` envía una solicitud `POST /admin/usuarios` con `{email, password, rol}` válidos,  
`THE SYSTEM SHALL` hashear la contraseña con **Argon2id** (parámetros: `memory=64MB, iterations=3, parallelism=2`), persistir el usuario en la tabla `usuarios` con `activo=1` y retornar `201 Created` con el usuario creado (sin incluir el hash en la respuesta).

**RF-01.2** — Contraseña mínima  
`WHERE` el campo `password` en la solicitud de registro tiene menos de 8 caracteres o no contiene al menos una mayúscula y un número,  
`THE SYSTEM SHALL` rechazar la solicitud con `400 Bad Request` y un mensaje de error descriptivo, sin persistir ningún dato.

**RF-01.3** — Email único  
`IF` el `email` enviado ya existe en la tabla `usuarios`,  
`THEN THE SYSTEM SHALL` retornar `409 Conflict` sin revelar si el email está o no registrado en el mensaje al cliente (anti-enumeración).

**RF-01.4** — Inicio de sesión  
`WHEN` cualquier agente externo envía `POST /login` con `{email, password}` correctos y el usuario tiene `activo=1`,  
`THE SYSTEM SHALL` verificar la contraseña contra el hash Argon2id, generar un token de sesión criptográficamente seguro (32 bytes de `crypto/rand`, codificado en hex), persistirlo en `sesiones` con `expires_at = NOW() + 8h`, establecerlo en una cookie HTTP-Only (`Secure`, `SameSite=Strict`) y redirigir al usuario a la ruta raíz de su rol.

**RF-01.5** — Credenciales incorrectas  
`WHEN` un agente externo envía `POST /login` con email no existente O contraseña incorrecta,  
`THE SYSTEM SHALL` retornar siempre `401 Unauthorized` con el mensaje genérico `"Credenciales inválidas"`, introduciendo un retardo mínimo de 300ms para mitigar ataques de temporización.

**RF-01.6** — Cuenta inactiva  
`IF` el usuario existe pero tiene `activo=0`,  
`THEN THE SYSTEM SHALL` retornar `403 Forbidden` con el mensaje `"Cuenta desactivada. Contacte al administrador"`.

---

### RF-02 — Gestión de Sesiones

**RF-02.1** — Validación de sesión en cada solicitud  
`WHILE` el servidor está operativo, `WHEN` llega cualquier solicitud a una ruta protegida,  
`THE SYSTEM SHALL` extraer el token de la cookie de sesión, consultar la tabla `sesiones` para verificar que el token exista y que `expires_at > NOW()`, y si es válido, cargar el `usuario_id` y su `rol` en el contexto de la solicitud (`context.WithValue`).

**RF-02.2** — Sesión expirada  
`IF` el token de sesión existe en `sesiones` pero `expires_at <= NOW()`,  
`THEN THE SYSTEM SHALL` eliminar el registro de la tabla `sesiones`, limpiar la cookie del cliente y redirigir a `GET /login?expired=1`.

**RF-02.3** — Token inválido o ausente  
`IF` la cookie de sesión está ausente o el token no existe en la tabla `sesiones`,  
`THEN THE SYSTEM SHALL` redirigir a `GET /login` para solicitudes de navegador (`Accept: text/html`) o retornar `401 Unauthorized` en JSON para solicitudes de API (`Accept: application/json`).

**RF-02.4** — Cierre de sesión  
`WHEN` un usuario autenticado envía `POST /logout`,  
`THE SYSTEM SHALL` eliminar el registro de `sesiones` correspondiente al token activo, borrar la cookie del cliente y redirigir a `GET /login`.

**RF-02.5** — Limpieza de sesiones expiradas  
`WHILE` el servidor está operativo, `WHEN` ha transcurrido el intervalo configurable de limpieza (por defecto 1 hora),  
`THE SYSTEM SHALL` ejecutar `DELETE FROM sesiones WHERE expires_at < datetime('now')` para evitar acumulación de tokens huérfanos.

**RF-02.6** — Sesión única por usuario (configurable)  
`IF` la variable de entorno `SINGLE_SESSION=true` está configurada y el usuario inicia sesión desde un segundo dispositivo,  
`THEN THE SYSTEM SHALL` invalidar todas las sesiones previas del usuario antes de crear la nueva.

---

### RF-03 — Control de Acceso por Roles (Middleware RBAC)

**RF-03.1** — Autorización en ruta protegida  
`WHEN` un usuario autenticado intenta acceder a una ruta que requiere un rol específico,  
`THE SYSTEM SHALL` comparar el `rol` del usuario (extraído del contexto de la solicitud) contra los roles permitidos para esa ruta, y si no coincide, retornar `403 Forbidden` (HTML para navegador, JSON para API).

**RF-03.2** — Roles explícitos por ruta  
`WHERE` una ruta está configurada como accesible por `["admin", "gerente"]`,  
`THE SYSTEM SHALL` autorizar el acceso a cualquier usuario cuyo rol esté listado exactamente en el conjunto permitido para esa ruta, sin herencia implícita entre roles.

**RF-03.3** — Restricción de KDS  
`WHERE` la ruta es `GET /kds/` o cualquier sub-ruta bajo `/kds/`,  
`THE SYSTEM SHALL` autorizar únicamente a usuarios con rol `cocinero` o `admin`, rechazando cualquier otro rol con `403 Forbidden`.

**RF-03.4** — Protección de gestión de usuarios  
`WHERE` la ruta es `GET|POST|PUT|DELETE /admin/usuarios/*`,  
`THE SYSTEM SHALL` autorizar únicamente al rol `admin`, retornando `403 Forbidden` a gerentes, cajeros, meseros y cocineros.

**RF-03.5** — Renderizado condicional en UI  
`WHEN` se renderiza cualquier plantilla HTML que incluye la barra de navegación,  
`THE SYSTEM SHALL` pasar el rol del usuario autenticado como variable de plantilla y mostrar u ocultar los elementos de menú según la matriz de permisos definida en §3.2.

---

### RF-04 — Trazabilidad y Auditoría de Acciones Críticas

**RF-04.1** — Registro de auditoría en movimientos de inventario  
`WHEN` un usuario con rol `admin` o `gerente` ejecuta exitosamente `RegistrarCompra`, `RegistrarMerma` o `AjustarStock`,  
`THE SYSTEM SHALL` incluir el `usuario_id` del ejecutor en el campo `referencia` del registro de `movimientos_inventario`, con el formato `"{referencia_original} | uid:{id}"`.

**RF-04.2** — Auditoría de cambios de estado de pedido  
`WHEN` cualquier usuario ejecuta `ActualizarEstadoPedido`,  
`THE SYSTEM SHALL` insertar un registro en `auditoria_acciones` con: `{usuario_id, accion="CAMBIO_ESTADO_PEDIDO", entidad_tipo="pedido", entidad_id, detalle="{estado_anterior}→{estado_nuevo}", ip_origen, created_at}`.

**RF-04.3** — Auditoría de cobros  
`WHEN` un usuario con rol `cajero` o `admin` registra el cobro de un pedido,  
`THE SYSTEM SHALL` insertar un registro en `auditoria_acciones` con `accion="COBRO_PEDIDO"` y el monto cobrado en `detalle`.

**RF-04.4** — Auditoría de gestión de usuarios  
`WHEN` un usuario `admin` crea, desactiva o modifica un usuario,  
`THE SYSTEM SHALL` insertar un registro en `auditoria_acciones` con `accion` en `{CREAR_USUARIO, DESACTIVAR_USUARIO, MODIFICAR_ROL}` y el `usuario_id` afectado en `entidad_id`.

**RF-04.5** — Consulta de auditoría  
`WHEN` un usuario con rol `admin` o `gerente` envía `GET /admin/auditoria?desde={fecha}&hasta={fecha}&usuario_id={id}`,  
`THE SYSTEM SHALL` retornar los registros de `auditoria_acciones` filtrados por rango de fechas y opcionalmente por usuario, paginados en bloques de 50 registros.

**RF-04.6** — Inmutabilidad de auditoría  
`WHERE` existe cualquier registro en la tabla `auditoria_acciones`,  
`THE SYSTEM SHALL NOT` proveer ningún endpoint ni función que permita su modificación o eliminación. La tabla es de solo inserción (INSERT-only).

---

### RF-05 — Seguridad Transversal

**RF-05.1** — Protección CSRF  
`WHEN` cualquier solicitud `POST|PUT|DELETE` proviene de un navegador (cookie de sesión presente),  
`THE SYSTEM SHALL` validar la presencia y corrección del header `X-CSRF-Token` (valor sincronizado con la sesión), rechazando con `403 Forbidden` si es inválido o ausente.

**RF-05.2** — Rate limiting en login  
`WHEN` más de 5 intentos fallidos de login ocurren desde la misma IP en un periodo de 15 minutos,  
`THE SYSTEM SHALL` bloquear temporalmente las solicitudes de esa IP por 15 minutos, retornando `429 Too Many Requests`.

**RF-05.3** — Sin exposición del hash  
`WHERE` cualquier endpoint retorna datos de un usuario,  
`THE SYSTEM SHALL NOT` incluir el campo `password_hash` en ninguna respuesta HTTP, ni en JSON ni en plantillas HTML.

**RF-05.4** — Headers de seguridad HTTP  
`WHEN` el servidor emite cualquier respuesta HTTP,  
`THE SYSTEM SHALL` incluir los headers: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'self'` y `Referrer-Policy: strict-origin-when-cross-origin`.

---

## 5. Restricciones No Funcionales

| ID | Restricción |
|---|---|
| RNF-01 | La capa `internal/core/` no debe importar `net/http`, `database/sql` ni ningún driver externo |
| RNF-02 | El token de sesión debe tener entropía mínima de 256 bits (`crypto/rand`, no `math/rand`) |
| RNF-03 | Argon2id es el único algoritmo permitido. `bcrypt` y MD5 están explícitamente prohibidos |
| RNF-04 | Las cookies de sesión deben ser `HttpOnly=true`, `Secure=true` (producción) y `SameSite=Strict` |
| RNF-05 | La tabla `auditoria_acciones` no tiene UPDATE ni DELETE en ninguna capa del sistema |
| RNF-06 | El middleware de autenticación debe añadir menos de 2ms de latencia en condiciones normales |
| RNF-07 | Compatible con el modo WAL de SQLite existente sin cambios en pragmas ni configuración de conexión |

---

## 6. Interfaces de Usuario Nuevas

| Ruta | Descripción | Roles |
|---|---|---|
| `GET /login` | Formulario de inicio de sesión | Todos (no autenticado) |
| `POST /login` | Procesa credenciales | Todos (no autenticado) |
| `POST /logout` | Cierre de sesión | Todos (autenticado) |
| `GET /admin/usuarios` | Listado de usuarios del sistema | admin |
| `POST /admin/usuarios` | Crear nuevo usuario | admin |
| `PUT /admin/usuarios/:id` | Modificar rol / desactivar usuario | admin |
| `GET /admin/auditoria` | Visor de log de auditoría | admin, gerente |

---

## 7. Fuera de Alcance (Spec 002)

- Autenticación multifactor (2FA/TOTP) — reservado para Spec 003.
- OAuth2 / SSO con proveedores externos.
- Cifrado de datos en reposo (SQLite encryption).
- Gestión de permisos granulares por recurso (más allá de rutas).
- API pública para terceros (requiere OAuth2).
