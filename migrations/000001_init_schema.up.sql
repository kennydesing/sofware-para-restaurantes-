-- ==============================================================================
-- Migration: 000001_init_schema.up.sql
-- Sistema de Inventario Interno y Punto de Venta (POS) para Restaurantes
-- Arquitectura Hexagonal - Persistencia SQLite de Alto Rendimiento (WAL + NORMAL)
-- Offline-First, Resistente a Cortes Eléctricos, Control Estricto por Escandallo
-- ==============================================================================

-- ==============================================================================
-- 1. TABLA: INSUMOS (Materias primas en su unidad base mínima: g, ml, u)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS insumos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT NOT NULL,
    unidad_base TEXT NOT NULL CHECK (unidad_base IN ('g', 'ml', 'u')),
    costo_promedio REAL NOT NULL DEFAULT 0.0 CHECK (costo_promedio >= 0),
    stock_minimo REAL NOT NULL DEFAULT 0.0 CHECK (stock_minimo >= 0),
    stock REAL NOT NULL DEFAULT 0.0,
    precio_venta_usd REAL NOT NULL DEFAULT 0.0 CHECK (precio_venta_usd >= 0),
    presentacion_nombre TEXT NOT NULL DEFAULT '',
    cantidad_base_por_presentacion REAL NOT NULL DEFAULT 1.0 CHECK (cantidad_base_por_presentacion >= 0),
    empaque_tipo TEXT NOT NULL DEFAULT 'UNITARIO' CHECK (empaque_tipo IN ('UNITARIO', 'GAVERA_36', 'CAJA_24_BOT', 'CAJA_24_LATA', 'SIX_PACK_6', 'CAJA_12_ART', 'PERSONALIZADO')),
    empaque_nombre TEXT NOT NULL DEFAULT '',
    unidades_por_empaque INTEGER NOT NULL DEFAULT 1 CHECK (unidades_por_empaque >= 1),
    contenido_ml_por_unidad REAL NOT NULL DEFAULT 0.0 CHECK (contenido_ml_por_unidad >= 0),
    es_retornable INTEGER NOT NULL DEFAULT 0 CHECK (es_retornable IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_insumos_nombre UNIQUE (nombre)
);

CREATE INDEX IF NOT EXISTS idx_insumos_unidad ON insumos(unidad_base);
CREATE INDEX IF NOT EXISTS idx_insumos_stock ON insumos(stock);

-- ==============================================================================
-- 2. TABLA: PLATOS (Menú de productos para venta en POS / Mesa QR)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS platos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT NOT NULL,
    precio_venta REAL NOT NULL DEFAULT 0.0 CHECK (precio_venta >= 0),
    activo INTEGER NOT NULL DEFAULT 1 CHECK (activo IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_platos_nombre UNIQUE (nombre)
);

CREATE INDEX IF NOT EXISTS idx_platos_activo ON platos(activo);

-- ==============================================================================
-- 3. TABLA: RECETAS_ESCANDALLO (Ficha técnica: relación plato -> ingredientes)
-- Define la cantidad requerida en g/ml/u y el porcentaje de merma esperado.
-- Nota: unidad_receta_id se añade en 000004_unidades_medida.up.sql para mantener
--       el orden de migraciones histórico; aquí se crea sin esa columna.
-- ==============================================================================
CREATE TABLE IF NOT EXISTS recetas_escandallo (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plato_id INTEGER NOT NULL,
    insumo_id INTEGER NOT NULL,
    cantidad_requerida REAL NOT NULL CHECK (cantidad_requerida > 0),
    porcentaje_merma REAL NOT NULL DEFAULT 0.0 CHECK (porcentaje_merma >= 0),
    unidad_receta_id INTEGER DEFAULT NULL REFERENCES unidades_medida(id) ON DELETE RESTRICT,
    FOREIGN KEY (plato_id) REFERENCES platos(id) ON DELETE CASCADE,
    FOREIGN KEY (insumo_id) REFERENCES insumos(id) ON DELETE RESTRICT,
    CONSTRAINT uq_receta_plato_insumo UNIQUE (plato_id, insumo_id)
);

CREATE INDEX IF NOT EXISTS idx_recetas_plato ON recetas_escandallo(plato_id);
CREATE INDEX IF NOT EXISTS idx_recetas_insumo ON recetas_escandallo(insumo_id);

-- ==============================================================================
-- 4. TABLA: MESAS (Control de salones y ubicaciones)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS mesas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    numero INTEGER NOT NULL,
    capacidad INTEGER NOT NULL DEFAULT 4 CHECK (capacidad > 0),
    activa INTEGER NOT NULL DEFAULT 1 CHECK (activa IN (0, 1)),
    CONSTRAINT uq_mesas_numero UNIQUE (numero)
);

-- ==============================================================================
-- 5. TABLA: PEDIDOS (Comandas y órdenes de venta)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS pedidos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mesa_id INTEGER NOT NULL,
    estado TEXT NOT NULL DEFAULT 'PENDIENTE' CHECK (estado IN ('PENDIENTE', 'EN_COCINA', 'LISTO', 'ENTREGADO', 'CANCELADO')),
    total REAL NOT NULL DEFAULT 0.0 CHECK (total >= 0),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (mesa_id) REFERENCES mesas(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_pedidos_mesa ON pedidos(mesa_id);
CREATE INDEX IF NOT EXISTS idx_pedidos_estado ON pedidos(estado);
CREATE INDEX IF NOT EXISTS idx_pedidos_created ON pedidos(created_at);

-- ==============================================================================
-- 6. TABLA: DETALLES_PEDIDO (Ítems y cantidades ordenadas por cada pedido)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS detalles_pedido (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pedido_id INTEGER NOT NULL,
    plato_id INTEGER NOT NULL,
    cantidad INTEGER NOT NULL CHECK (cantidad > 0),
    precio_unitario REAL NOT NULL DEFAULT 0.0 CHECK (precio_unitario >= 0),
    subtotal REAL NOT NULL DEFAULT 0.0 CHECK (subtotal >= 0),
    FOREIGN KEY (pedido_id) REFERENCES pedidos(id) ON DELETE CASCADE,
    FOREIGN KEY (plato_id) REFERENCES platos(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_detalles_pedido_pedido ON detalles_pedido(pedido_id);
CREATE INDEX IF NOT EXISTS idx_detalles_pedido_plato ON detalles_pedido(plato_id);

-- ==============================================================================
-- 7. TABLA: MOVIMIENTOS_INVENTARIO (Libro diario para auditoría total)
-- Audita cada entrada o salida: compras, ventas (escandallo), mermas o ajustes.
-- ==============================================================================
CREATE TABLE IF NOT EXISTS movimientos_inventario (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    insumo_id INTEGER NOT NULL,
    tipo TEXT NOT NULL CHECK (tipo IN ('ENTRADA_COMPRA', 'SALIDA_VENTA', 'SALIDA_MERMA', 'AJUSTE_MANUAL')),
    cantidad REAL NOT NULL,
    costo_unitario REAL NOT NULL DEFAULT 0.0 CHECK (costo_unitario >= 0),
    referencia TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (insumo_id) REFERENCES insumos(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_movimientos_insumo ON movimientos_inventario(insumo_id);
CREATE INDEX IF NOT EXISTS idx_movimientos_tipo ON movimientos_inventario(tipo);
CREATE INDEX IF NOT EXISTS idx_movimientos_created ON movimientos_inventario(created_at);