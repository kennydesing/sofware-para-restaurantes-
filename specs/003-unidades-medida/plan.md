# Plan de Arquitectura — 003: Sistema Integral de Unidades de Medida

**Versión:** 1.0  
**Estado:** Aprobado para implementación  
**Fecha:** 2026-10-14  
**Depende de:** `spec.md` (003-unidades-medida), Migraciones 000001–000003

---

## 1. Decisión de Arquitectura: Fracciones Exactas (INT64)

### Problema
`float64` acumula errores de redondeo en conversiones encadenadas. Ejemplo real:
```
1 gal × 3785.411784 ml/gal × (1/1000) l/ml = 3.785411784 l
// En float64: puede resultar 3.7854117839999997 l → merma fantasma
```

### Solución Adoptada
Almacenar todos los factores como fracciones racionales `(numerador INT64, denominador INT64)`:
```
1 gal = 3785411784 / 1000000 ml  (exacto)
1 lb  = 45359237  / 100000  g    (exacto, estándar NIST)
```
La división entera solo ocurre al final, en la presentación al usuario.

---

## 2. Diseño de Base de Datos

### 2.1 Nueva Migración: `000004_unidades_medida.up.sql`

> **Nota:** Se numera `000004` porque `000003` fue ocupada por tasas BCV.

```sql
-- ==============================================================================
-- Migration: 000004_unidades_medida.up.sql
-- ==============================================================================

-- 1. CATÁLOGO DE UNIDADES DE MEDIDA
CREATE TABLE IF NOT EXISTS unidades_medida (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre      TEXT    NOT NULL,
    abreviatura TEXT    NOT NULL UNIQUE,
    categoria   TEXT    NOT NULL CHECK (categoria IN ('VOLUMEN','PESO','CONTEO','CULINARIA')),
    es_base     INTEGER NOT NULL DEFAULT 0 CHECK (es_base IN (0, 1)),
    factor_n    INTEGER NOT NULL DEFAULT 1 CHECK (factor_n > 0),
    factor_d    INTEGER NOT NULL DEFAULT 1 CHECK (factor_d > 0),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unidades_abrev    ON unidades_medida(abreviatura);
CREATE INDEX        IF NOT EXISTS idx_unidades_cat      ON unidades_medida(categoria);
CREATE UNIQUE INDEX IF NOT EXISTS idx_unidades_base_cat ON unidades_medida(categoria) WHERE es_base = 1;

-- 2. FACTORES DE CONVERSIÓN EXPLÍCITOS
CREATE TABLE IF NOT EXISTS factores_conversion (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    unidad_origen_id  INTEGER NOT NULL REFERENCES unidades_medida(id) ON DELETE CASCADE,
    unidad_destino_id INTEGER NOT NULL REFERENCES unidades_medida(id) ON DELETE CASCADE,
    factor_n          INTEGER NOT NULL CHECK (factor_n > 0),
    factor_d          INTEGER NOT NULL CHECK (factor_d > 0),
    CONSTRAINT uq_conversion UNIQUE (unidad_origen_id, unidad_destino_id)
);

CREATE INDEX IF NOT EXISTS idx_conv_par ON factores_conversion(unidad_origen_id, unidad_destino_id);

-- 3. UNIDADES DE COMPRA POR INSUMO
CREATE TABLE IF NOT EXISTS insumo_unidades_compra (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    insumo_id    INTEGER NOT NULL REFERENCES insumos(id) ON DELETE CASCADE,
    unidad_id    INTEGER NOT NULL REFERENCES unidades_medida(id) ON DELETE RESTRICT,
    factor_n     INTEGER NOT NULL CHECK (factor_n > 0),
    factor_d     INTEGER NOT NULL CHECK (factor_d > 0),
    es_principal INTEGER NOT NULL DEFAULT 0 CHECK (es_principal IN (0, 1)),
    descripcion  TEXT    NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_insumo_unidad UNIQUE (insumo_id, unidad_id)
);

CREATE INDEX        IF NOT EXISTS idx_iuc_insumo    ON insumo_unidades_compra(insumo_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_iuc_principal ON insumo_unidades_compra(insumo_id) WHERE es_principal = 1;

-- 4. EXTENDER recetas_escandallo
ALTER TABLE recetas_escandallo ADD COLUMN unidad_receta_id INTEGER
    REFERENCES unidades_medida(id) ON DELETE RESTRICT;
-- NULL = usa unidad base del insumo (compatibilidad hacia atrás garantizada)

-- 5. SEED: CATÁLOGO ESTÁNDAR
INSERT OR IGNORE INTO unidades_medida (nombre, abreviatura, categoria, es_base, factor_n, factor_d) VALUES
    ('Mililitro',       'ml',     'VOLUMEN', 1, 1,          1),
    ('Litro',           'l',      'VOLUMEN', 0, 1000,       1),
    ('Galón US',        'gal',    'VOLUMEN', 0, 3785411784, 1000000),
    ('Onza líquida US', 'fl_oz',  'VOLUMEN', 0, 295735296,  10000000),
    ('Taza US',         'cup',    'VOLUMEN', 0, 236588237,  1000000),
    ('Cucharada',       'tbsp',   'VOLUMEN', 0, 147867648,  10000000),
    ('Cucharadita',     'tsp',    'VOLUMEN', 0, 492892159,  100000000),
    ('Gramo',           'g',      'PESO',    1, 1,          1),
    ('Kilogramo',       'kg',     'PESO',    0, 1000,       1),
    ('Miligramo',       'mg',     'PESO',    0, 1,          1000),
    ('Libra',           'lb',     'PESO',    0, 45359237,   100000),
    ('Onza',            'oz',     'PESO',    0, 283495231,  10000000),
    ('Unidad / Pieza',  'u',      'CONTEO',  1, 1,          1),
    ('Docena',          'dz',     'CONTEO',  0, 12,         1),
    ('Caja (genérica)', 'caja',   'CONTEO',  0, 1,          1),
    ('Bulto / Saco',    'bulto',  'CONTEO',  0, 1,          1),
    ('Atado / Manojo',  'atado',  'CONTEO',  0, 1,          1),
    ('Porción / Ración','porcion','CULINARIA',0, 1,          1);
```

### 2.2 Rollback: `000004_unidades_medida.down.sql`

```sql
DROP TABLE IF EXISTS insumo_unidades_compra;
DROP TABLE IF EXISTS factores_conversion;
DROP TABLE IF EXISTS unidades_medida;
-- ALTER TABLE DROP COLUMN requiere SQLite >= 3.35 (mattn/go-sqlite3 >= 1.14.15):
-- ALTER TABLE recetas_escandallo DROP COLUMN unidad_receta_id;
```

---

## 3. Capas Hexagonales en Go

### 3.1 Domain — `internal/core/domain/`

#### `unidad.go`

```go
package domain

type Categoria string

const (
    CategoriaVolumen   Categoria = "VOLUMEN"
    CategoriaPeso      Categoria = "PESO"
    CategoriaConteo    Categoria = "CONTEO"
    CategoriaCulinaria Categoria = "CULINARIA"
)

// UnidadMedida — entidad pura, sin dependencias de I/O.
// 1 unidad_this = (FactorN / FactorD) unidades_base_de_su_categoria
type UnidadMedida struct {
    ID          int64
    Nombre      string
    Abreviatura string
    Categoria   Categoria
    EsBase      bool
    FactorN     int64 // > 0
    FactorD     int64 // > 0, nunca 0
}

// InsumoUnidadCompra — equivalencia entre presentación de compra y unidad base.
type InsumoUnidadCompra struct {
    ID          int64
    InsumoID    int64
    UnidadID    int64
    FactorN     int64
    FactorD     int64
    EsPrincipal bool
    Descripcion string
}
```

#### `conversion.go`

```go
package domain

import (
    "errors"
    "math"
)

var (
    ErrCategoriaIncompatible = errors.New("unidades de categorías incompatibles")
    ErrDenominadorCero       = errors.New("denominador de conversión es cero")
)

// ConvertirEntreMismaCategoria — aritmética exacta de fracciones.
// Resultado: cantidad × (factorN_origen × factorD_destino) / (factorD_origen × factorN_destino)
func ConvertirEntreMismaCategoria(
    cantidadN, cantidadD int64,
    origen, destino UnidadMedida,
) (rN, rD int64, err error) {
    if origen.Categoria != destino.Categoria {
        return 0, 0, ErrCategoriaIncompatible
    }
    if origen.FactorD == 0 || destino.FactorD == 0 || cantidadD == 0 || destino.FactorN == 0 {
        return 0, 0, ErrDenominadorCero
    }
    rN = cantidadN * origen.FactorN * destino.FactorD
    rD = cantidadD * origen.FactorD * destino.FactorN
    return rN, rD, nil
}

// ConvertirAFloat64 materializa la fracción con redondeo a `decimals` lugares.
func ConvertirAFloat64(n, d int64, decimals int) float64 {
    if d == 0 {
        return 0
    }
    factor := math.Pow(10, float64(decimals))
    return math.Round(float64(n)/float64(d)*factor) / factor
}

// ConsumoBaseConMerma — cantidad base a descontar del stock.
func ConsumoBaseConMerma(
    cantidadRecetaN, cantidadRecetaD, factorN, factorD int64,
    porcentajeMerma float64, numPlatos int64,
) float64 {
    if factorD == 0 || cantidadRecetaD == 0 {
        return 0
    }
    base := float64(cantidadRecetaN*factorN) / float64(cantidadRecetaD*factorD)
    return base * (1 + porcentajeMerma/100) * float64(numPlatos)
}
```

---

### 3.2 Puertos — `internal/core/ports/`

Adiciones a `outbound_ports.go`:
```go
type UnidadMedidaRepository interface {
    ObtenerTodas(ctx context.Context) ([]domain.UnidadMedida, error)
    ObtenerPorID(ctx context.Context, id int64) (domain.UnidadMedida, error)
    ObtenerPorAbreviatura(ctx context.Context, abrev string) (domain.UnidadMedida, error)
    ObtenerPorCategoria(ctx context.Context, cat domain.Categoria) ([]domain.UnidadMedida, error)
    Crear(ctx context.Context, u domain.UnidadMedida) (int64, error)
    ObtenerUnidadesCompraInsumo(ctx context.Context, insumoID int64) ([]domain.InsumoUnidadCompra, error)
    GuardarUnidadCompraInsumo(ctx context.Context, iuc domain.InsumoUnidadCompra) error
}
```

Adiciones a `inbound_ports.go`:
```go
type ConversionService interface {
    Convertir(ctx context.Context, cantidad float64, origenAbrev, destinoAbrev string) (float64, error)
    CostoUnitarioBase(ctx context.Context, costoTotal, cantidadComprada float64, unidadCompraAbrev string, insumoID int64) (float64, error)
    ConsumoReceta(ctx context.Context, cantidadReceta float64, unidadRecetaAbrev string, insumoID int64, porcentajeMerma float64, numPlatos int) (float64, error)
}
```

---

### 3.3 Servicio — `internal/core/services/conversion_service.go`

Estrategia de cache en memoria:
```
Al arrancar (cmd/main.go):
  1. unidadRepo.ObtenerTodas(ctx)  → ~20 filas
  2. conversionSvc.CargarCache(unidades)  → map[abreviatura]UnidadMedida
  3. Resolución O(1) durante vida del proceso
  4. Nuevo unit → handler llama InvalidarCache() + recarga
```

---

### 3.4 Adaptador — `internal/adapters/outbound/sqlite/unidad_repo.go`

```
- ObtenerTodas      → SELECT * FROM unidades_medida (cache warmup al inicio)
- ObtenerPorAbrev   → indice único idx_unidades_abrev (O log n)
- GuardarCompra     → INSERT OR REPLACE + validación es_principal único via índice
```

---

## 4. Integración con Flujo Existente

### 4.1 AtomicDebitor — sin cambios de interfaz

```
PedidoService.ProcesarPedido()
  └─ Para cada RecetaItem:
       if item.UnidadRecetaID == 0 {
           // Ruta legacy: domain.CalcularConsumoTotalConMerma() — SIN CAMBIOS
       } else {
           // Ruta nueva: ConversionService.ConsumoReceta()
       }
       AtomicDebitor.DebitarInsumo(id, cantidad_base)  ← SIEMPRE IGUAL
```

### 4.2 Flujo de Compra con Unidad de Compra

```
Admin registra: 2 "Caja 10kg" Tomate @ $17.00 USD
  → ConversionService.CostoUnitarioBase()
      [2 × 10,000 g = 20,000 g]
      [costo = 17.00 / 20,000 = $0.00085 USD/g]
  → InventarioService.RegistrarCompra() — actualiza costo_promedio ponderado
  → movimientos_inventario: ENTRADA_COMPRA, cantidad=20000, ref="Caja 10kg × 2"
```

---

## 5. Riesgos y Mitigaciones

| Riesgo | Prob. | Impacto | Mitigación |
|--------|-------|---------|------------|
| Overflow INT64 en cadenas largas | Baja | Alto | Normalizar fracciones con GCD antes de cada operación |
| factor_d = 0 en DB | Mínima | Crítico | `CHECK (factor_d > 0)` + validación en dominio |
| Migración rompe insumos legacy | Baja | Alto | `ALTER TABLE` solo añade columna nullable |
| Cache desactualizada | Media | Medio | `InvalidarCache()` en handler POST + `sync.RWMutex` |
