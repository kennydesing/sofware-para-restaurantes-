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