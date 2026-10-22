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