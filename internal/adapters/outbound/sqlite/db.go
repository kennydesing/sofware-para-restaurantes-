package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB encapsula el pool de conexiones a SQLite con configuración de alto rendimiento y resistencia a apagones.
type DB struct {
	*sql.DB
}

// NewDB abre o crea la base de datos SQLite y aplica los PRAGMAs indispensables para entornos offline-first:
//   - journal_mode = WAL (Write-Ahead Logging: lecturas concurrentes sin bloquear escrituras, resistente a cortes eléctricos)
//   - synchronous = NORMAL (Garantiza durabilidad en WAL sin el overhead de sincronización completa a disco en cada commit)
//   - foreign_keys = ON (Integridad referencial estricta)
//   - busy_timeout = 5000 (Manejo inteligente de reintentos por contención de locks hasta 5 segundos)
//   - temp_store = MEMORY & cache_size = -32000 (Consumo estricto de memoria < 50MB RAM)
func NewDB(dbPath string) (*DB, error) {
	// Asegurar existencia del directorio padre
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("error al crear directorio para SQLite: %w", err)
		}
	}

	dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("error al abrir conexión SQLite: %w", err)
	}

	// SQLite en modo embebido de alto rendimiento trabaja óptimamente con 1 conexión activa
	// para evitar contenciones de archivo en Windows / Linux.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Validar conectividad
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error al verificar conectividad SQLite: %w", err)
	}

	// Ejecutar PRAGMAs explícitos adicionales
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA temp_store = MEMORY;",
		"PRAGMA cache_size = -32000;", // ~32MB límite de caché en memoria
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("error al ejecutar '%s': %w", pragma, err)
		}
	}

	return &DB{DB: db}, nil
}

// RunMigrations aplica una migración SQL de forma idempotente usando una tabla de versiones.
// Si la migración ya fue aplicada (según su nombre/versión), se omite silenciosamente.
// Esto permite que el servidor se reinicie contra una BD existente sin errores de ALTER TABLE duplicado.
func (db *DB) RunMigrations(ctx context.Context, migrationSQL string) error {
	return db.RunNamedMigration(ctx, "", migrationSQL)
}

// RunNamedMigration aplica una migración con nombre idempotente.
// Si name es vacío, ejecuta directamente (sin tracking de versión).
func (db *DB) RunNamedMigration(ctx context.Context, name string, migrationSQL string) error {
	if name == "" {
		_, err := db.ExecContext(ctx, migrationSQL)
		if err != nil {
			return fmt.Errorf("error al ejecutar migración SQL: %w", err)
		}
		return nil
	}

	// Crear tabla de tracking si no existe
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT    NOT NULL PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("error al crear tabla schema_migrations: %w", err)
	}

	// Verificar si la migración ya fue aplicada
	var count int
	row := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name)
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("error al consultar schema_migrations: %w", err)
	}
	if count > 0 {
		// Ya aplicada, saltar silenciosamente
		return nil
	}

	// Aplicar la migración en una transacción
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción de migración: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migrationSQL); err != nil {
		return fmt.Errorf("error al ejecutar migración '%s': %w", name, err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, name); err != nil {
		return fmt.Errorf("error al registrar migración '%s': %w", name, err)
	}

	return tx.Commit()
}