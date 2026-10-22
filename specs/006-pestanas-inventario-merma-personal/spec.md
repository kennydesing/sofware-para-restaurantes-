# Especificación 006 — Rediseño del Módulo de Inventario con Pestañas Superiores (Inventario Principal, Merma y Uso Personal) e Inserción en Línea (Inline Editing)

**Versión:** 1.0  
**Fecha:** 2026-10-21  
**Estado:** BORRADOR — Pendiente de aprobación  
**Stack:** Go 1.22 · SQLite WAL (`github.com/mattn/go-sqlite3`) · HTML5 / CSS3 / Vanilla JS · Arquitectura Hexagonal  
**Specs vinculadas:** 001 (Tres interfaces POS), 002 (RBAC multi-rol), 003 (Unidades de medida), 004 (Tabla insumos v2 + Alertas + Tasa BCV), 005 (Empaques y presentaciones)

---

## 1. Contexto del Negocio y Problemática Operativa

En la gestión diaria de un restaurante de alta rotación, el inventario de almacén no solo se reduce mediante las ventas registradas por comanda en el POS (escandallo de platos), sino a través de dos canales operativos críticos que requieren registro inmediato, sin fricción y sin desviar la atención del personal:

1. **Pérdidas operativas y descarte (Mermas):** Descomposición de perecederos, roturas de envases, derrames, errores de cocción o caducidad.
2. **Alimentación del personal (Uso Personal):** Ingredientes retirados del almacén general para elaborar la comida diaria del equipo de cocina, sala y administración.

### Limitaciones del Sistema Actual
- El módulo de inventario actual mezcla la vista de catálogo general con datos agregados de merma, sin ofrecer un espacio directo para auditar o registrar mermas históricas de forma ágil.
- No existía un canal diferenciado para imputar el costo y la salida de insumos destinados a la comida del personal ("Uso Personal"), distorsionando la merma real o forzando ajustes manuales opacos.
- Si bien la especificación previa eliminó el modal de creación de insumos en favor de la inserción en línea, este mecanismo no está unificado ni extendido hacia las operaciones de pérdida y autoconsumo.

### Objetivo de la Especificación
Rediseñar el módulo de inventario del panel de administración (`/admin/`) introduciendo una barra de **sub-pestañas superiores** dividida en:
1. **Inventario Principal:** Catálogo maestro de 8 columnas definitivas con edición e inserción en línea.
2. **Mermas:** Cuadro simplificado de 2 columnas para el registro directo y trazable de pérdidas.
3. **Uso Personal:** Cuadro estructurado de 2 columnas para la imputación y descuento automático de insumos para la alimentación del equipo.

Todo el módulo operará **100% bajo el paradigma de inserción directa en tabla (Inline Editing)**, eliminando ventanas modales flotantes y garantizando débitos atómicos sobre SQLite con modo WAL.

---

## 2. Requisitos Funcionales (Formato EARS)

### RF-01 — Navegación Fluida entre Sub-pestañas Superiores de Inventario

- **EARS (Event-driven):** *When* el usuario selecciona cualquiera de las sub-pestañas superiores del módulo de Inventario (**`Inventario Principal`**, **`Mermas`**, **`Uso Personal`**), *the system shall* alternar el panel visible en menos de 50 ms mediante manipulación directa de clases DOM (`hidden` / `active`), sin recargar la página ni perder el estado de la sesión.
- **EARS (State-driven):** *While* el usuario navega entre diferentes pestañas del sistema POS y regresa a Inventario, *the system shall* restaurar la última sub-pestaña activa utilizando persistencia en `sessionStorage` (`pos_subtab_inventario_activa`).
- **EARS (Unwanted-behavior):** *If* el usuario intenta cambiar de sub-pestaña mientras tiene una fila con datos sin guardar en modo edición o nueva inserción, *then the system shall* solicitar confirmación rápida mediante un diálogo ligero no bloqueante antes de descartar los cambios no persistidos.

#### Criterios de Aceptación:
1. Las pestañas se renderizan como botones segmentados en la parte superior de la sección de inventario con feedback visual inmediato (indicador activo con color de acento, tipografía bold y contraste WCAG AAA).
2. El cambio de pestaña no realiza peticiones HTTP redundantes si los datos ya fueron precargados en memoria; utiliza actualización optimista en segundo plano.
3. Accesibilidad: Las pestañas son navegables mediante teclado utilizando las teclas de flecha `[←]` / `[→]` y seleccionables con `[Enter]` o `[Espacio]`.

---

### RF-02 — Pestaña 1: Catálogo e Inventario Principal (8 Columnas Definitivas)

- **EARS (Ubiquitous):** *The system shall* renderizar la tabla del Inventario Principal con exactamente las siguientes 8 columnas estructuradas:

| # | Encabezado | Campo de Dominio / Origen | Comportamiento en Modo Lectura | Comportamiento en Modo Edición Inline |
|---|------------|---------------------------|--------------------------------|---------------------------------------|
| 1 | **Producto** | `Insumo.Nombre` | Texto plano (bold). Búsqueda rápida. | `<input type="text">` requerido, auto-foco. |
| 2 | **Cantidad** | `Insumo.Stock` | Número con formato decimal (2 decimales). Badge de alerta si `Stock <= StockMinimo`. | `<input type="number" step="0.01">` positivo. |
| 3 | **Peso / Unidad** | `Insumo.UnidadBase` / Presentación | Unidad base oficial (`g`, `ml`, `u`) o etiqueta de presentación. | `<select>` con unidades del sistema (`g`, `ml`, `u`). |
| 4 | **Stock Mínimo** | `Insumo.StockMinimo` | Número formateado + unidad. | `<input type="number" step="0.01">` requerido. |
| 5 | **Costo** | `Insumo.CostoPromedio` | Doble línea: **USD** (bold) y su equivalente en **Bs** dinámico según tasa BCV. | `<input type="number" step="0.01">` USD + preview Bs automático. |
| 6 | **Precio Final** | `Insumo.PrecioVentaUSD` | Doble línea: **USD** (bold) y su equivalente en **Bs** dinámico según tasa BCV. | `<input type="number" step="0.01">` USD + preview Bs automático. |
| 7 | **Margen de Ganancias** | Calculado en cliente | Badge de color: Verde ($\ge 20\%$), Ámbar ($5\% - 19.9\%$), Rojo ($< 5\%$). Muestra % y monto neto USD/Bs. | Cálculo dinámico en tiempo real ante eventos `input` de Costo o Precio Final. |
| 8 | **Ajuste** | Acciones de fila | Botones: ✏️ *Editar fila* y 🗑️ *Eliminar producto*. | Botones: 💾 *Guardar* y ❌ *Cancelar*. |

- **EARS (Event-driven):** *When* el usuario hace clic en el botón superior **`+ Agregar Producto`**, *the system shall* insertar instantáneamente una fila vacía editable en la primera posición (`tbody.firstChild`) de la tabla de Inventario Principal, asignando el foco automático al campo *Producto*.
- **EARS (Unwanted-behavior):** *If* ya existe una fila nueva (`data-insumo-id="nuevo"`) en proceso de edición, *then the system shall* enfocar el campo *Producto* de dicha fila sin generar duplicados en el DOM.

#### Criterios de Aceptación:
1. Cero ventanas modales flotantes: la inserción y edición ocurren estrictamente dentro de la rejilla de la tabla.
2. Fórmulas de margen:
   $$\text{Margen \%} = \frac{\text{Precio Final (USD)} - \text{Costo (USD)}}{\text{Precio Final (USD)}} \times 100$$
   $$\text{Margen Monto (USD)} = \text{Precio Final (USD)} - \text{Costo (USD)}$$
3. Conversión de divisas: Al recibir eventos de cambio en la tasa BCV (Spec 004), todas las celdas de Costo, Precio Final y Margen actualizan su valor en Bolívares de inmediato.

---

### RF-03 — Pestaña 2: Mermas (Pérdidas Operativas Centralizadas)

- **EARS (Ubiquitous):** *The system shall* proveer una tabla dedicada para el registro y auditoría de pérdidas operativas con exactamente **2 columnas principales**:

| # | Columna | Campo / Control | Descripción Operativa |
|---|---------|-----------------|-----------------------|
| 1 | **Producto** | Selector de insumo del catálogo | Desplegable o autocompletado con nombre de insumos activos y su stock actual disponible como referencia (`Ej: Tomate Manzano (Disp: 4.50 kg)`). |
| 2 | **Cantidad** | Input numérico con unidad base | Magnitud física descartada (número mayor a cero) y selector/indicador de la unidad de medida (`g`, `ml`, `u`). |
| - | *(Acciones)* | Botones de confirmación | Botones inline `💾 Registrar Merma` y `❌ Cancelar` durante inserción; visualización de fecha/hora y responsable en modo lectura. |

- **EARS (Event-driven):** *When* el usuario pulsa el botón superior **`+ Registrar Merma`**, *the system shall* insertar una fila editable al inicio de la tabla de mermas con el selector de productos precargado y foco en el campo de selección.
- **EARS (Event-driven):** *When* el usuario confirma el registro de una merma válida, *the system shall* persistir el movimiento con tipo `SALIDA_MERMA`, debitar el stock del insumo de forma atómica en SQLite y reflejar el descuento instantáneo en la Pestaña 1 (Inventario Principal).

#### Criterios de Aceptación:
1. La tabla muestra los registros históricos de merma ordenados descendentemente por fecha (`created_at DESC`).
2. La carga inline no bloquea la navegación ni requiere recargar la página.
3. La interfaz totaliza visualmente en un banner de métricas superiores el impacto de mermas del día/semana en unidades y valor monetario estimado (USD/Bs).

---

### RF-04 — Pestaña 3: Uso Personal (Alimentación del Personal del Restaurante)

- **EARS (Ubiquitous):** *The system shall* disponer de una tabla estructurada para la alimentación del personal compuesta por **2 columnas funcionales**:

| # | Columna | Campo / Control | Descripción Operativa |
|---|---------|-----------------|-----------------------|
| 1 | **Producto** | Selector de insumo retirado | Selección rápida del insumo de cocina retirado del almacén para el consumo del personal (`Ej: Arroz Blanco`, `Pollo Entero`, `Aceite Vegetal`). |
| 2 | **Cantidad y Peso** | Entrada numérica y unidad | Magnitud física descontada para el equipo (volumen en `ml`/`L`, peso en `g`/`kg`, o unidades `u`). Incluye validación de stock disponible. |
| - | *(Acciones)* | Botones de control | Botones inline `💾 Registrar Consumo` y `❌ Cancelar` durante inserción; indicador de fecha/hora y usuario en lectura. |

- **EARS (Event-driven):** *When* el usuario pulsa el botón superior **`+ Registrar Uso Personal`**, *the system shall* desplegar la fila editable de captura rápida al inicio de la tabla con autofocus.
- **EARS (Event-driven):** *When* se confirma el registro, *the system shall* descontar de forma automática e inmediata la cantidad especificada del inventario principal, registrando el movimiento en el libro diario de auditoría con la referencia de consumo de personal.

#### Criterios de Aceptación:
1. El consumo de personal queda perfectamente diferenciado de las mermas operativas por descomposición, permitiendo balances de costos laborales y beneficios al empleado separados de las pérdidas no controladas.
2. Cada registro de uso personal descuenta directamente el inventario físico disponible en el almacén principal.

---

### RF-05 — Lógica de Débito Automático, Validación y Trazabilidad Inmutable

- **EARS (Event-driven):** *When* se procesa una solicitud de salida por Merma o Uso Personal, *the system shall* ejecutar una transacción atómica en SQLite que:
  1. Verifique que la cantidad requerida sea estrictamente mayor que cero ($> 0$).
  2. Obtenga el insumo bloqueando concurrentemente la fila mediante la conexión de base de datos en modo WAL.
  3. Calcule el nuevo stock:
     $$\text{Stock}_{\text{nuevo}} = \text{Stock}_{\text{actual}} - \text{Cantidad}_{\text{descontada}}$$
  4. Inserte un registro inmutable en `movimientos_inventario` con:
     - `insumo_id`: ID del insumo.
     - `tipo`: `SALIDA_MERMA` (con prefijo o metadato en referencia para Uso Personal) o `SALIDA_USO_PERSONAL` según la estrategia de arquitectura aprobada.
     - `cantidad`: Cantidad convertida a la unidad base mínima (`g`, `ml`, `u`).
     - `costo_unitario`: Costo promedio ponderado vigente del insumo al momento del débito.
     - `referencia`: Identificador legible (`"MERMA: Pérdida operativa"` o `"USO_PERSONAL: Alimentación de personal"`).
  5. Actualice la columna `stock` en la tabla `insumos`.
- **EARS (Unwanted-behavior):** *If* la cantidad solicitada para Merma o Uso Personal supera el stock actual disponible ($\text{Cantidad} > \text{Stock}_{\text{actual}}$), *then the system shall* rechazar la operación con código HTTP 400 y mensaje de error semántico (`"Stock insuficiente en almacén para procesar la salida"`), realizando ROLLBACK inmediato sin alterar ningún registro.
- **EARS (State-driven):** *While* el stock resultante de un insumo caiga por debajo de su umbral mínimo ($\text{Stock}_{\text{nuevo}} \le \text{StockMinimo}$), *the system shall* activar de forma inmediata el estado de **Alerta de Stock Crítico** (badge visual rojo, animación suave y notificación sonora si no está silenciada, según Spec 004).

#### Criterios de Aceptación:
1. Resistencia a fallos y apagones: El uso de SQLite WAL con `PRAGMA synchronous = NORMAL` garantiza durabilidad ACID.
2. Todo movimiento genera una fila de auditoría obligatoria; prohibida la mutación directa de `stock` sin su respectivo `movimientos_inventario`.
3. Sincronización multi-pestaña: Al registrar una merma o uso personal, si el usuario conmuta a la Pestaña 1 (Inventario Principal), el stock, el badge de nivel y el estado de alerta del producto reflejan el valor ya descontado sin necesidad de recargar la página (`F5`).

---

### RF-06 — Comportamiento y UX de Inserción y Edición en Línea (Inline Editing)

- **EARS (Event-driven):** *When* una fila entra en modo de edición o inserción, *the system shall* transformar las celdas en controles de formulario optimizados para entrada rápida por teclado.
- **EARS (State-driven):** *While* el usuario interactúa dentro de una fila en modo edición:
  - Presionar la tecla `[Enter]` en cualquier campo de texto o numérico debe enviar el formulario (equivalente a pulsar `💾 Guardar`).
  - Presionar la tecla `[Escape]` debe cancelar la edición (equivalente a pulsar `❌ Cancelar`), restaurando los valores originales en filas existentes o removiendo la fila temporal en inserciones nuevas.
  - La tecla `[Tab]` y `[Shift+Tab]` deben navegar secuencialmente a través de los inputs de la fila sin salirse de la tabla de forma imprevista.
- **EARS (Unwanted-behavior):** *If* los campos obligatorios (*Producto*, *Cantidad* o *Stock Mínimo*) contienen valores vacíos, nulos o formatos no numéricos al momento de presionar Guardar, *then the system shall* resaltar el campo infractor con un borde rojo (`border-red-500`) y tooltip de validación, impidiendo la emisión de la petición HTTP.

#### Criterios de Aceptación:
1. Feedback táctil y visual: Durante el guardado (`async fetch`), los botones muestran un micro-spinner o texto `"Guardando..."` y se deshabilitan para evitar doble sumisión (debounce).
2. Toast de notificación: Éxito genera un toast flotante verde en la esquina superior derecha (`"Producto guardado con éxito"` / `"Merma registrada correctamente"`); fallo genera toast rojo con el motivo devuelto por el backend.
3. Máxima ergonomía en caja y cocina: Todos los campos numéricos admiten coma o punto decimal como separador, convirtiéndolo transparentemente en float estándar antes del envío.

---

## 3. Matriz de Trazabilidad de Requisitos

| Requisito | Componente Frontend | Endpoint / Servicio Backend | Tabla SQLite Afectada |
|---|---|---|---|
| **RF-01** (Sub-pestañas superiores) | `admin.html` (subnav), `admin.js` (`cambiarSubtabInventario`) | N/A (Estado de cliente + SessionStorage) | N/A |
| **RF-02** (Inventario Principal 8 cols) | `#tabla-insumos-v2`, `renderizarFilaEditableHTML` | `GET/POST/PUT /admin/api/insumos` | `insumos` |
| **RF-03** (Mermas 2 cols) | `#tabla-mermas`, `agregarFilaNuevaMerma` | `POST /admin/api/mermas`, `GET /admin/api/mermas` | `insumos`, `movimientos_inventario` |
| **RF-04** (Uso Personal 2 cols) | `#tabla-uso-personal`, `agregarFilaNuevaUsoPersonal` | `POST /admin/api/uso-personal`, `GET /admin/api/uso-personal` | `insumos`, `movimientos_inventario` |
| **RF-05** (Débito y Trazabilidad) | Feedback reactivo en UI + Toasts | `InventarioService.RegistrarMerma`, `RegistrarUsoPersonal` | `insumos`, `movimientos_inventario` |
| **RF-06** (Inline Editing & Atajos) | Event listeners `keydown` (`Enter`/`Esc`), inputs DOM | Validación cliente + endpoints REST | N/A |

---

## 4. Casos de Borde y Manejo de Errores

1. **Intento de registrar merma con cantidad cero o negativa:** Rechazo en cliente con mensaje de alerta y rechazo en backend mediante `domain.ErrCantidadInvalida`.
2. **Registro de merma/uso personal que supera el stock físico disponible:** Backend retorna error `ErrStockInsuficiente`. Frontend muestra notificación de advertencia en rojo y no altera la fila hasta que el usuario corrija la cantidad.
3. **Pérdida de conectividad o reinicio del servidor durante el guardado inline:** Frontend detecta fallo en `fetch()` y mantiene la fila en modo edición con los datos ingresados intactos para que el usuario no pierda el trabajo, ofreciendo botón de reintento.
4. **Desconexión con la API del BCV:** Las columnas en Bolívares muestran el equivalente utilizando la última tasa persistida en SQLite; si la tasa es cero o indefinida, muestra `Bs --` sin bloquear la interacción ni los cálculos en USD.
5. **Edición concurrente de un mismo insumo:** Se valida en backend la integridad de la versión o timestamp; la transacción atómica SQLite con `busy_timeout = 5000` garantiza orden serializable sin colisiones de base de datos.
