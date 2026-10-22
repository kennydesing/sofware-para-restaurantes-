# Especificación Técnica — 003: Sistema Integral de Unidades de Medida, Equivalencias y Conversiones

**Versión:** 1.0  
**Estado:** Draft  
**Fecha:** 2026-10-14  
**Autor:** Equipo de Ingeniería  
**Base:** AGENTS.md — POS Core & Inventario por Escandallo para Restaurantes

---

## 1. Introducción

### 1.1 Propósito

Esta especificación define el sistema de gestión de unidades de medida para el POS e Inventario de restaurantes. Resuelve la dualidad entre **unidades de compra** (cajas, bultos, kilogramos, litros) y **unidades de receta/consumo en cocina** (gramos, mililitros, piezas), garantizando conversiones precisas sin errores de punto flotante que generen mermas fantasma por redondeo.

### 1.2 Alcance

Incluye:
- Registro y clasificación de unidades de medida (Volumen, Peso/Masa, Conteo/Empaque, Culinarias).
- Factores de conversión categóricos y personalizados por insumo.
- Asociación de múltiples unidades de compra a un mismo insumo.
- Motor de conversión automática en escandallos y reducción de stock.
- Cálculo de costos unitarios basados en conversión de empaque a porción.

No incluye:
- Cambios en la lógica de dominio existente fuera de las entidades nuevas (`UnidadMedida`, `FactorConversion`).
- Modificaciones a la tabla `insumos` que rompan compatibilidad (solo adición de campos opcionales).
- UI/UX — solo backend hexagonal.

### 1.3 Definiciones

| Término | Definición |
|---------|-----------|
| **Unidad Base Mínima** | Unidad canónica de almacenamiento interno: `g` (gramos), `ml` (mililitros), `u` (unidades). Todas las conversiones terminan aquí. |
| **Unidad de Compra** | Unidad en la que se adquiere el insumo al proveedor (ej: `caja`, `bulto`, `kg`, `L`, `docena`). |
| **Unidad de Receta** | Unidad en la que el chef expresa la ficha técnica (ej: `g`, `ml`, `pza`, `tbsp`, `tsp`, `porcion`). |
| **Factor de Conversión** | Multiplicador exacto (racional/entero) para transformar de una unidad a otra dentro de la misma categoría. |
| **Equivalencia de Compra** | Regla que define cuántas unidades base mínimas contiene una unidad de compra (ej: 1 `caja` = 10 `kg` = 10,000 `g`). |
| **Precisión Fija** | Representación numérica que evita errores de punto flotante (ej: enteros en micro-unidades o `decimal.Decimal`). |

---

## 2. Clasificación de Unidades (Catálogo Base)

El sistema soporta 4 categorías con sus unidades base mínimas obligatorias:

### 2.1 Volumen (Base: `ml`)
| Unidad | Abreviatura | Factor a Base (`ml`) | Tipo |
|--------|-------------|---------------------|------|
| Mililitro | `ml` | 1 | Base |
| Litro | `l` | 1000 | Métrica |
| Centilitro | `cl` | 10 | Métrica |
| Galón (US) | `gal` | 3785.411784 | Imperial |
| Onza líquida (US) | `fl_oz` | 29.5735296 | Imperial |
| Taza (US) | `cup` | 236.588237 | Culinaria |
| Cucharada | `tbsp` | 14.7867648 | Culinaria |
| Cucharadita | `tsp` | 4.9289216 | Culinaria |

### 2.2 Peso / Masa (Base: `g`)
| Unidad | Abreviatura | Factor a Base (`g`) | Tipo |
|--------|-------------|---------------------|------|
| Gramo | `g` | 1 | Base |
| Kilogramo | `kg` | 1000 | Métrica |
| Miligramo | `mg` | 0.001 | Métrica |
| Libra | `lb` | 453.59237 | Imperial |
| Onza | `oz` | 28.3495231 | Imperial |

### 2.3 Conteo / Empaque (Base: `u`)
| Unidad | Abreviatura | Factor a Base (`u`) | Tipo |
|--------|-------------|---------------------|------|
| Unidad / Pieza | `pza` / `u` | 1 | Base |
| Docena | `dz` | 12 | Empaque |
| Caja (genérica) | `caja` | Variable* | Empaque |
| Bulto / Saco | `bulto` | Variable* | Empaque |
| Atado / Manojo | `atado` | Variable* | Empaque |

> *Las unidades de empaque genéricas (`caja`, `bulto`, `atado`) requieren factor **personalizado por insumo** (ej: "Caja de tomate = 10 kg", "Atado de cilantro = 50 g").

### 2.4 Culinarias / Porciones (Base: según categoría del insumo)
| Unidad | Abreviatura | Equivalencia | Aplica a |
|--------|-------------|--------------|----------|
| Porción / Ración | `porcion` | Definida por receta | Todos |
| Cucharada | `tbsp` | 15 ml / ~12 g | Volumen / Peso |
| Cucharadita | `tsp` | 5 ml / ~4 g | Volumen / Peso |
| Taza | `cup` | 240 ml / ~120 g | Volumen / Peso |

---

## 3. Requisitos Funcionales (Formato EARS)

### 3.1 RF-01 — Catálogo de Unidades de Medida

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-01.01 | Universal | El [sistema] shall mantener una tabla `unidades_medida` con: `id`, `nombre`, `abreviatura`, `categoria` (VOLUMEN, PESO, CONTEO, CULINARIA), `es_base` (boolean), `factor_a_base` (racional: numerador/denominador). |
| RF-01.02 | Universal | El [sistema] shall precargar las unidades estándar de la Sección 2 al ejecutar la migración 000003. |
| RF-01.03 | Universal | El [sistema] shall permitir al administrador crear unidades personalizadas (ej: "Caja de 10kg", "Bulto de 25kg") con factor a base configurado. |
| RF-01.04 | Universal | El [sistema] shall validar que `factor_a_base` > 0 y que `categoria` sea uno de los 4 permitidos. |
| RF-01.05 | Universal | El [sistema] shall garantizar que exista exactamente 1 unidad `es_base = true` por categoría. |

### 3.2 RF-02 — Factores de Conversión entre Unidades

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-02.01 | Universal | El [sistema] shall mantener una tabla `factores_conversion` con: `id`, `unidad_origen_id`, `unidad_destino_id`, `factor_numerador` (INTEGER), `factor_denominador` (INTEGER). |
| RF-02.02 | Universal | El [sistema] shall almacenar factores como fracciones exactas (ej: 1 lb = 45359237/100000 g) para evitar errores de punto flotante. |
| RF-02.03 | Universal | El [sistema] shall permitir conversiones directas entre cualquier par de unidades de la **misma categoría**. |
| RF-02.04 | Universal | El [sistema] shall calcular conversiones indirectas (A→B→C) componiendo factores si no existe conversión directa A→C. |
| RF-02.05 | Unwanted-behavior | Si [se solicita conversión entre categorías incompatibles] (ej: `kg` → `ml`), el [sistema] shall retornar error `ErrCategoriaIncompatible`. |

### 3.3 RF-03 — Equivalencias de Compra por Insumo

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-03.01 | Universal | El [sistema] shall extender la tabla `insumos` con campos opcionales: `unidad_compra_id` (FK a `unidades_medida`), `factor_compra_a_base_numerador`, `factor_compra_a_base_denominador`. |
| RF-03.02 | Universal | El [sistema] shall permitir que un insumo tenga **múltiples** unidades de compra registradas (ej: se compra por "Caja de 10kg" Y por "Bulto de 25kg"). |
| RF-03.03 | Universal | El [sistema] shall crear tabla `insumo_unidades_compra` con: `id`, `insumo_id`, `unidad_medida_id`, `factor_numerador`, `factor_denominador`, `es_principal` (boolean). |
| RF-03.04 | State-driven | Cuando [el admin registra una compra], el [sistema] shall convertir la cantidad comprada a unidad base usando la equivalencia seleccionada. |
| RF-03.05 | Universal | El [sistema] shall calcular el costo unitario en base mínima: `costo_unitario_base = costo_total_compra / (cantidad_comprada * factor_compra_a_base)`. |

### 3.4 RF-04 — Unidad de Receta y Conversión en Escandallo

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-04.01 | Universal | El [sistema] shall extender `recetas_escandallo` con `unidad_receta_id` (FK a `unidades_medida`) y campos de factor a base para la unidad de receta. |
| RF-04.02 | State-driven | Cuando [el chef define la ficha técnica], el [sistema] shall permitir expresar `cantidad_requerida` en la unidad de receta elegida (ej: `2 tbsp`, `1 porcion`, `50 g`). |
| RF-04.03 | Universal | El [sistema] shall convertir automáticamente `cantidad_requerida` (en unidad de receta) a unidad base del insumo al calcular consumo y costos. |
| RF-04.04 | Universal | El [sistema] shall validar que la unidad de receta sea compatible con la categoría de la unidad base del insumo. |

### 3.5 RF-05 — Motor de Conversión y Cálculo de Costos

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-05.01 | Universal | El [sistema] shall proveer un servicio `ConversionService` con método `Convertir(cantidad, unidadOrigen, unidadDestino) (float64, error)` usando aritmética de enteros (numerador/denominador) internamente. |
| RF-05.02 | Universal | El [sistema] shall proveer `CalcularCostoUnitarioBase(costoTotalCompra, cantidadComprada, unidadCompra) float64` que retorna costo por unidad base mínima con precisión de 4 decimales. |
| RF-05.03 | Universal | El [sistema] shall proveer `CalcularConsumoReceta(cantidadReceta, unidadReceta, insumo) float64` que retorna cantidad en unidad base del insumo (incluyendo merma). |
| RF-05.04 | Universal | El [sistema] shall redondear resultados finales a **4 decimales** para `g`/`ml` y **2 decimales** para `u` al persistir en DB. |

### 3.6 RF-06 — Integración con Flujo Existente

| ID | Tipo | Requisito |
|----|------|-----------|
| RF-06.01 | State-driven | Cuando [se procesa un pedido], el [AtomicDebitor] shall usar las conversiones definidas para debitar stock en unidad base correcta. |
| RF-06.02 | Universal | El [sistema] shall registrar en `movimientos_inventario` la cantidad en unidad base y referenciar la unidad de compra/receta usada. |
| RF-06.03 | Universal | El [sistema] shall mantener compatibilidad total: insumos sin `unidad_compra_id` usan conversión hardcoded actual (`ConvertirAUnidadBase`). |
| RF-06.04 | Universal | El [admin dashboard] shall mostrar stock en unidad base Y en unidad de compra principal para referencia visual. |

---

## 4. Modelo de Datos Lógico

```
┌──────────────────┐       ┌──────────────────────────┐
│  unidades_medida │       │  factores_conversion     │
├──────────────────┤       ├──────────────────────────┤
│ id (PK)          │       │ id (PK)                  │
│ nombre           │       │ unidad_origen_id (FK)    │
│ abreviatura      │◄──────│ unidad_destino_id (FK)   │
│ categoria        │       │ factor_numerador (INT)   │
│ es_base          │       │ factor_denominador (INT) │
│ factor_a_base_n  │       └──────────────────────────┘
│ factor_a_base_d  │                ▲
└──────────────────┘                │
        ▲                           │
        │                           │
        │                    ┌──────┴──────┐
        │                    │             │
┌───────┴───────┐    ┌───────┴───────┐ ┌───┴────┐
│   insumos     │    │ insumo_unidades│ │recetas │
├───────────────┤    │ _compra        │ │_escandal│
│ unidad_compra_│    ├────────────────┤ │lo      │
│ _id (FK, opt) │    │ insumo_id (FK) │ ├────────┤
│ ...           │    │ unidad_medida_ │ │unidad_ │
└───────────────┘    │ _id (FK)       │ │receta_ │
                     │ factor_numer.  │ │id (FK) │
                     │ factor_denom.  │ │factor_ │
                     │ es_principal   │ │numer.  │
                     └────────────────┘ │factor_ │
                                        │denom.  │
                                        └────────┘
```

---

## 5. Reglas de Negocio Críticas

### 5.1 Precisión Numérica (Anti-Redondeo)
- **Regla de Oro:** Todas las conversiones internas usan `INT64` para numeradores/denominadores.
- Fórmula: `resultado = (cantidad * numerador) / denominador` — división entera al final.
- Solo al presentar al usuario se convierte a `float64` con 4 decimales.

### 5.2 Compatibilidad Hacia Atrás
- Insumos existentes sin `unidad_compra_id` siguen usando `domain.ConvertirAUnidadBase()` (hardcoded).
- Migración es **aditiva**: no elimina ni modifica columnas existentes, solo añade.

### 5.3 Validación de Categoría
- Solo se puede convertir entre unidades de la **misma categoría** (Volumen↔Volumen, Peso↔Peso, Conteo↔Conteo).
- Excepción: Unidades culinarias (`tbsp`, `tsp`, `cup`) pueden usarse en Volumen o Peso según densidad del insumo (requiere campo `densidad_g_ml` opcional en insumo — **Fase 2**).

---

## 6. Consideraciones de Seguridad

| Requisito | Detalle |
|-----------|---------|
| SC-01 | Validar que `factor_denominador != 0` en todas las inserciones/actualizaciones. |
| SC-02 | Prevenir división por cero en `ConversionService`. |
| SC-03 | Sanitizar inputs de admin para nombres de unidades (evitar inyección SQL via prepared statements). |
| SC-04 | Auditar cambios a factores de conversión (tabla `movimientos_inventario` o `auditoria`). |

---

## 7. Consideraciones de Rendimiento

| Requisito | Detalle |
|-----------|---------|
| PF-01 | Cache en memoria de `unidades_medida` y `factores_conversion` al arranque (tablas pequeñas, <200 filas). |
| PF-02 | Consultas de conversión resueltas en O(1) via mapa `map[par_unidades]factor`. |
| PF-03 | Índices compuestos en `factores_conversion(unidad_origen_id, unidad_destino_id)`. |

---