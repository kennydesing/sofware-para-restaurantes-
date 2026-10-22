let unidadesCache = [];

// State global
let insumosCache = [];
let platosCache = [];
let tasasCache = { USD: 877.90, EUR: 982.75 };
let tasaUSDActual = tasasCache.USD; // Tasa USD actual para recálculo de Bs

// Spec 004 — Estado de alerta sonora
let alertaSonidoActiva = localStorage.getItem('pos_alerta_sonido') !== 'false';
let paginaInteractuada = false;
let alarmaDisparada = false;

// Obtener token CSRF del meta tag
function getCSRFToken() {
  const meta = document.querySelector('meta[name="csrf-token"]');
  return meta ? meta.getAttribute('content') : '';
}

// Helper para hacer fetch con CSRF token automático
async function apiFetch(url, options = {}) {
  const token = getCSRFToken();
  const headers = new Headers(options.headers || {});
  if (token) {
    headers.set('X-CSRF-Token', token);
  }
  if (!headers.has('Content-Type') && options.body) {
    headers.set('Content-Type', 'application/json');
  }
  const res = await fetch(url, { ...options, headers });
  if (res.status === 401) {
    window.location.href = '/login';
    return res;
  }
  return res;
}

// ============================================================================
// Spec 004 — Funciones auxiliares de formateo (TASK-007)
// ============================================================================

// Formatea cantidad con separador de miles y unidad base.
// Ej: formatStock(2000, 'ml') → "2,000 ml"
function formatStock(cantidad, unidad) {
  if (cantidad == null || cantidad === '') return '-';
  const n = parseFloat(cantidad);
  if (isNaN(n)) return '-';
  return n.toLocaleString('es-VE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' ' + (unidad || '');
}

// Convierte un stock en unidad base a unidad legible (kg, L, g, ml, u).
// Devuelve { valor, unidad }.
function formatearPeso(stock, unidadBase) {
  const s = parseFloat(stock) || 0;
  const u = (unidadBase || 'u').toLowerCase();
  if (u === 'g') {
    return s >= 1000 ? { valor: s / 1000, unidad: 'kg' } : { valor: s, unidad: 'g' };
  }
  if (u === 'ml') {
    return s >= 1000 ? { valor: s / 1000, unidad: 'L' } : { valor: s, unidad: 'ml' };
  }
  return { valor: s, unidad: 'u' };
}

// Calcula cuántas presentaciones físicas representa el stock actual.
// Ej: 3000 g con presentación "pieza" de 1500 g → 2 piezas.
function cantidadEnPresentacion(stock, basePorPresentacion) {
  const s = parseFloat(stock) || 0;
  const base = parseFloat(basePorPresentacion) || 1;
  return s / base;
}

// Convierte una cantidad en unidad comercial a unidad base mínima.
// Soporta: kg/g, L/ml, u.
function convertirAUnidadBase(cantidad, unidadOrigen, unidadBaseDestino) {
  const c = parseFloat(cantidad) || 0;
  const origen = (unidadOrigen || '').toLowerCase();
  const destino = (unidadBaseDestino || 'u').toLowerCase();
  if (destino === 'g') {
    if (origen === 'kg' || origen === 'kilo') return c * 1000;
    if (origen === 'g' || origen === 'gr') return c;
  }
  if (destino === 'ml') {
    if (origen === 'l' || origen === 'lt' || origen === 'litro') return c * 1000;
    if (origen === 'ml') return c;
  }
  if (destino === 'u') {
    if (origen === 'u' || origen === 'unidad') return c;
  }
  return c;
}

// Convierte una cantidad en unidad base a la unidad humana correspondiente.
// Ej: 1500 g → { valor: 1.5, unidad: 'kg' }
function convertirAUnidadHumana(cantidad, unidadBase) {
  return formatearPeso(cantidad, unidadBase);
}

// Formatea una cantidad de presentaciones con su nombre (pluralización simple).
function formatCantidadPresentacion(cantidad, nombrePresentacion) {
  const n = parseFloat(cantidad) || 0;
  const nombre = (nombrePresentacion || 'unidad').trim();
  // Pluralización básica para español
  const plural = nombre.endsWith('s') ? nombre : (nombre + 's');
  const etiqueta = n === 1 ? nombre : plural;
  return n.toLocaleString('es-VE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' ' + etiqueta;
}

// Formatea monto en Bs con separador de miles y 2 decimales.
// Ej: formatBs(46500) → "46,500.00"
function formatBs(monto) {
  if (monto == null || isNaN(monto)) return '-';
  return parseFloat(monto).toLocaleString('es-VE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

// Formatea monto en USD con 2 decimales.
// Ej: fmtUSD(3.5) → "3.50"
function fmtUSD(monto) {
  if (monto == null || isNaN(monto)) return '-';
  return parseFloat(monto).toFixed(2);
}

// Escapa caracteres HTML para prevenir XSS.
function escHtml(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

// ============================================================================
// Inline Table Editing helpers
// ============================================================================

const UNIDADES_BASE = ['g', 'ml', 'u', 'l'];

function crearInputInline(valor, tipo = 'text', placeholder = '', clase = '', attrs = {}) {
  const input = document.createElement('input');
  input.type = tipo;
  input.value = (valor !== null && valor !== undefined) ? valor : '';
  input.placeholder = placeholder;
  input.className = clase || 'form-control';
  Object.entries(attrs).forEach(([k, v]) => input.setAttribute(k, v));
  return input;
}

function crearSelectUnidadInline(valorSeleccionado) {
  const select = document.createElement('select');
  select.className = 'form-control';
  UNIDADES_BASE.forEach(u => {
    const opt = document.createElement('option');
    opt.value = u;
    opt.textContent = u;
    if (u === valorSeleccionado) opt.selected = true;
    select.appendChild(opt);
  });
  return select;
}

function calcularMargenInline(fila) {
  const costoInput = fila.querySelector('[data-field="costo_promedio"] input');
  const ventaInput = fila.querySelector('[data-field="precio_venta_usd"] input');
  const margenCell = fila.querySelector('.col-margen');
  if (!costoInput || !ventaInput || !margenCell) return;

  const costo = parseFloat(costoInput.value) || 0;
  const venta = parseFloat(ventaInput.value) || 0;
  const tasa = tasaUSDActual || 1;

  let margenHtml = '<span class="valor-nd">—</span><br><span class="valor-nd">—</span>';
  if (venta > 0) {
    const margenPct = ((venta - costo) / venta) * 100;
    const margenMonto = venta - costo;
    const margenMontoBs = margenMonto * tasa;
    let margenClase = 'margen-rojo';
    if (margenPct >= 20) margenClase = 'margen-verde';
    else if (margenPct >= 5) margenClase = 'margen-ambar';
    margenHtml = `<span class="margen-badge ${margenClase}">${margenPct.toFixed(1)}%</span><br><span class="margen-monto">${fmtUSD(margenMonto)} USD / Bs ${formatBs(margenMontoBs)}</span>`;
  }
  margenCell.innerHTML = margenHtml;
}

function actualizarBsInline(fila) {
  const costoInput = fila.querySelector('[data-field="costo_promedio"] input');
  const ventaInput = fila.querySelector('[data-field="precio_venta_usd"] input');
  const tasa = tasaUSDActual || 1;
  const costoBs = fila.querySelector('.costo-bs-inline');
  const ventaBs = fila.querySelector('.venta-bs-inline');
  if (costoInput && costoBs) costoBs.textContent = 'Bs ' + formatBs((parseFloat(costoInput.value) || 0) * tasa);
  if (ventaInput && ventaBs) ventaBs.textContent = 'Bs ' + formatBs((parseFloat(ventaInput.value) || 0) * tasa);
}

function agregarFilaNuevaInsumo() {
  const tbody = document.getElementById('tbl-insumos');
  if (!tbody) return;

  // Si ya hay una fila nueva sin guardar, no crear otra
  if (tbody.querySelector('tr[data-insumo-id="nuevo"]')) {
    const inputNombre = tbody.querySelector('tr[data-insumo-id="nuevo"] [data-field="nombre"] input');
    if (inputNombre) inputNombre.focus();
    return;
  }

  const tr = document.createElement('tr');
  tr.className = 'fila-nueva';
  tr.setAttribute('data-insumo-id', 'nuevo');
  tr.setAttribute('data-editando', 'true');
  tr.innerHTML = renderizarFilaEditableHTML({ id: 'nuevo' });

  tbody.insertBefore(tr, tbody.firstChild);
  activarListenersEdicionInline(tr, true);
  activarAtajosTecladoInline(tr, () => guardarNuevaFila(tr.querySelector('button[onclick^="guardarNuevaFila"]')));
  tr.querySelector('[data-field="nombre"] input').focus();
}

function renderizarFilaEditableHTML(ins = {}) {
  const tasa = tasaUSDActual || 1;
  const unidadBase = (ins.unidad_base || 'u').trim();
  const stock = ins.stock || 0;
  const stockMinimo = ins.stock_minimo || 0;
  const costo = ins.costo_promedio || 0;
  const venta = ins.precio_venta_usd || 0;
  const nombre = ins.nombre || '';
  const presentacionNombre = ins.presentacion_nombre || '';
  const basePorPresentacion = ins.cantidad_base_por_presentacion || 1;
  const unidadesPorEmpaque = ins.unidades_por_empaque || 1;
  const contenidoMl = ins.contenido_ml_por_unidad || 0;
  const esEmpaqueCompuestoEditable = unidadesPorEmpaque > 1 && contenidoMl > 0;

  const cantidadPres = cantidadEnPresentacion(stock, basePorPresentacion);
  const pesoHumano = convertirAUnidadHumana(basePorPresentacion, unidadBase);
  const stockMinimoEnUnidades = esEmpaqueCompuestoEditable ? stockMinimo / contenidoMl : stockMinimo;
  // En edición inline el input inferior izquierdo representa el TOTAL de unidades para la cantidad de empaques actual.
  const totalUnidadesEditable = esEmpaqueCompuestoEditable ? cantidadPres * unidadesPorEmpaque : unidadesPorEmpaque;
  const totalBaseCalculado = esEmpaqueCompuestoEditable ? totalUnidadesEditable * contenidoMl : basePorPresentacion;

  const costoBs = costo * tasa;
  const ventaBs = venta * tasa;
  let margenHtml = '<span class="valor-nd">—</span><br><span class="valor-nd">—</span>';
  if (venta > 0) {
    const margenPct = ((venta - costo) / venta) * 100;
    const margenMonto = venta - costo;
    let margenClase = 'margen-rojo';
    if (margenPct >= 20) margenClase = 'margen-verde';
    else if (margenPct >= 5) margenClase = 'margen-ambar';
    margenHtml = `<span class="margen-badge ${margenClase}">${margenPct.toFixed(1)}%</span><br><span class="margen-monto">${fmtUSD(margenMonto)} USD / Bs ${formatBs(margenMonto * tasa)}</span>`;
  }

  return `
    <td class="col-producto td-input" data-field="nombre" data-original="${escHtml(nombre)}">
      <input type="text" class="form-control" value="${escHtml(nombre)}" placeholder="Nombre del producto" required />
    </td>
    <td class="col-cantidad td-input" data-field="cantidad_presentacion" data-unidad-base="${unidadBase}" data-original="${cantidadPres}" data-presentacion="${escHtml(presentacionNombre)}">
      <div class="input-unidad-pair">
        <input type="number" step="0.01" class="form-control input-cantidad-empaque" value="${cantidadPres > 0 ? cantidadPres : ''}" placeholder="0.00" required />
        <input type="text" class="form-control input-presentacion" value="${escHtml(presentacionNombre)}" placeholder="caja, saco..." required />
      </div>
    </td>
    <td class="col-peso td-input" data-field="presentacion_medida" data-unidad-base="${unidadBase}" data-original="${basePorPresentacion}" data-es-empaque-compuesto="${esEmpaqueCompuestoEditable ? '1' : '0'}">
      <div class="input-unidad-pair">
        <input type="number" step="0.01" class="form-control input-total-base" value="${totalBaseCalculado > 0 ? totalBaseCalculado.toLocaleString('es-VE', { minimumFractionDigits: 0, maximumFractionDigits: 2 }) : ''}" placeholder="0.00" ${esEmpaqueCompuestoEditable ? 'readonly' : 'required'} title="Total en unidad base (calculado automáticamente para empaques compuestos)" />
        <select class="form-control select-unidad-base" ${esEmpaqueCompuestoEditable ? 'disabled' : ''}>${UNIDADES_BASE.map(u => `<option value="${u}" ${u === unidadBase ? 'selected' : ''}>${u}</option>`).join('')}</select>
      </div>
      <div class="empaque-config-inline" style="margin-top:4px;">
        <div class="input-triple">
          <input type="number" step="1" min="1" class="form-control input-unidades-empaque" data-empaque="unidades_por_empaque" data-unidades-por-empaque-original="${unidadesPorEmpaque}" value="${totalUnidadesEditable > 0 ? totalUnidadesEditable.toLocaleString('es-VE', { minimumFractionDigits: 0, maximumFractionDigits: 0 }) : ''}" placeholder="Unid. totales" title="Total de unidades (cantidad × unidades por caja); editable manualmente" />
          <input type="number" step="0.1" min="0" class="form-control input-contenido-ml" data-empaque="contenido_ml" value="${contenidoMl > 0 ? contenidoMl : ''}" placeholder="ml/unid." title="ml por unidad (botella/lata)" />
          <label class="check-retornable" title="Envase retornable">
            <input type="checkbox" data-empaque="es_retornable" ${ins.es_retornable ? 'checked' : ''} /> ♻️
          </label>
        </div>
        <div class="empaque-hint">${esEmpaqueCompuestoEditable ? `${totalUnidadesEditable.toLocaleString('es-VE', { minimumFractionDigits: 0, maximumFractionDigits: 0 })} unid. × ${contenidoMl} ml` : 'Configure unid. × ml si aplica'}</div>
      </div>
    </td>
    <td class="col-stock-minimo td-input" data-field="stock_minimo" data-unidad-base="${unidadBase}" data-original="${stockMinimo}" data-es-empaque-compuesto="${esEmpaqueCompuestoEditable ? '1' : '0'}">
      <div class="input-unidad-pair">
        <input type="number" step="0.01" class="form-control" value="${stockMinimoEnUnidades > 0 ? stockMinimoEnUnidades : ''}" placeholder="0.00" required />
        <span class="unidad-tag">${esEmpaqueCompuestoEditable ? 'unid.' : (unidadBase || 'u')}</span>
      </div>
    </td>
    <td class="col-costo td-input" data-field="costo_promedio" data-unidad-base="${unidadBase}" data-original="${costo}">
      <input type="number" step="0.01" class="form-control" value="${costo > 0 ? costo : ''}" placeholder="0.00" />
      <div class="costo-bs-inline" style="font-size:0.75rem;color:var(--text-muted);margin-top:2px;">Bs ${formatBs(costoBs)}</div>
    </td>
    <td class="col-venta td-input" data-field="precio_venta_usd" data-unidad-base="${unidadBase}" data-original="${venta}">
      <input type="number" step="0.01" class="form-control" value="${venta > 0 ? venta : ''}" placeholder="0.00" />
      <div class="venta-bs-inline" style="font-size:0.75rem;color:var(--text-muted);margin-top:2px;">Bs ${formatBs(ventaBs)}</div>
    </td>
    <td class="col-margen td-margen">${margenHtml}</td>
    <td class="col-ajustes">
      <div class="ajustes-btns">
        ${ins.id && ins.id !== 'nuevo'
          ? `<button class="btn-guardar-inline" onclick="guardarEdicionFila(${ins.id})" title="Guardar">💾 Guardar</button>
             <button class="btn-cancelar-inline" onclick="cancelarEdicionFila(${ins.id})" title="Cancelar">❌ Cancelar</button>`
          : `<button class="btn-guardar-inline" onclick="guardarNuevaFila(this)" title="Guardar">💾 Guardar</button>
             <button class="btn-cancelar-inline" onclick="cancelarNuevaFila(this)" title="Cancelar">❌ Cancelar</button>`}
      </div>
    </td>
  `;
}

function activarListenersEdicionInline(fila, esNueva) {
  const costoInput = fila.querySelector('[data-field="costo_promedio"] input');
  const ventaInput = fila.querySelector('[data-field="precio_venta_usd"] input');
  const pesoSelect = fila.querySelector('[data-field="presentacion_medida"] select');

  [costoInput, ventaInput].forEach(input => {
    if (!input) return;
    input.addEventListener('input', () => {
      calcularMargenInline(fila);
      actualizarBsInline(fila);
    });
  });

  // Recalcular total base y hint para empaques compuestos
  const cantidadInput = fila.querySelector('[data-field="cantidad_presentacion"] input.input-cantidad-empaque');
  const unidadesInput = fila.querySelector('.input-unidades-empaque');
  const contenidoInput = fila.querySelector('.input-contenido-ml');
  const totalBaseInput = fila.querySelector('.input-total-base');
  const hintEl = fila.querySelector('.empaque-hint');

  function getUnidadesPorEmpaqueOriginal() {
    return parseInt(unidadesInput?.getAttribute('data-unidades-por-empaque-original')) || 1;
  }

  function setUnidadesPorEmpaqueOriginal(val) {
    if (unidadesInput) unidadesInput.setAttribute('data-unidades-por-empaque-original', val);
  }

  function parseLocaleInt(str) {
    if (typeof str !== 'string') return parseInt(str) || 0;
    // es-VE: 1.000 = 1000, 1,5 = 1.5 (no usamos decimales para unidades totales)
    return parseInt(str.replace(/\./g, '').replace(/,/g, '.').replace(/\.\d*/, '')) || 0;
  }

  function parseLocaleFloat(str) {
    if (typeof str !== 'string') return parseFloat(str) || 0;
    // es-VE: 15.984,00 → 15984.00
    return parseFloat(str.replace(/\./g, '').replace(',', '.')) || 0;
  }

  function recalcularDesdeCantidad() {
    if (!cantidadInput || !unidadesInput || !contenidoInput || !totalBaseInput) return;
    const cantidad = parseFloat(cantidadInput.value) || 0;
    const unidadesPorEmpaque = getUnidadesPorEmpaqueOriginal();
    if (cantidad > 0 && unidadesPorEmpaque > 1) {
      const totalUnidades = cantidad * unidadesPorEmpaque;
      unidadesInput.value = totalUnidades.toLocaleString('es-VE', { minimumFractionDigits: 0, maximumFractionDigits: 0 });
    }
    recalcularDesdeUnidades(false);
  }

  function recalcularDesdeUnidades(actualizarOriginal = true) {
    if (!cantidadInput || !unidadesInput || !contenidoInput || !totalBaseInput) return;
    const cantidad = parseFloat(cantidadInput.value) || 0;
    const totalUnidades = parseLocaleInt(unidadesInput.value) || 1;
    const contenido = parseLocaleFloat(contenidoInput.value) || 0;
    const esCompuesto = totalUnidades > 1 && contenido > 0;
    if (esCompuesto) {
      const totalBase = totalUnidades * contenido;
      totalBaseInput.value = totalBase > 0 ? totalBase.toLocaleString('es-VE', { minimumFractionDigits: 0, maximumFractionDigits: 2 }) : '';
      if (hintEl) hintEl.textContent = `${totalUnidades.toLocaleString('es-VE', { minimumFractionDigits: 0, maximumFractionDigits: 0 })} unid. × ${contenido} ml`;
      if (actualizarOriginal && cantidad > 0) {
        const nuevoUnidadesPorEmpaque = Math.max(1, Math.round(totalUnidades / cantidad));
        setUnidadesPorEmpaqueOriginal(nuevoUnidadesPorEmpaque);
      }
    } else {
      if (hintEl) hintEl.textContent = 'Configure unid. × ml si aplica';
    }
  }

  if (cantidadInput) cantidadInput.addEventListener('input', recalcularDesdeCantidad);
  if (unidadesInput) unidadesInput.addEventListener('input', () => recalcularDesdeUnidades(true));
  if (contenidoInput) contenidoInput.addEventListener('input', () => recalcularDesdeUnidades(false));

  if (pesoSelect) {
    pesoSelect.addEventListener('change', () => {
      const u = pesoSelect.value;
      fila.querySelectorAll('[data-unidad-base]').forEach(el => el.setAttribute('data-unidad-base', u));
    });
  }
}

function cancelarNuevaFila(btn) {
  const fila = btn.closest('tr');
  if (fila) fila.remove();
}

async function guardarNuevaFila(btn) {
  const fila = btn.closest('tr');
  if (!fila) return;

  const nombre = fila.querySelector('[data-field="nombre"] input')?.value.trim();
  const cantidadPresentacion = parseFloat(fila.querySelector('[data-field="cantidad_presentacion"] input.input-cantidad-empaque')?.value) || 0;
  const presentacionNombre = fila.querySelector('[data-field="cantidad_presentacion"] input.input-presentacion')?.value.trim() || 'unidad';
  const pesoPorPresentacion = parseFloat(fila.querySelector('[data-field="presentacion_medida"] input.input-total-base')?.value.replace(/\./g, '').replace(',', '.')) || 0;
  const unidadPeso = fila.querySelector('[data-field="presentacion_medida"] select.select-unidad-base')?.value || 'u';
  const stockMinimoInput = fila.querySelector('[data-field="stock_minimo"] input');
  const esEmpaqueCompuestoInput = fila.querySelector('[data-field="stock_minimo"]')?.getAttribute('data-es-empaque-compuesto') === '1';
  let stockMinimoUnidades = parseFloat(stockMinimoInput?.value) || 0;
  const costoPromedio = parseFloat(fila.querySelector('[data-field="costo_promedio"] input')?.value) || 0;
  const precioVentaUSD = parseFloat(fila.querySelector('[data-field="precio_venta_usd"] input')?.value) || 0;

  const totalUnidadesInput = fila.querySelector('[data-empaque="unidades_por_empaque"].input-unidades-empaque');
  const totalUnidades = parseInt((totalUnidadesInput?.value || '').replace(/\./g, '').replace(/,/g, '.').replace(/\.\d*/, '')) || 1;
  const contenidoMl = parseFloat(fila.querySelector('[data-empaque="contenido_ml"].input-contenido-ml')?.value) || 0;
  const esRetornable = fila.querySelector('[data-empaque="es_retornable"]')?.checked || false;

  if (!nombre) {
    showToast('El nombre del producto es obligatorio', true);
    return;
  }
  if (cantidadPresentacion <= 0) {
    showToast('La cantidad de presentaciones debe ser mayor a cero', true);
    return;
  }

  let basePorPresentacion, unidadBase, unidadesPorEmpaque;
  if (totalUnidades > 1 && contenidoMl > 0) {
    // El input visible es el total de unidades; derivamos unidades por empaque y base por presentación.
    unidadesPorEmpaque = Math.max(1, Math.round(totalUnidades / cantidadPresentacion));
    basePorPresentacion = (totalUnidades * contenidoMl) / cantidadPresentacion;
    unidadBase = 'ml';
  } else {
    unidadesPorEmpaque = 1;
    basePorPresentacion = convertirAUnidadBase(pesoPorPresentacion, unidadPeso, unidadPeso);
    unidadBase = unidadPeso;
  }

  // Convertir mínimo requerido de unidades a ml si es empaque compuesto.
  const stockMinimo = esEmpaqueCompuestoInput ? stockMinimoUnidades * contenidoMl : stockMinimoUnidades;

  if (basePorPresentacion <= 0) {
    showToast('El peso por presentación debe ser mayor a cero', true);
    return;
  }

  const stock = cantidadPresentacion * basePorPresentacion;

  try {
    const resInsumo = await apiFetch('/admin/api/insumos', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        nombre,
        unidad_base: unidadBase,
        stock_minimo: stockMinimo,
        stock: 0,
        precio_venta_usd: precioVentaUSD,
        presentacion_nombre: presentacionNombre,
        cantidad_base_por_presentacion: basePorPresentacion,
        empaque_tipo: 'UNITARIO',
        empaque_nombre: '',
        unidades_por_empaque: unidadesPorEmpaque,
        contenido_ml_por_unidad: contenidoMl,
        es_retornable: esRetornable
      })
    });
    if (!resInsumo.ok) throw new Error(await resInsumo.text());
    const insumoCreado = await resInsumo.json();
    const insumoId = insumoCreado.id;

    if (stock > 0) {
      const resCompra = await apiFetch('/admin/api/compras', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          insumo_id: insumoId,
          cantidad: stock,
          unidad_compra: unidadBase,
          costo_unitario: costoPromedio,
          referencia: 'Registro inicial - ' + nombre
        })
      });
      if (!resCompra.ok) throw new Error(await resCompra.text());
    }

    showToast('Producto registrado correctamente');
    cargarInsumos();
  } catch (err) {
    showToast(err.message, true);
  }
}

// ============================================================================
// Spec 006 — Sub-pestañas de Inventario, Mermas y Uso Personal
// ============================================================================

const SUBTAB_INVENTARIO_KEY = 'pos_subtab_inventario_activa';
let subtabInventarioActiva = 'inventario-principal';
let mermasHistoryCache = [];
let personalHistoryCache = [];

function hayFilaEnEdicion() {
  return document.querySelector('#tbl-insumos tr[data-editando="true"], #tbl-mermas tr[data-merma-id="nuevo"], #tbl-uso-personal tr[data-personal-id="nuevo"]') !== null;
}

function cambiarSubtabInventario(subtabId) {
  if (hayFilaEnEdicion()) {
    if (!confirm('Tiene cambios sin guardar en una fila abierta. ¿Desea cambiar de pestaña?')) {
      return;
    }
    cancelarFilasEdicionAbiertas();
  }

  subtabInventarioActiva = subtabId;
  sessionStorage.setItem(SUBTAB_INVENTARIO_KEY, subtabId);

  // Actualizar botones
  document.querySelectorAll('.subtab-btn').forEach(btn => {
    const isActive = btn.getAttribute('data-subtab') === subtabId;
    btn.classList.toggle('active', isActive);
    btn.setAttribute('aria-selected', isActive ? 'true' : 'false');
  });

  // Alternar paneles
  document.querySelectorAll('.subtab-panel').forEach(panel => {
    panel.classList.remove('active');
    panel.classList.add('hidden');
  });
  const activePanel = document.getElementById(`panel-${subtabId}`);
  if (activePanel) {
    activePanel.classList.remove('hidden');
    activePanel.classList.add('active');
  }

  // Alternar botones de acción
  document.querySelectorAll('.subtab-action-group').forEach(group => group.classList.add('hidden'));
  const activeActions = document.getElementById(`actions-${subtabId}`);
  if (activeActions) activeActions.classList.remove('hidden');

  cargarDatosSubtabActiva();
}

function restaurarSubtabInventario() {
  const guardada = sessionStorage.getItem(SUBTAB_INVENTARIO_KEY);
  if (guardada && ['inventario-principal', 'mermas', 'uso-personal'].includes(guardada)) {
    subtabInventarioActiva = guardada;
  }
  cambiarSubtabInventario(subtabInventarioActiva);
}

function cargarDatosSubtabActiva() {
  if (subtabInventarioActiva === 'inventario-principal') {
    cargarInsumos();
  } else if (subtabInventarioActiva === 'mermas') {
    cargarHistorialMermas();
  } else if (subtabInventarioActiva === 'uso-personal') {
    cargarHistorialUsoPersonal();
  }
}

function cancelarFilasEdicionAbiertas() {
  const filaInsumo = document.querySelector('#tbl-insumos tr[data-editando="true"]');
  if (filaInsumo) {
    const id = filaInsumo.getAttribute('data-insumo-id');
    if (id === 'nuevo') {
      filaInsumo.remove();
    } else {
      cargarInsumos();
    }
  }
  const filaMerma = document.querySelector('#tbl-mermas tr[data-merma-id="nuevo"]');
  if (filaMerma) filaMerma.remove();
  const filaPersonal = document.querySelector('#tbl-uso-personal tr[data-personal-id="nuevo"]');
  if (filaPersonal) filaPersonal.remove();
}

// Renderiza el HTML de una fila editable para Mermas.
function renderizarFilaEditableMerma() {
  const options = insumosCache.length > 0
    ? insumosCache.map(i => `<option value="${i.id}">${escHtml(i.nombre)} (Disp: ${formatStock(i.stock || 0, i.unidad_base)})</option>`).join('')
    : '<option value="" disabled>Cargando insumos...</option>';

  return `
    <td class="p-2.5">
      <select class="form-control select-insumo-inline" required onchange="actualizarIndicadorUnidadMerma(this)">
        <option value="">-- Seleccionar Producto del Almacén --</option>
        ${options}
      </select>
    </td>
    <td class="p-2.5 text-right">
      <div class="celda-cantidad-inline">
        <input type="number" step="0.01" min="0.01" class="form-control input-cantidad-inline" placeholder="0.00" required />
        <span class="unidad-indicador-inline">--</span>
      </div>
    </td>
    <td class="p-2.5 text-center">
      <div class="ajustes-btns">
        <button class="btn-guardar-inline btn-guardar-merma" onclick="guardarNuevaFilaMerma(this)" title="Registrar Merma">💾 Guardar</button>
        <button class="btn-cancelar-inline" onclick="this.closest('tr').remove()" title="Cancelar">❌ Cancelar</button>
      </div>
    </td>
  `;
}

// Renderiza el HTML de una fila editable para Uso Personal.
function renderizarFilaEditablePersonal() {
  const options = insumosCache.length > 0
    ? insumosCache.map(i => `<option value="${i.id}">${escHtml(i.nombre)} (Disp: ${formatStock(i.stock || 0, i.unidad_base)})</option>`).join('')
    : '<option value="" disabled>Cargando insumos...</option>';

  return `
    <td class="p-2.5">
      <select class="form-control select-insumo-inline" required onchange="actualizarIndicadorUnidadPersonal(this)">
        <option value="">-- Seleccionar Insumo para Cocina de Personal --</option>
        ${options}
      </select>
    </td>
    <td class="p-2.5 text-right">
      <div class="celda-cantidad-inline">
        <input type="number" step="0.01" min="0.01" class="form-control input-cantidad-inline" placeholder="0.00" required />
        <span class="unidad-indicador-inline">--</span>
      </div>
    </td>
    <td class="p-2.5 text-center">
      <div class="ajustes-btns">
        <button class="btn-guardar-inline" onclick="guardarNuevaFilaPersonal(this)" title="Registrar Consumo" style="background:var(--yellow-bg);color:var(--yellow);">💾 Guardar</button>
        <button class="btn-cancelar-inline" onclick="this.closest('tr').remove()" title="Cancelar">❌ Cancelar</button>
      </div>
    </td>
  `;
}

function agregarFilaNuevaMerma() {
  const tbody = document.getElementById('tbl-mermas');
  if (!tbody) return;
  if (tbody.querySelector('tr[data-merma-id="nuevo"]')) return;

  const tr = document.createElement('tr');
  tr.className = 'fila-inline-merma';
  tr.setAttribute('data-merma-id', 'nuevo');
  tr.innerHTML = renderizarFilaEditableMerma();
  tbody.insertBefore(tr, tbody.firstChild);

  const select = tr.querySelector('select');
  if (select) select.focus();
  activarAtajosTecladoInline(tr, () => guardarNuevaFilaMerma(tr.querySelector('.btn-guardar-merma')));
}

function agregarFilaNuevaUsoPersonal() {
  const tbody = document.getElementById('tbl-uso-personal');
  if (!tbody) return;
  if (tbody.querySelector('tr[data-personal-id="nuevo"]')) return;

  const tr = document.createElement('tr');
  tr.className = 'fila-inline-personal';
  tr.setAttribute('data-personal-id', 'nuevo');
  tr.innerHTML = renderizarFilaEditablePersonal();
  tbody.insertBefore(tr, tbody.firstChild);

  const select = tr.querySelector('select');
  if (select) select.focus();
  activarAtajosTecladoInline(tr, () => guardarNuevaFilaPersonal(tr.querySelector('button[onclick^="guardarNuevaFilaPersonal"]')));
}

function actualizarIndicadorUnidadMerma(select) {
  actualizarIndicadorUnidad(select);
}

function actualizarIndicadorUnidadPersonal(select) {
  actualizarIndicadorUnidad(select);
}

function actualizarIndicadorUnidad(select) {
  const fila = select.closest('tr');
  const insumoId = parseInt(select.value);
  const indicador = fila.querySelector('.unidad-indicador-inline');
  const insumo = insumosCache.find(i => i.id === insumoId);
  if (indicador) {
    indicador.textContent = insumo ? insumo.unidad_base : '--';
  }
}

function activarAtajosTecladoInline(fila, guardarFn) {
  const inputs = fila.querySelectorAll('input, select');
  inputs.forEach(input => {
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        guardarFn();
      } else if (e.key === 'Escape') {
        e.preventDefault();
        fila.remove();
      }
    });
  });
}

async function guardarNuevaFilaMerma(btn) {
  const fila = btn ? btn.closest('tr') : document.querySelector('#tbl-mermas tr[data-merma-id="nuevo"]');
  if (!fila) return;

  const select = fila.querySelector('select');
  const inputCantidad = fila.querySelector('input[type="number"]');
  const insumoId = parseInt(select.value);
  const cantidad = parseFloat(inputCantidad.value);

  if (!insumoId || isNaN(cantidad) || cantidad <= 0) {
    if (!insumoId) select.classList.add('input-error');
    if (isNaN(cantidad) || cantidad <= 0) inputCantidad.classList.add('input-error');
    showToast('Seleccione un producto y una cantidad mayor a cero', true);
    return;
  }

  select.classList.remove('input-error');
  inputCantidad.classList.remove('input-error');

  const insumo = insumosCache.find(i => i.id === insumoId);
  const unidad = insumo ? insumo.unidad_base : 'u';

  setBotonGuardando(btn, true);
  try {
    const res = await apiFetch('/admin/api/mermas', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        insumo_id: insumoId,
        cantidad: cantidad,
        unidad_medida: unidad,
        motivo: 'Merma operativa registrada desde panel admin'
      })
    });
    if (!res.ok) throw new Error(await res.text());

    fila.remove();
    actualizarStockEnCache(insumoId, cantidad);
    await cargarHistorialMermas();
    showToast('Merma registrada correctamente');
  } catch (err) {
    showToast(err.message, true);
  } finally {
    setBotonGuardando(btn, false);
  }
}

async function guardarNuevaFilaPersonal(btn) {
  const fila = btn ? btn.closest('tr') : document.querySelector('#tbl-uso-personal tr[data-personal-id="nuevo"]');
  if (!fila) return;

  const select = fila.querySelector('select');
  const inputCantidad = fila.querySelector('input[type="number"]');
  const insumoId = parseInt(select.value);
  const cantidad = parseFloat(inputCantidad.value);

  if (!insumoId || isNaN(cantidad) || cantidad <= 0) {
    if (!insumoId) select.classList.add('input-error');
    if (isNaN(cantidad) || cantidad <= 0) inputCantidad.classList.add('input-error');
    showToast('Seleccione un producto y una cantidad mayor a cero', true);
    return;
  }

  select.classList.remove('input-error');
  inputCantidad.classList.remove('input-error');

  const insumo = insumosCache.find(i => i.id === insumoId);
  const unidad = insumo ? insumo.unidad_base : 'u';

  setBotonGuardando(btn, true);
  try {
    const res = await apiFetch('/admin/api/uso-personal', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        insumo_id: insumoId,
        cantidad: cantidad,
        unidad_medida: unidad,
        detalle: 'Consumo de personal de cocina/sala'
      })
    });
    if (!res.ok) throw new Error(await res.text());

    fila.remove();
    actualizarStockEnCache(insumoId, cantidad);
    await cargarHistorialUsoPersonal();
    showToast('Consumo de personal registrado correctamente');
  } catch (err) {
    showToast(err.message, true);
  } finally {
    setBotonGuardando(btn, false);
  }
}

function setBotonGuardando(btn, guardando) {
  if (!btn) return;
  btn.disabled = guardando;
  btn.innerHTML = guardando
    ? '<span class="spinner-guardando"></span> Guardando...'
    : '💾 Guardar';
}

function actualizarStockEnCache(insumoId, cantidadDescontada) {
  const idx = insumosCache.findIndex(i => i.id === insumoId);
  if (idx >= 0) {
    insumosCache[idx].stock = Math.max(0, (insumosCache[idx].stock || 0) - cantidadDescontada);
    // Actualizar también la fila visible en el panel principal si está renderizada
    const fila = document.querySelector(`#tbl-insumos tr[data-insumo-id="${insumoId}"]`);
    if (fila && fila.getAttribute('data-editando') !== 'true') {
      const cantidadCell = fila.querySelector('[data-field="cantidad_presentacion"]');
      const pesoCell = fila.querySelector('[data-field="peso"]');
      if (cantidadCell) {
        const cantidadPres = cantidadEnPresentacion(insumosCache[idx].stock, insumosCache[idx].cantidad_base_por_presentacion);
        const esEmpaqueCompuesto = (insumosCache[idx].unidades_por_empaque || 1) > 1 && (insumosCache[idx].contenido_ml_por_unidad || 0) > 0;
        const cantidadDesglose = esEmpaqueCompuesto
          ? ` <span class="cantidad-desglose">(${(insumosCache[idx].stock / insumosCache[idx].contenido_ml_por_unidad).toLocaleString('es-VE', { minimumFractionDigits: 0, maximumFractionDigits: 1 })} unidades)</span>`
          : '';
        cantidadCell.innerHTML = formatCantidadPresentacion(cantidadPres, insumosCache[idx].presentacion_nombre) + cantidadDesglose;
        cantidadCell.setAttribute('data-original', cantidadPres);
      }
      if (pesoCell) {
        const pesoHumano = formatearPeso(insumosCache[idx].stock, insumosCache[idx].unidad_base);
        const esEmpaqueCompuesto = (insumosCache[idx].unidades_por_empaque || 1) > 1 && (insumosCache[idx].contenido_ml_por_unidad || 0) > 0;
        const pesoDesglose = esEmpaqueCompuesto
          ? ` <span class="peso-desglose">(${insumosCache[idx].unidades_por_empaque || 1} × ${(insumosCache[idx].contenido_ml_por_unidad || 0).toLocaleString('es-VE', { minimumFractionDigits: 0, maximumFractionDigits: 1 })} ml)</span>`
          : '';
        pesoCell.innerHTML = `<strong>${pesoHumano.valor.toLocaleString('es-VE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${pesoHumano.unidad}</strong><span class="peso-desglose">${pesoDesglose}</span>`;
        pesoCell.setAttribute('data-original', insumosCache[idx].stock);
      }
      // Refrescar estado crítico
      renderizarInsumos(insumosCache, tasaUSDActual);
    }
  }
}

async function cargarHistorialMermas() {
  try {
    const res = await apiFetch('/admin/api/mermas/historial');
    if (!res.ok) throw new Error('Error al cargar historial de mermas');
    mermasHistoryCache = await res.json();
    renderizarHistorialMermas(mermasHistoryCache);
    actualizarBadgeMermas();
  } catch (err) {
    console.error(err);
    const tbody = document.getElementById('tbl-mermas');
    if (tbody) tbody.innerHTML = `<tr><td colspan="3" class="empty-state">Error al cargar mermas: ${escHtml(err.message)}</td></tr>`;
  }
}

async function cargarHistorialUsoPersonal() {
  try {
    const res = await apiFetch('/admin/api/uso-personal/historial');
    if (!res.ok) throw new Error('Error al cargar historial de uso personal');
    personalHistoryCache = await res.json();
    renderizarHistorialUsoPersonal(personalHistoryCache);
    actualizarBadgePersonal();
  } catch (err) {
    console.error(err);
    const tbody = document.getElementById('tbl-uso-personal');
    if (tbody) tbody.innerHTML = `<tr><td colspan="3" class="empty-state">Error al cargar uso personal: ${escHtml(err.message)}</td></tr>`;
  }
}

function renderizarHistorialMermas(historial) {
  const tbody = document.getElementById('tbl-mermas');
  if (!tbody) return;

  // Preservar fila en edición si existe
  const filaEdicion = tbody.querySelector('tr[data-merma-id="nuevo"]');

  if (!historial || historial.length === 0) {
    tbody.innerHTML = `<tr><td colspan="3" class="empty-state">No hay mermas registradas</td></tr>`;
    if (filaEdicion) tbody.insertBefore(filaEdicion, tbody.firstChild);
    actualizarResumenMermas(0);
    return;
  }

  let totalHoy = 0;
  const hoy = new Date().toISOString().slice(0, 10);

  const filasHTML = historial.map(item => {
    const fecha = new Date(item.created_at);
    const fechaStr = fecha.toLocaleString('es-VE');
    const esHoy = item.created_at && item.created_at.slice(0, 10) === hoy;
    if (esHoy) totalHoy += item.cantidad;

    return `
      <tr class="fila-historial-merma">
        <td class="p-2.5">
          <strong>${escHtml(item.insumo_nombre || 'Insumo #' + item.insumo_id)}</strong>
          <div class="costo-historial">${formatStock(item.cantidad, item.unidad_base)} · Costo: $ ${fmtUSD(item.costo_total || 0)} / Bs ${formatBs((item.costo_total || 0) * (tasaUSDActual || 1))}</div>
        </td>
        <td class="p-2.5 text-right font-mono">${formatStock(item.cantidad, item.unidad_base)}</td>
        <td class="p-2.5 text-center celda-fecha-historial">${fechaStr}</td>
      </tr>
    `;
  }).join('');

  tbody.innerHTML = filasHTML;
  if (filaEdicion) tbody.insertBefore(filaEdicion, tbody.firstChild);
  actualizarResumenMermas(totalHoy);
}

function renderizarHistorialUsoPersonal(historial) {
  const tbody = document.getElementById('tbl-uso-personal');
  if (!tbody) return;

  const filaEdicion = tbody.querySelector('tr[data-personal-id="nuevo"]');

  if (!historial || historial.length === 0) {
    tbody.innerHTML = `<tr><td colspan="3" class="empty-state">No hay consumos de personal registrados</td></tr>`;
    if (filaEdicion) tbody.insertBefore(filaEdicion, tbody.firstChild);
    actualizarResumenPersonal(0);
    return;
  }

  let totalHoy = 0;
  const hoy = new Date().toISOString().slice(0, 10);

  const filasHTML = historial.map(item => {
    const fecha = new Date(item.created_at);
    const fechaStr = fecha.toLocaleString('es-VE');
    const esHoy = item.created_at && item.created_at.slice(0, 10) === hoy;
    if (esHoy) totalHoy += 1;

    const detalle = item.referencia && item.referencia.startsWith('USO_PERSONAL:')
      ? escHtml(item.referencia.substring(14).trim())
      : escHtml(item.referencia || '');

    return `
      <tr class="fila-historial-personal">
        <td class="p-2.5">
          <strong>${escHtml(item.insumo_nombre || 'Insumo #' + item.insumo_id)}</strong>
          <div class="costo-historial">${detalle} · Costo: $ ${fmtUSD(item.costo_total || 0)} / Bs ${formatBs((item.costo_total || 0) * (tasaUSDActual || 1))}</div>
        </td>
        <td class="p-2.5 text-right font-mono">${formatStock(item.cantidad, item.unidad_base)}</td>
        <td class="p-2.5 text-center celda-fecha-historial">${fechaStr}</td>
      </tr>
    `;
  }).join('');

  tbody.innerHTML = filasHTML;
  if (filaEdicion) tbody.insertBefore(filaEdicion, tbody.firstChild);
  actualizarResumenPersonal(totalHoy);
}

function actualizarResumenMermas(total) {
  const el = document.getElementById('stat-resumen-mermas');
  if (el) el.innerHTML = `Total Mermas Hoy: <strong>${formatStock(total, 'u')}</strong>`;
}

function actualizarResumenPersonal(total) {
  const el = document.getElementById('stat-resumen-personal');
  if (el) el.innerHTML = `Consumo Personal Hoy: <strong>${total} ítems</strong>`;
}

function actualizarBadgeMermas() {
  const badge = document.getElementById('badge-mermas-hoy');
  if (badge) badge.textContent = mermasHistoryCache.length;
}

function actualizarBadgePersonal() {
  const badge = document.getElementById('badge-personal-hoy');
  if (badge) badge.textContent = personalHistoryCache.length;
}

// Actualiza el badge de total de items en inventario principal
function actualizarBadgeTotalItems() {
  const badge = document.getElementById('badge-total-items');
  if (badge) badge.textContent = insumosCache.length;
}

document.addEventListener('DOMContentLoaded', () => {
  initClock();
  initTabs();
  cargarPerfil();
  cargarTasas();
  cargarUnidades();
  cargarTodo();

  // Spec 006 — Restaurar sub-pestaña activa de inventario
  restaurarSubtabInventario();

  // Spec 004 — Inicializar toggle de sonido y listener de tasa (TASK-010, TASK-009)
  inicializarToggleSonido();
  paginaInteractuada = false;
  document.addEventListener('click', function() { paginaInteractuada = true; }, { once: true });

  document.addEventListener('tasaActualizada', function(e) {
    const nuevaTasa = e.detail && e.detail.tasa ? parseFloat(e.detail.tasa) : null;
    if (nuevaTasa && !isNaN(nuevaTasa)) {
      tasaUSDActual = nuevaTasa;
      tasasCache.USD = nuevaTasa;
      recalcularValoresBs(nuevaTasa);
    }
  });

  const selCompra = document.getElementById('compra-insumo-id');
  if (selCompra) selCompra.addEventListener('change', actualizarUnidadesCompraPorInsumo);
  const selMerma = document.getElementById('merma-insumo-id');
  if (selMerma) selMerma.addEventListener('change', actualizarUnidadesMermaPorInsumo);

  // Spec 005 — Cálculo de Bs en tiempo real para costo de compra y costeo de empaque
  const costoCompraInput = document.getElementById('compra-costo');
  if (costoCompraInput) {
    costoCompraInput.addEventListener('input', function() {
      const usd = parseFloat(this.value) || 0;
      const bsEl = document.getElementById('compra-costo-bs');
      if (bsEl) {
        bsEl.textContent = 'Bs ' + formatBs(usd * tasaUSDActual);
      }
    });
  }

});

function initClock() {
  const timeEl = document.getElementById('current-time');
  const updateTime = () => {
    const now = new Date();
    timeEl.textContent = now.toLocaleDateString('es-ES', {
      weekday: 'short',
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };
  updateTime();
  setInterval(updateTime, 30000);
}

function initTabs() {
  const tabBtns = document.querySelectorAll('.tab-btn');
  tabBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      tabBtns.forEach(b => b.classList.remove('active'));
      document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));

      btn.classList.add('active');
      const tabId = btn.getAttribute('data-tab');
      const targetContent = document.getElementById(`tab-${tabId}`);
      if (targetContent) {
        targetContent.classList.add('active');
      }

      // Cargar datos específicos si es necesario
      if (tabId === 'dashboard') cargarStats();
      else if (tabId === 'inventario') {
        restaurarSubtabInventario();
        cargarDatosSubtabActiva();
      }
      else if (tabId === 'menu') cargarPlatos();
      else if (tabId === 'pedidos') cargarPedidos();
      else if (tabId === 'auditoria') cargarAuditoria();
      else if (tabId === 'conteo') poblarSelectInsumos('conteo-insumo-id');
      else if (tabId === 'usuarios') cargarUsuarios();
    });
  });
}

async function cargarTodo() {
  await Promise.all([cargarTasas(), cargarStats(), cargarInsumos(), cargarPlatos(), cargarUnidades()]);
}

// Carga las unidades de medida desde la API y pobla los selects correspondientes (Spec 003).
async function cargarUnidades() {
  try {
    const resp = await apiFetch('/admin/api/unidades');
    if (!resp.ok) throw new Error('Error al cargar unidades: ' + resp.status);
    const unidades = await resp.json();
    unidadesCache = Array.isArray(unidades) ? unidades : (unidades.data || []);
    poblarSelectsUnidades(unidadesCache);
  } catch (err) {
    console.warn('No se pudieron cargar unidades de medida:', err);
  }
}

// Pobla los selects con todas las unidades agrupadas por categoría según la Spec 003.
function poblarSelectsUnidades(unidades) {
  if (!Array.isArray(unidades) || unidades.length === 0) return;
  unidadesCache = unidades;

  const selectsBase = document.querySelectorAll('select[data-tipo="base"]');
  const selectsAll = document.querySelectorAll('select[data-tipo="all"]');

  const categorias = {
    'VOLUMEN': 'Volumen',
    'PESO': 'Peso / Masa',
    'CONTEO': 'Conteo / Empaque',
    'CULINARIA': 'Culinarias / Porciones'
  };

  const generarOpcionesCompletas = (placeholder) => {
    let html = `<option value="" disabled selected>${placeholder}</option>`;
    for (const [catKey, catLabel] of Object.entries(categorias)) {
      const items = unidades.filter(u => u.categoria === catKey);
      if (items.length > 0) {
        html += `<optgroup label="── ${catLabel} ──">`;
        html += items.map(u => {
          const badge = u.es_base ? ' ⭐ [Base]' : '';
          return `<option value="${u.abreviatura}">${u.nombre} (${u.abreviatura})${badge}</option>`;
        }).join('');
        html += `</optgroup>`;
      }
    }
    return html;
  };

  selectsBase.forEach(select => {
    const prevVal = select.value;
    select.innerHTML = generarOpcionesCompletas('Seleccionar unidad base...');
    if (prevVal) select.value = prevVal;
  });

  selectsAll.forEach(select => {
    const prevVal = select.value;
    const placeholder = select.id === 'compra-unidad' ? 'Seleccionar unidad de compra...' : 'Seleccionar unidad...';
    select.innerHTML = generarOpcionesCompletas(placeholder);
    if (prevVal) select.value = prevVal;
  });
}

// Pre-selecciona la unidad sugerida del insumo seleccionado sin borrar las demás opciones de la Spec 003
function actualizarUnidadesCompraPorInsumo() {
  const insumoId = parseInt(document.getElementById('compra-insumo-id').value);
  const selectUnidad = document.getElementById('compra-unidad');
  if (!selectUnidad || !unidadesCache || unidadesCache.length === 0) return;

  // Asegurar que el select tenga todas las unidades
  if (selectUnidad.options.length <= 1) {
    poblarSelectsUnidades(unidadesCache);
  }

  const insumo = insumosCache.find(i => i.id === insumoId);
  if (!insumo) return;

  // Pre-seleccionar la unidad base del insumo
  if (insumo.unidad_base) {
    selectUnidad.value = insumo.unidad_base;
  }
}

// Pre-selecciona la unidad de merma sugerida del insumo seleccionado
function actualizarUnidadesMermaPorInsumo() {
  const insumoId = parseInt(document.getElementById('merma-insumo-id').value);
  const selectUnidad = document.getElementById('merma-unidad');
  if (!selectUnidad || !unidadesCache || unidadesCache.length === 0) return;

  if (selectUnidad.options.length <= 1) {
    poblarSelectsUnidades(unidadesCache);
  }

  const insumo = insumosCache.find(i => i.id === insumoId);
  if (!insumo) return;

  if (insumo.unidad_base) {
    selectUnidad.value = insumo.unidad_base;
  }
}

function showToast(msg, isError = false) {
  const container = document.getElementById('toast-container');
  const toast = document.createElement('div');
  toast.className = `toast ${isError ? 'toast-error' : 'toast-success'}`;
  toast.innerHTML = `<span>${isError ? '❌' : '✅'}</span> <span>${msg}</span>`;
  container.appendChild(toast);
  setTimeout(() => {
    toast.remove();
  }, 4000);
}

function abrirModal(id) {
  document.getElementById(id).classList.add('active');
}

function cerrarModal(id) {
  document.getElementById(id).classList.remove('active');
}

// -------------------------------------------------------------
// APIs & Rendering
// -------------------------------------------------------------

async function cargarStats() {
  try {
    const res = await apiFetch('/admin/api/stats');
    if (!res.ok) throw new Error('Error al cargar stats');
    const data = await res.json();

    document.getElementById('stat-total-insumos').textContent = data.total_insumos;
    document.getElementById('stat-stock-bajo').textContent = data.stock_bajo;
    document.getElementById('stat-total-platos').textContent = data.total_platos;
    const totalUSD = data.total_ventas_bs || 0;
    const totalBs = totalUSD * (tasasCache.USD || 1);
    const ventasEl = document.getElementById('stat-total-ventas');
    if (ventasEl) {
      ventasEl.dataset.usd = totalUSD;
      ventasEl.innerHTML = `$ ${totalUSD.toFixed(2)}<br><span class="bs-value" data-usd="${totalUSD}" style="font-size: 0.9rem; color: #4ade80; font-weight: 600;">Bs ${formatBs(totalBs)}</span>`;
    }

    // Render tabla stock crítico
    cargarStockCritico();
  } catch (err) {
    console.error(err);
  }
}



async function cargarInsumos() {
  try {
    const res = await apiFetch('/admin/api/insumos');
    if (!res.ok) throw new Error('Error al listar insumos');
    insumosCache = await res.json();

    const tasa = tasasCache.USD || 1;
    tasaUSDActual = tasa;

    renderizarInsumos(insumosCache, tasa);
    actualizarBadgeTotalItems();
  } catch (err) {
    console.error(err);
  }
}

// ============================================================================
// Spec 004 — renderizarInsumos v2 con doble moneda, margen, merma y alertas (TASK-008)
// ============================================================================

function renderizarInsumos(insumos, tasaUSD) {
  const tbody = document.getElementById('tbl-insumos');
  if (!insumos || insumos.length === 0) {
    tbody.innerHTML = `<tr><td colspan="8" class="empty-state">No hay insumos registrados</td></tr>`;
    return;
  }

  let hayCritico = false;
  let stockBajoCount = 0;

  const filasHTML = insumos.map(ins => {
    const stock = parseFloat(ins.stock || 0);
    const stockMinimo = parseFloat(ins.stock_minimo || 0);
    const costoPromedio = parseFloat(ins.costo_promedio || 0);
    const precioVentaUSD = parseFloat(ins.precio_venta_usd || 0);
    const esCritico = stock <= stockMinimo;
    const unidadBaseTag = (ins.unidad_base || 'u').trim();

    if (esCritico) {
      hayCritico = true;
      stockBajoCount++;
    }

    const costoBs = costoPromedio * tasaUSD;
    const ventaBs = precioVentaUSD > 0 ? precioVentaUSD * tasaUSD : 0;

    let margenHtml = '<span class="valor-nd">—</span>';
    let margenMontoHtml = '<span class="valor-nd">—</span>';
    if (precioVentaUSD > 0) {
      const margenPct = ((precioVentaUSD - costoPromedio) / precioVentaUSD) * 100;
      const margenMonto = precioVentaUSD - costoPromedio;
      let margenClase = 'margen-rojo';
      if (margenPct >= 20) margenClase = 'margen-verde';
      else if (margenPct >= 5) margenClase = 'margen-ambar';
      margenHtml = `<span class="margen-badge ${margenClase}">${margenPct.toFixed(1)}%</span>`;
      margenMontoHtml = `<span class="margen-monto">${fmtUSD(margenMonto)} USD / <span class="bs-value" data-usd="${margenMonto}">Bs ${formatBs(margenMonto * tasaUSD)}</span></span>`;
    }

    const filaClase = esCritico ? 'row-critica' : '';
    const nombreEscapado = escHtml(ins.nombre || '');
    const nombreInput = (ins.nombre || '').replace(/'/g, "\\'");

    const cantidadPres = cantidadEnPresentacion(stock, ins.cantidad_base_por_presentacion);
    const pesoHumano = formatearPeso(stock, ins.unidad_base);
    const esEmpaqueCompuesto = (ins.unidades_por_empaque || 1) > 1 && (ins.contenido_ml_por_unidad || 0) > 0;
    const pesoDesglose = esEmpaqueCompuesto
      ? ` <span class="peso-desglose">(${ins.unidades_por_empaque || 1} × ${(ins.contenido_ml_por_unidad || 0).toLocaleString('es-VE', { minimumFractionDigits: 0, maximumFractionDigits: 1 })} ml)</span>`
      : '';
    const cantidadDesglose = esEmpaqueCompuesto
      ? ` <span class="cantidad-desglose">(${ins.unidades_totales_stock || ins.stock / ins.contenido_ml_por_unidad} unidades)</span>`
      : '';

    return `
      <tr class="${filaClase}" data-insumo-id="${ins.id}" data-editando="false">
        <td class="col-producto" data-field="nombre" data-original="${escHtml(ins.nombre)}">${nombreEscapado}</td>
        <td class="col-cantidad" data-field="cantidad_presentacion" data-unidad-base="${unidadBaseTag}" data-original="${cantidadPres}" data-presentacion="${escHtml(ins.presentacion_nombre || 'unidad')}">
          ${formatCantidadPresentacion(cantidadPres, ins.presentacion_nombre)}${cantidadDesglose}
        </td>
        <td class="col-peso" data-field="peso" data-unidad-base="${unidadBaseTag}" data-original="${stock}">
          <strong>${pesoHumano.valor.toLocaleString('es-VE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${pesoHumano.unidad}</strong>${pesoDesglose}
        </td>
        <td class="col-stock-minimo" data-field="stock_minimo" data-unidad-base="${unidadBaseTag}" data-original="${stockMinimo}">
          ${esEmpaqueCompuesto
            ? `<strong>${(ins.stock_minimo_en_unidades || 0).toLocaleString('es-VE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} unidades</strong><br><small>${formatStock(stockMinimo, ins.unidad_base)}</small>`
            : (ins.unidad_compra_abrev && ins.stock_minimo_en_compra > 0
              ? `<strong>${ins.stock_minimo_en_compra.toLocaleString('es-VE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${ins.unidad_compra_abrev}</strong><br><small>${formatStock(stockMinimo, ins.unidad_base)}</small>`
              : `<strong>${formatStock(stockMinimo, ins.unidad_base)}</strong><br><small>${formatStock(stockMinimo, ins.unidad_base)}</small>`)}
        </td>
        <td class="col-costo" data-field="costo_promedio" data-unidad-base="${unidadBaseTag}" data-original="${costoPromedio}">
          <span class="usd-value">$ ${fmtUSD(costoPromedio)}</span><br>
          <span class="bs-value" data-usd="${costoPromedio}">Bs ${formatBs(costoBs)}</span>
        </td>
        <td class="col-venta" data-field="precio_venta_usd" data-unidad-base="${unidadBaseTag}" data-original="${precioVentaUSD}">
          ${precioVentaUSD > 0
            ? `<span class="usd-value">$ ${fmtUSD(precioVentaUSD)}</span><br><span class="bs-value" data-usd="${precioVentaUSD}">Bs ${formatBs(ventaBs)}</span>`
            : `<span class="valor-nd">—</span><br><span class="valor-nd">—</span>`}
        </td>
        <td style="display:none;" data-unidad-base="${unidadBaseTag}" data-empaque-tipo="${ins.empaque_tipo || 'UNITARIO'}" data-empaque-nombre="${escHtml(ins.empaque_nombre || '')}" data-unidades-por-empaque="${ins.unidades_por_empaque || 1}" data-contenido-ml="${ins.contenido_ml_por_unidad || 0}" data-es-retornable="${ins.es_retornable ? 1 : 0}"></td>
        <td class="col-margen">
          ${margenHtml}<br>${margenMontoHtml}
        </td>
        <td class="col-ajustes">
          <div class="ajustes-btns">
            <button class="btn-icon btn-editar-fila" onclick="activarEdicionFila(${ins.id})" title="Editar fila">✏️</button>
            <button class="btn-icon btn-icon-danger" onclick="eliminarInsumo(${ins.id}, '${nombreInput}', ${stock})" title="Eliminar Insumo">🗑️</button>
          </div>
        </td>
      </tr>
    `;
  }).join('');

  tbody.innerHTML = filasHTML;

  const statBajo = document.getElementById('stat-stock-bajo');
  if (statBajo) statBajo.textContent = stockBajoCount;

  if (hayCritico) {
    dispararAlertaSonora();
  }
}

function actualizarMargenFila(fila, tasaUSD) {
  const costoCell = fila.querySelector('[data-field="costo_promedio"]');
  const ventaCell = fila.querySelector('[data-field="precio_venta_usd"]');
  const margenCell = fila.querySelector('.col-margen');

  if (!costoCell || !ventaCell || !margenCell) return;

  const costoText = costoCell.textContent.replace(/[^\d.,-]/g, '').replace(',', '.');
  const ventaText = ventaCell.textContent.replace(/[^\d.,-]/g, '').replace(',', '.');

  const costo = parseFloat(costoText) || 0;
  const venta = parseFloat(ventaText) || 0;

  const costoBs = costo * tasaUSD;
  const ventaBs = venta > 0 ? venta * tasaUSD : 0;

  const costoBsSpan = costoCell.querySelector('.bs-value');
  const ventaBsSpan = ventaCell.querySelector('.bs-value');
  if (costoBsSpan) costoBsSpan.textContent = 'Bs ' + formatBs(costoBs);
  if (ventaBsSpan) ventaBsSpan.textContent = venta > 0 ? 'Bs ' + formatBs(ventaBs) : '—';

  if (venta > 0) {
    const margenPct = ((venta - costo) / venta) * 100;
    const margenMonto = venta - costo;
    let margenClase = 'margen-rojo';
    if (margenPct >= 20) margenClase = 'margen-verde';
    else if (margenPct >= 5) margenClase = 'margen-ambar';

    const badge = fila.querySelector('.margen-badge');
    const monto = fila.querySelector('.margen-monto');
    if (badge) badge.textContent = margenPct.toFixed(1) + '%';
    if (badge) badge.className = 'margen-badge ' + margenClase;
    if (monto) monto.innerHTML = `${fmtUSD(margenMonto)} USD / <span class="bs-value" data-usd="${margenMonto}">Bs ${formatBs(margenMonto * tasaUSD)}</span>`;
  } else {
    const badge = fila.querySelector('.margen-badge');
    const monto = fila.querySelector('.margen-monto');
    if (badge) badge.textContent = '— %';
    if (badge) badge.className = 'margen-badge margen-rojo';
    if (monto) monto.innerHTML = '— USD / — Bs';
  }
}

// Activar modo edición inline en una fila existente: reemplaza celdas por inputs
function activarEdicionFila(insumoId) {
  const fila = document.querySelector(`tr[data-insumo-id="${insumoId}"]`);
  if (!fila) return;
  if (fila.getAttribute('data-editando') === 'true') return;

  const ins = insumosCache.find(i => i.id === insumoId);
  if (!ins) return;

  fila.setAttribute('data-editando', 'true');
  fila.classList.add('fila-editando');

  fila.innerHTML = renderizarFilaEditableHTML(ins);
  activarListenersEdicionInline(fila, false);
  activarAtajosEdicionExistente(fila, insumoId);
  fila.querySelector('[data-field="nombre"] input').focus();
}

function activarAtajosEdicionExistente(fila, insumoId) {
  const inputs = fila.querySelectorAll('input, select');
  inputs.forEach(input => {
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        guardarEdicionFila(insumoId);
      } else if (e.key === 'Escape') {
        e.preventDefault();
        cancelarEdicionFila(insumoId);
      }
    });
  });
}

function cancelarEdicionFila(insumoId) {
  cargarInsumos();
}

async function guardarEdicionFila(insumoId) {
  const fila = document.querySelector(`tr[data-insumo-id="${insumoId}"]`);
  if (!fila) return;

  const nombre = fila.querySelector('[data-field="nombre"] input')?.value.trim();
  const cantidadPresentacion = parseFloat(fila.querySelector('[data-field="cantidad_presentacion"] input.input-cantidad-empaque')?.value) || 0;
  const presentacionNombre = fila.querySelector('[data-field="cantidad_presentacion"] input.input-presentacion')?.value.trim() || 'unidad';
  const pesoPorPresentacion = parseFloat(fila.querySelector('[data-field="presentacion_medida"] input.input-total-base')?.value.replace(/\./g, '').replace(',', '.')) || 0;
  const unidadPeso = fila.querySelector('[data-field="presentacion_medida"] select.select-unidad-base')?.value || 'u';
  const stockMinimoInput = fila.querySelector('[data-field="stock_minimo"] input');
  const esEmpaqueCompuestoInput = fila.querySelector('[data-field="stock_minimo"]')?.getAttribute('data-es-empaque-compuesto') === '1';
  let stockMinimoUnidades = parseFloat(stockMinimoInput?.value) || 0;
  const costoPromedio = parseFloat(fila.querySelector('[data-field="costo_promedio"] input')?.value) || 0;
  const precioVentaUSD = parseFloat(fila.querySelector('[data-field="precio_venta_usd"] input')?.value) || 0;

  const totalUnidadesInput = fila.querySelector('[data-empaque="unidades_por_empaque"].input-unidades-empaque');
  const totalUnidades = parseInt((totalUnidadesInput?.value || '').replace(/\./g, '').replace(/,/g, '.').replace(/\.\d*/, '')) || 1;
  const contenidoMl = parseFloat(fila.querySelector('[data-empaque="contenido_ml"].input-contenido-ml')?.value) || 0;
  const esRetornable = fila.querySelector('[data-empaque="es_retornable"]')?.checked || false;

  const empaqueMeta = fila.querySelector('[data-empaque-tipo]') || {};
  const empaqueTipoOriginal = empaqueMeta.getAttribute ? empaqueMeta.getAttribute('data-empaque-tipo') : 'UNITARIO';
  const empaqueNombreOriginal = empaqueMeta.getAttribute ? empaqueMeta.getAttribute('data-empaque-nombre') : '';

  if (!nombre) {
    showToast('El nombre del producto es obligatorio', true);
    return;
  }
  if (cantidadPresentacion <= 0) {
    showToast('La cantidad de presentaciones debe ser mayor a cero', true);
    return;
  }

  let basePorPresentacion, unidadBase, unidadesPorEmpaque;
  if (totalUnidades > 1 && contenidoMl > 0) {
    // El input visible es el total de unidades; derivamos unidades por empaque y base por presentación.
    unidadesPorEmpaque = Math.max(1, Math.round(totalUnidades / cantidadPresentacion));
    basePorPresentacion = (totalUnidades * contenidoMl) / cantidadPresentacion;
    unidadBase = 'ml';
  } else {
    unidadesPorEmpaque = 1;
    basePorPresentacion = convertirAUnidadBase(pesoPorPresentacion, unidadPeso, unidadPeso);
    unidadBase = unidadPeso;
  }

  if (basePorPresentacion <= 0) {
    showToast('El peso por presentación debe ser mayor a cero', true);
    return;
  }

  // Convertir mínimo requerido de unidades a ml si es empaque compuesto.
  const stockMinimo = esEmpaqueCompuestoInput ? stockMinimoUnidades * contenidoMl : stockMinimoUnidades;
  const stock = cantidadPresentacion * basePorPresentacion;

  try {
    const res = await apiFetch(`/admin/api/insumos/${insumoId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        nombre,
        unidad_base: unidadBase,
        stock_minimo: stockMinimo,
        stock,
        costo_promedio: costoPromedio,
        precio_venta_usd: precioVentaUSD,
        presentacion_nombre: presentacionNombre,
        cantidad_base_por_presentacion: basePorPresentacion,
        empaque_tipo: empaqueTipoOriginal,
        empaque_nombre: empaqueNombreOriginal,
        unidades_por_empaque: unidadesPorEmpaque,
        contenido_ml_por_unidad: contenidoMl,
        es_retornable: esRetornable
      })
    });
    if (!res.ok) throw new Error(await res.text());

    showToast('Insumo actualizado correctamente');
    cargarInsumos();
  } catch (err) {
    showToast('Error al actualizar: ' + err.message, true);
    cargarInsumos();
  }
}

// Recalcular todos los valores Bs de la tabla tras cambio de tasa (TASK-009)
function recalcularValoresBs(nuevaTasa) {
  if (!nuevaTasa || isNaN(nuevaTasa) || nuevaTasa <= 0) return;
  tasaUSDActual = nuevaTasa;
  tasasCache.USD = nuevaTasa;

  document.querySelectorAll('.bs-value[data-usd]').forEach(function(el) {
    const usd = parseFloat(el.getAttribute('data-usd')) || 0;
    el.textContent = 'Bs ' + formatBs(usd * nuevaTasa);
  });

  const costoCompraInput = document.getElementById('compra-costo');
  if (costoCompraInput) {
    const usd = parseFloat(costoCompraInput.value) || 0;
    const bsEl = document.getElementById('compra-costo-bs');
    if (bsEl) {
      bsEl.textContent = 'Bs ' + formatBs(usd * nuevaTasa);
    }
  }
}

async function cargarStockCritico() {
  try {
    const res = await apiFetch('/admin/api/insumos');
    const insumos = await res.json();

    const tbody = document.getElementById('tbl-critical-stock');
    const criticos = insumos.filter(i => (i.stock || 0) <= (i.stock_minimo || 0));

    if (criticos.length === 0) {
      tbody.innerHTML = `<tr><td colspan="8" style="text-align: center; color: var(--success); font-weight: 500;">🎉 ¡Todos los insumos tienen stock suficiente!</td></tr>`;
      return;
    }

    tbody.innerHTML = criticos.map(ins => `
      <tr>
        <td>#${ins.id}</td>
        <td><strong>${ins.nombre}</strong></td>
        <td style="color: var(--danger); font-weight: 700;">${(ins.stock || 0).toFixed(2)} ${ins.unidad_base}</td>
        <td>${(ins.stock_minimo || 0).toFixed(2)} ${ins.unidad_base}</td>
        <td>${ins.unidad_base}</td>
        <td>
          <div class="stock-progress" style="width: 100px;">
            <div class="stock-progress-fill" style="width: 20%; background: var(--danger);"></div>
          </div>
        </td>
        <td><span class="badge badge-danger">ALERTA REPOSICIÓN</span></td>
      </tr>
    `).join('');
  } catch (err) {
    console.error(err);
  }
}

async function cargarPlatos() {
  try {
    const res = await apiFetch('/admin/api/platos');
    if (!res.ok) throw new Error('Error al cargar platos');
    platosCache = await res.json();

    const tbody = document.getElementById('tbl-platos');
    if (!platosCache || platosCache.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align: center;">No hay platos en el menú</td></tr>`;
      return;
    }

    tbody.innerHTML = platosCache.map(p => {
      const precioUSD = p.precio_venta || 0;
      const precioBs = precioUSD * (tasasCache.USD || 1);
      const recetaHtml = (p.receta && p.receta.length > 0)
        ? p.receta.map(r => `<span class="badge badge-secondary" style="margin-right: 4px; margin-bottom: 4px;">Insumo #${r.insumo_id}: ${r.cantidad_requerida}g/ml (+${r.porcentaje_merma}% merma)</span>`).join(' ')
        : `<span style="color: var(--text-muted); font-size: 0.85rem;">Sin ingredientes cargados</span>`;

      return `
        <tr>
          <td>#${p.id}</td>
          <td><strong>${p.nombre}</strong></td>
          <td>
            <strong style="color: #f8fafc; font-size: 0.95rem;">$ ${precioUSD.toFixed(2)}</strong><br>
            <span class="bs-value" data-usd="${precioUSD}" style="font-size: 0.8rem; color: #4ade80; font-weight: 600;">Bs ${formatBs(precioBs)}</span>
          </td>
          <td>${p.activo ? '<span class="badge badge-success">ACTIVO</span>' : '<span class="badge badge-danger">INACTIVO</span>'}</td>
          <td>${recetaHtml}</td>
        </tr>
      `;
    }).join('');

  } catch (err) {
    console.error(err);
  }
}

async function cargarPedidos() {
  try {
    const res = await apiFetch('/admin/api/pedidos');
    if (!res.ok) throw new Error('Error al cargar pedidos');
    const pedidos = await res.json();

    const tbody = document.getElementById('tbl-pedidos');
    if (!pedidos || pedidos.length === 0) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align: center;">No hay pedidos registrados</td></tr>`;
      return;
    }

    tbody.innerHTML = pedidos.map(p => {
      const estadoBadge = getEstadoBadge(p.estado);
      const detallesStr = (p.items || []).map(i => `${i.cantidad}x ${i.nombre}`).join(', ');
      const totalUSD = p.total || 0;
      const totalBs = totalUSD * (tasasCache.USD || 1);

      return `
        <tr>
          <td><strong>#${p.id}</strong></td>
          <td>Mesa ${p.mesa_id}</td>
          <td>${estadoBadge}</td>
          <td style="font-size: 0.9rem;">${detallesStr || 'Sin detalles'}</td>
          <td style="color: var(--success); font-weight: 700;">
            $ ${totalUSD.toFixed(2)}<br>
            <span class="bs-value" data-usd="${totalUSD}" style="font-size: 0.8rem; color: #4ade80; font-weight: 600;">Bs ${formatBs(totalBs)}</span>
          </td>
          <td style="color: var(--text-muted); font-size: 0.85rem;">${p.created_at || 'Reciente'}</td>
        </tr>
      `;
    }).join('');

  } catch (err) {
    console.error(err);
  }
}

function getEstadoBadge(estado) {
  switch (estado) {
    case 'PENDIENTE': return `<span class="badge badge-warning">PENDIENTE</span>`;
    case 'EN_COCINA': return `<span class="badge badge-info">EN COCINA</span>`;
    case 'LISTO': return `<span class="badge badge-success">LISTO</span>`;
    case 'ENTREGADO': return `<span class="badge badge-secondary">ENTREGADO</span>`;
    default: return `<span class="badge badge-danger">${estado}</span>`;
  }
}

async function cargarAuditoria() {
  try {
    const res = await apiFetch('/admin/api/auditoria');
    if (!res.ok) throw new Error('Error al cargar auditoría');
    const movs = await res.json();

    const tbody = document.getElementById('tbl-auditoria');
    if (!movs || movs.length === 0) {
      tbody.innerHTML = `<tr><td colspan="7" style="text-align: center;">No hay movimientos registrados</td></tr>`;
      return;
    }

    tbody.innerHTML = movs.reverse().map(m => {
      let tipoBadge = `<span class="badge badge-secondary">${m.tipo}</span>`;
      if (m.tipo === 'ENTRADA_COMPRA') tipoBadge = `<span class="badge badge-success">ENTRADA COMPRA</span>`;
      else if (m.tipo === 'SALIDA_VENTA') tipoBadge = `<span class="badge badge-info">SALIDA VENTA</span>`;
      else if (m.tipo === 'SALIDA_MERMA') tipoBadge = `<span class="badge badge-danger">SALIDA MERMA</span>`;
      else if (m.tipo === 'AJUSTE_MANUAL') tipoBadge = `<span class="badge badge-warning">AJUSTE MANUAL</span>`;

      return `
        <tr>
          <td>#${m.id}</td>
          <td>Insumo #${m.insumo_id}</td>
          <td>${tipoBadge}</td>
          <td><strong>${m.cantidad.toFixed(2)}</strong></td>
          <td>Bs ${m.costo_unitario.toFixed(2)}</td>
          <td style="font-size: 0.85rem;">${m.referencia}</td>
          <td style="color: var(--text-muted); font-size: 0.8rem;">${new Date(m.created_at).toLocaleString()}</td>
        </tr>
      `;
    }).join('');

  } catch (err) {
    console.error(err);
  }
}

function poblarSelectInsumos(selectId) {
  const select = document.getElementById(selectId);
  if (!select) return;
  select.innerHTML = insumosCache.map(i => `<option value="${i.id}">${i.nombre} (ID #${i.id} - ${(i.stock || 0).toFixed(2)} ${i.unidad_base})</option>`).join('');
}

// -------------------------------------------------------------
// Formularios & Acciones Modales
// -------------------------------------------------------------

function abrirModalNuevoPlato() {
  const container = document.getElementById('receta-items-container');
  container.innerHTML = '';
  agregarFilaReceta();
  abrirModal('modal-nuevo-plato');
}

function agregarFilaReceta() {
  const container = document.getElementById('receta-items-container');
  const div = document.createElement('div');
  div.className = 'receta-row';
  div.style.cssText = 'display: flex; gap: 0.5rem; margin-bottom: 0.5rem; align-items: center;';

  const options = insumosCache.map(i => `<option value="${i.id}">${i.nombre} (${i.unidad_base})</option>`).join('');

  div.innerHTML = `
    <select class="form-control r-insumo" style="flex: 2;" required>${options}</select>
    <input type="number" step="0.01" class="form-control r-cant" placeholder="Cant. Req." style="flex: 1;" required />
    <input type="number" step="0.1" class="form-control r-merma" placeholder="% Merma" style="flex: 1;" value="0" required />
    <button type="button" class="btn btn-danger btn-sm" onclick="this.parentElement.remove()">&times;</button>
  `;
  container.appendChild(div);
}

async function guardarNuevoPlato(e) {
  e.preventDefault();
  const nombre = document.getElementById('plato-nombre').value;
  const precio_venta = parseFloat(document.getElementById('plato-precio').value);

  const receta = [];
  const rows = document.querySelectorAll('.receta-row');
  rows.forEach(r => {
    const insumo_id = parseInt(r.querySelector('.r-insumo').value);
    const cantidad_requerida = parseFloat(r.querySelector('.r-cant').value);
    const porcentaje_merma = parseFloat(r.querySelector('.r-merma').value);
    if (insumo_id && cantidad_requerida > 0) {
      receta.push({ insumo_id, cantidad_requerida, porcentaje_merma });
    }
  });

  try {
    const res = await apiFetch('/admin/api/platos', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nombre, precio_venta, receta })
    });
    if (!res.ok) throw new Error(await res.text());

    showToast('Plato y ficha técnica guardados');
    cerrarModal('modal-nuevo-plato');
    e.target.reset();
    cargarTodo();
  } catch (err) {
    showToast(err.message, true);
  }
}

async function auditarStockFisico(e) {
  e.preventDefault();
  const insumo_id = parseInt(document.getElementById('conteo-insumo-id').value);
  const stock_fisico = parseFloat(document.getElementById('conteo-fisico-cant').value);

  try {
    const res = await apiFetch('/admin/api/auditoria-stock', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ insumo_id, stock_fisico })
    });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();

    const card = document.getElementById('res-auditoria-card');
    const det = document.getElementById('auditoria-detalles');
    card.style.display = 'block';

    const esDescuadre = Math.abs(data.diferencia) > 0.001;
    const diffColor = esDescuadre ? 'var(--danger)' : 'var(--success)';

    det.innerHTML = `
      <p><strong>Insumo:</strong> ${data.nombre_insumo} (ID #${data.insumo_id})</p>
      <p><strong>Stock Teórico en Sistema:</strong> ${data.stock_teorico.toFixed(2)} ${data.unidad_base}</p>
      <p><strong>Conteo Físico Real:</strong> ${data.stock_fisico.toFixed(2)} ${data.unidad_base}</p>
      <p style="color: ${diffColor}; font-weight: 700; margin-top: 0.5rem;">
        Diferencia de Stock: ${data.diferencia > 0 ? '+' : ''}${data.diferencia.toFixed(2)} ${data.unidad_base}
      </p>
      <p style="color: ${diffColor}; font-weight: 700;">
        Descuadre Valorizado Monetario: Bs ${data.descuadre_valorizado.toFixed(2)}
      </p>
    `;

    showToast('Auditoría calculada correctamente');
  } catch (err) {
    showToast(err.message, true);
  }
}

// ---- Gestión de Usuarios y Perfil RBAC ----

async function cargarPerfil() {
  try {
    const res = await apiFetch('/admin/api/perfil');
    if (res.ok) {
      const data = await res.json();
      const badge = document.getElementById('user-role-badge');
      if (badge) badge.textContent = data.rol ? data.rol.toUpperCase() : 'ADMIN';
      const nameDisp = document.getElementById('user-name-display');
      if (nameDisp) nameDisp.textContent = data.nombre || data.email;

      // Ocultar pestaña de usuarios a no-admins
      if (data.rol && data.rol !== 'admin') {
        const btnUsr = document.getElementById('tab-btn-usuarios');
        if (btnUsr) btnUsr.style.display = 'none';
      }
    }
  } catch (err) {
    console.error('Error cargando perfil:', err);
  }
}

async function cargarUsuarios() {
  const tbody = document.getElementById('tabla-usuarios');
  if (!tbody) return;
  tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;">Cargando usuarios...</td></tr>';

  try {
    const res = await apiFetch('/admin/api/usuarios');
    if (!res.ok) throw new Error(await res.text());
    const usuarios = await res.json();

    if (!usuarios || usuarios.length === 0) {
      tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;">No hay usuarios registrados.</td></tr>';
      return;
    }

    tbody.innerHTML = usuarios.map(u => {
      const fecha = new Date(u.CreatedAt).toLocaleDateString('es-ES', {
        year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
      });
      const estadoBadge = u.Activo
        ? '<span class="status-badge status-LISTO">Activo</span>'
        : '<span class="status-badge status-CANCELADO" style="background:rgba(239,68,68,0.2);color:#ef4444;">Inactivo</span>';

      const toggleBtnTexto = u.Activo ? 'Desactivar' : 'Activar';
      const toggleBtnClase = u.Activo ? 'btn-danger' : 'btn-success';

      return `
        <tr>
          <td>#${u.ID}</td>
          <td><strong>${u.Nombre}</strong></td>
          <td>${u.Email}</td>
          <td>
            <select onchange="cambiarRolUsuario(${u.ID}, this.value)" style="background:#0f172a;color:#cbd5e1;border:1px solid #334155;border-radius:4px;padding:3px 6px;font-size:0.8rem;">
              <option value="admin" ${u.Rol === 'admin' ? 'selected' : ''}>Admin</option>
              <option value="gerente" ${u.Rol === 'gerente' ? 'selected' : ''}>Gerente</option>
              <option value="cocinero" ${u.Rol === 'cocinero' ? 'selected' : ''}>Cocinero</option>
              <option value="mesero" ${u.Rol === 'mesero' ? 'selected' : ''}>Mesero</option>
              <option value="cajero" ${u.Rol === 'cajero' ? 'selected' : ''}>Cajero</option>
            </select>
          </td>
          <td>${estadoBadge}</td>
          <td style="font-size:0.8rem;color:#94a3b8;">${fecha}</td>
          <td>
            <button class="btn btn-sm ${toggleBtnClase}" onclick="toggleUsuario(${u.ID}, ${u.Activo})" style="padding:3px 8px;font-size:0.75rem;">
              ${toggleBtnTexto}
            </button>
          </td>
        </tr>
      `;
    }).join('');
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="7" style="color:var(--danger);text-align:center;">Error: ${err.message}</td></tr>`;
  }
}

function abrirModalUsuario() {
  document.getElementById('usr-nombre').value = '';
  document.getElementById('usr-email').value = '';
  document.getElementById('usr-password').value = '';
  document.getElementById('usr-rol').value = 'mesero';
  abrirModal('modal-nuevo-usuario');
}

async function guardarNuevoUsuario(e) {
  e.preventDefault();
  const nombre = document.getElementById('usr-nombre').value.trim();
  const email = document.getElementById('usr-email').value.trim();
  const password = document.getElementById('usr-password').value;
  const rol = document.getElementById('usr-rol').value;

  try {
    const res = await apiFetch('/admin/api/usuarios', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nombre, email, password, rol })
    });

    if (!res.ok) {
      const errData = await res.json().catch(() => ({ error: 'Error al crear usuario' }));
      throw new Error(errData.error || 'Error al crear usuario');
    }

    cerrarModal('modal-nuevo-usuario');
    showToast(`Usuario ${nombre} creado con éxito`);
    cargarUsuarios();
  } catch (err) {
    showToast(err.message, true);
  }
}

async function toggleUsuario(id, activoActual) {
  const nuevoActivo = !activoActual;
  const accion = nuevoActivo ? 'activar' : 'desactivar';
  if (!confirm(`¿Está seguro de que desea ${accion} este usuario?`)) return;

  try {
    const res = await apiFetch('/admin/api/usuarios/modificar', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, activo: nuevoActivo })
    });

    if (!res.ok) {
      const errData = await res.json().catch(() => ({ error: 'Error al modificar usuario' }));
      throw new Error(errData.error || 'Error al modificar usuario');
    }

    showToast(`Usuario ${nuevoActivo ? 'activado' : 'desactivado'} correctamente`);
    cargarUsuarios();
  } catch (err) {
    showToast(err.message, true);
  }
}

async function cambiarRolUsuario(id, nuevoRol) {
  try {
    const res = await apiFetch('/admin/api/usuarios/modificar', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, rol: nuevoRol })
    });

    if (!res.ok) {
      const errData = await res.json().catch(() => ({ error: 'Error al cambiar rol' }));
      throw new Error(errData.error || 'Error al cambiar rol');
    }

    showToast(`Rol actualizado a ${nuevoRol}`);
    cargarUsuarios();
  } catch (err) {
    showToast(err.message, true);
  }
}

// ---- Edición y Eliminación de Insumos ----

function abrirModalEditarInsumo(id) {
  // Delega a la edición en línea de la tabla.
  activarEdicionFila(id);
}

// guardarEditarInsumo ya no se usa; la edición se realiza en línea con activarEdicionFila/guardarEdicionFila.
async function guardarEditarInsumo(e) {
  e.preventDefault();
  showToast('Use la edición en línea de la tabla (botón ✏️) para editar insumos.', true);
}

async function eliminarInsumo(id, nombre, currentStock) {
  // En modo pruebas se permite eliminar aunque tenga stock; el backend limpia dependencias.
  if (!confirm(`¿Está seguro de eliminar permanentemente el insumo "${nombre}" (ID #${id})?\n\nEsta acción no se puede deshacer.`)) {
    return;
  }

  try {
    const res = await apiFetch(`/admin/api/insumos/${id}`, {
      method: 'DELETE'
    });

    if (!res.ok) {
      const errData = await res.json().catch(() => ({ error: 'Error al eliminar insumo' }));
      throw new Error(errData.error || 'Error al eliminar insumo');
    }

    showToast(`Insumo "${nombre}" eliminado correctamente`);
    cargarTodo();
  } catch (err) {
    showToast(err.message, true);
  }
}

// ============================================================================
// Spec 004 — Sistema de Alerta Sonora con Web Audio API (TASK-010)
// ============================================================================

function inicializarToggleSonido() {
  const btn = document.getElementById('btn-toggle-sonido');
  if (!btn) return;
  btn.textContent = alertaSonidoActiva ? '🔔' : '🔕';
  btn.addEventListener('click', toggleAlertaSonido);
}

function toggleAlertaSonido() {
  alertaSonidoActiva = !alertaSonidoActiva;
  localStorage.setItem('pos_alerta_sonido', String(alertaSonidoActiva));
  const btn = document.getElementById('btn-toggle-sonido');
  if (btn) btn.textContent = alertaSonidoActiva ? '🔔' : '🔕';
}

function mostrarBannerAutoplay() {
  const banner = document.getElementById('banner-autoplay');
  if (banner) {
    banner.style.display = 'block';
    setTimeout(() => { banner.style.display = 'none'; }, 8000);
  }
}

function dispararAlertaSonora() {
  if (alarmaDisparada || !alertaSonidoActiva) return;
  alarmaDisparada = true;
  setTimeout(function() { alarmaDisparada = false; }, 30000);

  if (!paginaInteractuada) {
    mostrarBannerAutoplay();
    return;
  }

  try {
    const ctx = new (window.AudioContext || window.webkitAudioContext)();
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.type = 'sine';
    osc.frequency.setValueAtTime(880, ctx.currentTime);
    gain.gain.setValueAtTime(0.4, ctx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.4);
    osc.start(ctx.currentTime);
    osc.stop(ctx.currentTime + 0.4);
    osc.onended = function() { ctx.close(); };
  } catch (err) {
    console.warn('[POS] Alerta sonora no disponible:', err);
  }
}

// Abre el modal de ajuste pre-seleccionando el insumo
function abrirModalAjusteConInsumo(insumoId) {
  const select = document.getElementById('ajuste-insumo-id');
  if (select) {
    select.value = insumoId;
  }
  abrirModal('modal-ajuste');
}

// ---- Gestión de Tasas de Cambio Oficiales (BCV) ----

async function cargarTasas(emitirEvento = true) {
  try {
    const res = await apiFetch('/admin/api/tasas');
    if (!res.ok) return;
    const tasas = await res.json();
    if (Array.isArray(tasas)) {
      tasas.forEach(t => {
        // Normalizar campos: la API Go puede devolver Moneda/Tasa (mayúscula) o moneda/tasa (minúscula)
        const moneda = t.moneda || t.Moneda;
        const tasa   = t.tasa   || t.Tasa;
        if (moneda && tasa) {
          tasasCache[moneda] = parseFloat(tasa);
        }
      });
    }

    const usdEl = document.getElementById('rate-usd-display');
    const eurEl = document.getElementById('rate-eur-display');
    if (usdEl) {
      usdEl.textContent = tasasCache.USD
        ? `💵 $ 1 = Bs ${tasasCache.USD.toFixed(2)}`
        : '💵 $ 1 = Bs --';
    }
    if (eurEl) {
      eurEl.textContent = tasasCache.EUR
        ? `💶 € 1 = Bs ${tasasCache.EUR.toFixed(2)}`
        : '💶 € 1 = Bs --';
    }

    // Actualizar tasa USD activa para cálculos de Bs
    if (tasasCache.USD) {
      tasaUSDActual = tasasCache.USD;
      if (emitirEvento) {
        document.dispatchEvent(new CustomEvent('tasaActualizada', { detail: { tasa: tasasCache.USD } }));
      }
    }
  } catch (err) {
    console.warn('Modo offline: no se pudieron cargar tasas de cambio:', err);
  }
}

// Auto-sincronizar con BCV y recalcular tasas cada 10 minutos (600,000 ms)
setInterval(async () => {
  try {
    const res = await apiFetch('/admin/api/tasas/sincronizar', { method: 'POST' });
    if (res.ok) {
      await cargarTasas(true);
    } else {
      await cargarTasas(true);
    }
  } catch (err) {
    await cargarTasas(true);
  }
}, 10 * 60 * 1000);

async function sincronizarTasasBCV(silencioso = false) {
  if (!silencioso) showToast('Consultando Banco Central de Venezuela...');
  try {
    const res = await apiFetch('/admin/api/tasas/sincronizar', { method: 'POST' });
    if (!res.ok) {
      const errData = await res.json().catch(() => ({ error: 'Error sincronizando con BCV' }));
      throw new Error(errData.error || 'Fallo de conexión');
    }
    const data = await res.json();
    if (!silencioso) showToast(data.mensaje || 'Tasas sincronizadas con éxito');
    await cargarTasas(true);
    cargarInsumos();
    cargarPlatos();
  } catch (err) {
    if (!silencioso) showToast(`Error BCV: ${err.message}. Puedes fijarla manualmente con ✏️`, true);
  }
}

function abrirModalTasaManual() {
  const moneda = document.getElementById('manual-tasa-moneda').value || 'USD';
  document.getElementById('manual-tasa-valor').value = (tasasCache[moneda] || 0).toFixed(2);
  abrirModal('modal-tasa-manual');
}

// Actualizar valor sugerido al cambiar moneda en el select
document.addEventListener('DOMContentLoaded', () => {
  const selMoneda = document.getElementById('manual-tasa-moneda');
  if (selMoneda) {
    selMoneda.addEventListener('change', (e) => {
      const m = e.target.value;
      document.getElementById('manual-tasa-valor').value = (tasasCache[m] || 0).toFixed(2);
    });
  }
});

async function guardarTasaManual(e) {
  e.preventDefault();
  const moneda = document.getElementById('manual-tasa-moneda').value;
  const tasa = parseFloat(document.getElementById('manual-tasa-valor').value);

  if (!tasa || tasa <= 0) {
    showToast('La tasa debe ser mayor a 0', true);
    return;
  }

  try {
    const res = await apiFetch('/admin/api/tasas/manual', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ moneda, tasa })
    });
    if (!res.ok) {
      const errData = await res.json().catch(() => ({ error: 'Error al guardar tasa' }));
      throw new Error(errData.error || 'Error al guardar tasa');
    }
    showToast(`Tasa de ${moneda} fijada a Bs ${tasa.toFixed(2)}`);
    cerrarModal('modal-tasa-manual');
    await cargarTasas();
    // Emitir evento para recálculo masivo de Bs sin recarga (Spec 004)
    document.dispatchEvent(new CustomEvent('tasaActualizada', { detail: { tasa: tasasCache.USD } }));
    cargarInsumos();
    cargarPlatos();
  } catch (err) {
    showToast(err.message, true);
  }
}