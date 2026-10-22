# Especificación Técnica — 005: Presentaciones Predefinidas y Personalizadas de Empaques/Cajas de Cerveza y Bebidas

**Versión:** 1.0  
**Fecha:** 2026-10-20  
**Estado:** BORRADOR — Pendiente de Aprobación  
**Autor:** Arquitecto de Software Senior & Diseñador UI/UX  
**Base:** AGENTS.md — POS Core & Inventario por Escandallo para Restaurantes  
**Specs Previas:** 001 (Tres interfaces POS), 002 (RBAC multi-rol), 003 (Unidades de medida y equivalencias), 004 (Tabla de insumos v2, alertas y tasa cambiaria)

---

## 1. Introducción y Contexto de Negocio

### 1.1 Propósito

En la operación gastronómica y de bares (especialmente en el mercado cervecero y de bebidas comerciales), existe una asimetría estructural constante entre la **modalidad de adquisición y recepción logística** (mayorista por gaveras, cajas, bultos o packs) y la **unidad de venta/dispensación al comensal** (botella, lata individual, balde/tobo o jarra).

El objetivo de esta especificación es resolver de forma nativa, estandarizada y a prueba de fallos operativos el registro, costeo y control de stock de cervezas y bebidas, incorporando:
1. **Empaques predefinidos estándar del mercado** (gaveras de 36 botellas retornables, cajas de 24 botellas, bultos de 24 latas, six-packs de 6 unidades, cajas artesanales de 12 botellas).
2. **Empaques personalizados ("Custom Pack")** para configuraciones no estándar (packs de 8, cajas de 18, etc.) configurables de forma dinámica con especificación de unidades por caja y volumen unitario en mililitros ($ml$).
3. **Conversión automática y atómica de stock:** Compras registradas en empaques/cajas que incrementan automáticamente el inventario en unidades individuales base (`u`), recalculando el costo unitario promedio ponderado por botella y permitiendo que la venta en comandas POS descuente directamente botella a botella con precisión matemática absoluta ($100\%$ libre de discrepancias de inventario).

### 1.2 Alcance

- **Incluye:**
  - Catálogo de empaques predefinidos para cervezas industriales y artesanales.
  - Generador dinámico de empaques personalizados en modal de gestión de inventario.
  - Persistencia en SQLite WAL mediante migración formal `000006_empaques_cervezas_bebidas`.
  - Dominio y lógica de negocio hexagonal para conversión de empaque a unidad mínima de venta (`u`).
  - Cálculo instantáneo de costos unitarios por botella, margen de utilidad por unidad y precio sugerido.
  - Vinculación directa con fichas técnicas/escandallos de venta directa (botella) o combinada (tobos de cerveza, promociones o cócteles con cerveza).
  - Trazabilidad y auditoría inmutable en `movimientos_inventario` (tipo `ENTRADA_COMPRA` y `SALIDA_VENTA`).

- **No incluye:**
  - Gestión logística de préstamos de envases retornables a proveedores (se modela el atributo `es_retornable` como bandera informativa/operativa, dejando el módulo de cuentas por cobrar/devolver de cascos para una fase futura).
  - Cambios en el protocolo de pedidos POS fuera del débito por escandallo ya establecido.

### 1.3 Glosario Técnico y de Negocio

| Término | Definición |
|---|---|
| **Gavera de Plástico** | Empaque tradicional de plástico rígido utilizado para cerveza retornable en botellas de vidrio ($222\text{ ml}$ o $250\text{ ml}$), con capacidad fija de $36\text{ unidades}$. |
| **Caja no Retornable** | Empaque de cartón corrugado para botellas no retornables ($222\text{ ml}$, $250\text{ ml}$ o $355\text{ ml}$), con capacidad de $24\text{ unidades}$. |
| **Bulto / Caja de Latas** | Empaque termoencogible o bandeja de cartón con $24\text{ latas}$ ($250\text{ ml}$, $295\text{ ml}$, $350\text{ ml}$ o $355\text{ ml}$). |
| **Six-Pack** | Presentación comercial termoencogible o de canastilla de $6\text{ unidades}$ (latas o botellas). |
| **Caja Artesanal / Especial** | Empaque de $12\text{ unidades}$ habitual en cervezas de microcervecería o importadas ($330\text{ ml}$ o $355\text{ ml}$). |
| **Custom Pack** | Presentación arbitraria configurada por el usuario con $N$ unidades por empaque y $V\text{ ml}$ nominales por unidad. |
| **Unidad Base Mínima (`u`)** | Botella o lata individual física. Es la unidad canónica de almacenamiento en la tabla `insumos`. |
| **Costo por Botella** | $\text{Costo Unitario Base} = \frac{\text{Costo Total del Empaque}}{\text{Unidades por Empaque}}$. |
| **Margen Unitario** | $\text{Margen \%} = \frac{\text{Precio Venta Botella} - \text{Costo Unitario Base}}{\text{Precio Venta Botella}} \times 100$. |

---

## 2. Catálogo de Presentaciones Estándar del Mercado

El sistema debe proveer las siguientes opciones preconfiguradas con sus metadatos operativos:

| Identificador Tipo | Nombre Comercial | Unidades por Empaque | Formatos Volumétricos Típicos ($ml$) | Envase Retornable | Marcas / Referencias de Mercado |
|---|---|:---:|:---:|:---:|---|
| `GAVERA_36` | **Gavera tradicional de plástico (Retornable)** | **36** | $222\text{ ml}$, $250\text{ ml}$ | **Sí (1)** | Polar Pilsen (tercio), Polar Light, Polar Ice, Regional Regular/Light. |
| `CAJA_24_BOT` | **Caja de cartón no retornable (Vidrio)** | **24** | $222\text{ ml}$, $250\text{ ml}$, $355\text{ ml}$ | **No (0)** | Solera Classic / Verde ($222\text{ ml}$), Solera IPA, Regional Light descartable, Coronita ($210\text{ ml}$), Heineken ($330\text{ ml}$). |
| `CAJA_24_LATA` | **Caja / Bulto de latas** | **24** | $250\text{ ml}$, $295\text{ ml}$, $355\text{ ml}$ | **No (0)** | Polar Light lata, Pilsen lata, Regional lata, Brahma, Budweiser. |
| `SIX_PACK_6` | **Six-pack (Termoencogible / Canastilla)** | **6** | $250\text{ ml}$, $330\text{ ml}$, $355\text{ ml}$ | **No (0)** | Six-pack lata Polar, Solera, Corona, Modelo. |
| `CAJA_12_ART` | **Caja edición especial / artesanal** | **12** | $330\text{ ml}$, $355\text{ ml}$, $500\text{ ml}$ | **No (0)** | Cervecería Tovar (Helles, Dunkel), Norte del Sur, Cerveza Zulia botella especial. |
| `PERSONALIZADO` | **Personalizado / Otra presentación** | **Variable ($N$)** | **Variable ($V\text{ ml}$)** | **Configurable** | Packs de $8\text{ u}$, cajas de $18\text{ u}$, botellones de $1000\text{ ml}$, barriles o presentaciones de importación. |

---

## 3. Requisitos Funcionales (Formato EARS)

### 3.1 RF-01 — Registro y Selección de Presentaciones Predefinidas de Cervezas y Bebidas

| ID | Formato EARS | Requisito Detallado |
|---|---|---|
| **RF-01.01** | **Ubiquitous** | El [sistema] *shall* incorporar en los formularios modales de creación y edición de inventario (`#modal-gestionar-inventario` y `#modal-editar-insumo`) un selector denominado **"Presentación / Empaque de Adquisición"** que permita clasificar el insumo como ingrediente tradicional o bebida empacada. |
| **RF-01.02** | **Event-driven** | *When* el usuario selecciona una de las 5 categorías predefinidas (`GAVERA_36`, `CAJA_24_BOT`, `CAJA_24_LATA`, `SIX_PACK_6`, `CAJA_12_ART`), el [sistema] *shall* prellenar automáticamente: (a) Unidad base como `u` (unidades), (b) Unidades por empaque correspondientes ($36, 24, 24, 6, 12$), (c) Indicador de retornabilidad según catálogo, y (d) Lista de sugerencia de contenido volumétrico en $ml$. |
| **RF-01.03** | **State-driven** | *While* esté seleccionada una presentación predefinida, el [sistema] *shall* mantener ocultos y deshabilitados los campos de entrada manual libre de nombre de empaque y factor de empaque, previniendo errores tipográficos del operador. |
| **RF-01.04** | **Universal** | El [sistema] *shall* almacenar en la tabla `insumos` los campos: `empaque_tipo`, `empaque_nombre`, `unidades_por_empaque`, `contenido_ml_por_unidad` y `es_retornable`. |

---

### 3.2 RF-02 — Creación Dinámica de Empaques Personalizados ("Custom Pack")

| ID | Formato EARS | Requisito Detallado |
|---|---|---|
| **RF-02.01** | **Event-driven** | *When* el usuario selecciona la opción **"Personalizado / Otra presentación"** (`PERSONALIZADO`), el [sistema] *shall* desplegar dinámicamente un subformulario interactivo (`#custom-pack-fields`) con animación fluida sin recargar el modal. |
| **RF-02.02** | **State-driven** | *While* la opción `PERSONALIZADO` esté activa, el [sistema] *shall* exigir obligatoriamente los siguientes campos: (1) **Nombre descriptivo del empaque** (ej. "Caja de 18 unidades", "Pack Promocional 8u"), (2) **Unidades por empaque** (entero positivo $\ge 1$), (3) **Contenido nominal por unidad en ml** (número $> 0$), y (4) **Casilla de verificación de envase retornable** (Sí/No). |
| **RF-02.03** | **Unwanted-behavior** | *If* el usuario intenta enviar el formulario con `unidades_por_empaque` menor a $1$, vacío o con decimales, el [sistema] *shall* bloquear el envío, resaltar el campo en rojo y mostrar la notificación toast: `"Las unidades por empaque deben ser un número entero mayor o igual a 1"`. |
| **RF-02.04** | **Unwanted-behavior** | *If* el usuario ingresa un valor de `contenido_ml_por_unidad` $\le 0$, el [sistema] *shall* rechazar la validación indicando: `"El contenido por unidad en ml debe ser mayor a 0"`. |
| **RF-02.05** | **Universal** | El [sistema] *shall* registrar la configuración personalizada tanto en las propiedades del insumo como en una regla de compra asociada en `insumo_unidades_compra` para su reutilización inmediata en entradas de almacén. |

---

### 3.3 RF-03 — Conversión Automática de Stock en Compras y Entradas

| ID | Formato EARS | Requisito Detallado |
|---|---|---|
| **RF-03.01** | **State-driven** | *When* el operador registra una compra o recepción de mercancía indicando cantidad en empaques (ej. $5\text{ Gaveras}$ de $36\text{ u}$), el [sistema] *shall* convertir la cantidad a la unidad base física según la fórmula: $$\text{Stock Incrementado (u)} = \text{Cantidad de Empaques Comprados} \times \text{Unidades por Empaque}$$ |
| **RF-03.02** | **Universal** | El [sistema] *shall* calcular el costo unitario por botella para actualizar el Costo Promedio Ponderado del insumo de acuerdo a: $$\text{Costo Unitario Base (USD/u)} = \frac{\text{Costo Total de la Compra (USD)}}{\text{Stock Incrementado (u)}}$$ |
| **RF-03.03** | **Universal** | El [sistema] *shall* garantizar que el costo unitario base se almacene con una precisión mínima de 4 cifras decimales en la base de datos para neutralizar discrepancias de redondeo al multiplicar por cientos de botellas. |
| **RF-03.04** | **Event-driven** | *When* la transacción de compra se confirma exitosamente, el [sistema] *shall* persistir un registro en `movimientos_inventario` con: `tipo_movimiento = 'ENTRADA_COMPRA'`, `cantidad = Stock Incrementado (u)`, `costo_unitario = Costo Unitario Base`, y `referencia` documentando explícitamente el empaque recibido (ej. `"Compra 5 Gavera (36 u) Factura #F-4412"`). |

---

### 3.4 RF-04 — Desglose de Precios, Doble Moneda (USD / Bs) y Márgenes en Interfaz

| ID | Formato EARS | Requisito Detallado |
|---|---|---|
| **RF-04.01** | **State-driven** | *While* el usuario interactúa con el modal de inventario registrando o editando un empaque de bebida, el [sistema] *shall* calcular y exhibir en tiempo real: (a) **Costo unitario por botella** ($\text{USD}$ y $\text{Bs}$), (b) **Precio de venta al público por botella** ($\text{USD}$ y $\text{Bs}$), (c) **Margen porcentual neto** por unidad, y (d) **Rendimiento económico proyectado por empaque completo**. |
| **RF-04.02** | **Event-driven** | *When* el usuario ingresa el costo por caja/empaque en el campo de costo inicial, el [sistema] *shall* dividir automáticamente entre las unidades del empaque para reflejar de forma sincronizada el costo individual. |
| **RF-04.03** | **Event-driven** | *When* se actualiza la tasa cambiaria oficial BCV en el sistema, el [sistema] *shall* propagar el evento reactivo `tasaActualizada` para refrescar los equivalentes en Bolívares ($\text{Bs}$) de los insumos empacados en la tabla de inventario y en las vistas de escandallo. |
| **RF-04.04** | **Universal** | El [sistema] *shall* catalogar el margen por botella con los identificadores visuales del sistema de diseño: Verde para márgenes $\ge 20\%$, Ámbar para márgenes entre $5\%$ y $19.9\%$, y Rojo para márgenes $< 5\%$. |

---

### 3.5 RF-05 — Débito Atómico en Ventas POS y Consistencia con Escandallos

| ID | Formato EARS | Requisito Detallado |
|---|---|---|
| **RF-05.01** | **Event-driven** | *When* un mesero o cajero registra una orden de comanda que contiene un plato de venta de cerveza individual (ej. "Polar Pilsen $222\text{ ml}$"), el [sistema] *shall* debitar exactamente $1\text{ u}$ del insumo base asociado mediante el servicio `AtomicDebitor`. |
| **RF-05.02** | **Event-driven** | *When* la orden comanda un plato compuesto (ej. "Tobo de 10 Polar Pilsen" o "Cubetazo 6 Soleras"), el [sistema] *shall* descontar del insumo base la cantidad multiplicada especificada en la ficha técnica escandallada (ej. $10\text{ u}$ o $6\text{ u}$) de forma estrictamente indivisible. |
| **RF-05.03** | **Unwanted-behavior** | *If* el stock disponible del insumo en botellas individuales es menor a la cantidad requerida por la orden, el [sistema] *shall* cancelar la transacción mediante un `ROLLBACK` total en SQLite, preservar el stock preexistente y emitir el error `domain.ErrStockInsuficiente`. |
| **RF-05.04** | **Universal** | El [sistema] *shall* registrar cada salida por comanda en `movimientos_inventario` como `SALIDA_VENTA`, indicando el insumo, la cantidad exacta de botellas dispensadas y el ID del pedido asociado. |

---

## 4. Matriz de Trazabilidad de Requisitos

| Requisito EARS | Componente Frontend | Componente Backend / Servicio | Persistencia / Migración |
|---|---|---|---|
| **RF-01** (Empaques Predefinidos) | `admin.html` (Select `#prod-empaque-tipo`), `admin.js` | `handlers.go`, `InsumoRequestDTO` | `migrations/000006_empaques_cervezas_bebidas.up.sql` |
| **RF-02** (Custom Pack Dinámico) | `#custom-pack-fields`, `admin.js` | `handlers.go`, `domain.Insumo` | Columnas `unidades_por_empaque`, `contenido_ml_por_unidad` |
| **RF-03** (Conversión de Stock) | Modal Compra `#form-compra`, `admin.js` | `InventarioService.RegistrarCompra()`, `conversion_service.go` | Tabla `insumos`, `movimientos_inventario` |
| **RF-04** (Costeo y Márgenes Duales) | Card Dinámica Costeo `#costeo-empaque-preview` | `handlers.go` (`InsumoResponseDTO`), `tasa_cambio` | Evento JS `tasaActualizada` |
| **RF-05** (Débito Atómico POS) | POS Comanda / Menú Platos | `PedidoService`, `AtomicDebitor` | Transacción SQLite WAL (`insumos.stock`) |

---

## 5. Criterios de Aceptación y Validación

1. **Cero Números Flotantes en Conteo:** Las botellas/latas deben contabilizarse en el stock como unidades enteras o números exactos sin dispersiones de precisión.
2. **Reversibilidad SQL:** La migración debe incluir tanto el script `.up.sql` como su correspondiente contraparte `.down.sql`.
3. **Validación Hexagonal:** Cero dependencias de drivers de base de datos o HTTP en `internal/core/domain` e `internal/core/ports`.
4. **Resistencia a Apagones:** Toda la conversión y deducción de stock en modo WAL garantiza que ante un corte súbito de energía no se generen registros huérfanos entre la compra de empaques y el inventario en botellas.
