let unidadesCache = [];

// State global
let insumosCache = [];
let platosCache = [];
let tasasCache = { USD: 877.90, EUR: 982.75 };

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

document.addEventListener('DOMContentLoaded', () => {
  initClock();
  initTabs();
  cargarPerfil();
  cargarTasas();
  cargarUnidades();
  cargarTodo();

  const selCompra = document.getElementById('compra-insumo-id');
  if (selCompra) selCompra.addEventListener('change', actualizarUnidadesCompraPorInsumo);
  const selMerma = document.getElementById('merma-insumo-id');
  if (selMerma) selMerma.addEventListener('change', actualizarUnidadesMermaPorInsumo);
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
      else if (tabId === 'inventario') cargarInsumos();
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
    document.getElementById('stat-total-ventas').textContent = `Bs ${data.total_ventas_bs.toFixed(2)}`;

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

    const tbody = document.getElementById('tbl-insumos');
    if (!insumosCache || insumosCache.length === 0) {
      tbody.innerHTML = `<tr><td colspan="8" style="text-align: center;">No hay insumos registrados</td></tr>`;
      return;
    }

    tbody.innerHTML = insumosCache.map(ins => {
      const currentStock = ins.stock || 0;
      const minStock = ins.stock_minimo || 0;
      const pct = Math.min(100, Math.round((currentStock / (minStock * 2 || 1)) * 100));
      const esBajo = currentStock <= minStock;
      const statusBadge = esBajo
        ? `<span class="badge badge-danger">STOCK BAJO</span>`
        : `<span class="badge badge-success">OK</span>`;
      const barColor = esBajo ? 'var(--danger)' : 'var(--success)';

      const nombreEscapado = (ins.nombre || '').replace(/'/g, "\\'");
      const costoUSD = ins.costo_promedio || 0;
      const costoBs = costoUSD * (tasasCache.USD || 1);

      return `
        <tr>
          <td>#${ins.id}</td>
          <td><strong>${ins.nombre}</strong></td>
          <td>${currentStock.toFixed(2)} ${ins.unidad_base}</td>
          <td>${minStock.toFixed(2)} ${ins.unidad_base}</td>
          <td><span class="badge badge-info">${ins.unidad_base}</span></td>
          <td>
            <strong style="color: #f8fafc;">$ ${costoUSD.toFixed(2)}</strong> <span style="font-size:0.75rem; color:#94a3b8;">/ ${ins.unidad_base}</span><br>
            <span style="font-size:0.75rem; color:#4ade80; font-weight:600;">Bs ${costoBs.toFixed(2)}</span>
          </td>
          <td style="width: 150px;">
            <div style="font-size: 0.75rem; display: flex; justify-content: space-between; margin-bottom: 2px;">
              <span>${pct}%</span>
            </div>
            <div class="stock-progress">
              <div class="stock-progress-fill" style="width: ${pct}%; background: ${barColor};"></div>
            </div>
          </td>
          <td>
            <div style="display: flex; gap: 0.35rem; align-items: center; flex-wrap: wrap;">
              ${statusBadge}
              <button class="btn btn-sm btn-secondary" style="padding: 3px 8px; font-size: 0.75rem;" onclick="abrirModalEditarInsumo(${ins.id})" title="Editar Insumo">✏️ Editar</button>
              <button class="btn btn-sm btn-danger" style="padding: 3px 8px; font-size: 0.75rem;" onclick="eliminarInsumo(${ins.id}, '${nombreEscapado}', ${currentStock})" title="Eliminar Insumo">🗑️</button>
            </div>
          </td>
        </tr>
      `;
    }).join('');

  } catch (err) {
    console.error(err);
  }
}

async function cargarStockCritico() {
  try {
    const res = await apiFetch('/admin/api/insumos');
    const insumos = await res.json();

    const tbody = document.getElementById('tbl-critical-stock');
    const criticos = insumos.filter(i => (i.stock || 0) <= (i.stock_minimo || 0));

    if (criticos.length === 0) {
      tbody.innerHTML = `<tr><td colspan="7" style="text-align: center; color: var(--success); font-weight: 500;">🎉 ¡Todos los insumos tienen stock suficiente!</td></tr>`;
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
            <span style="font-size: 0.8rem; color: #4ade80; font-weight: 600;">Bs ${precioBs.toFixed(2)}</span>
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

      return `
        <tr>
          <td><strong>#${p.id}</strong></td>
          <td>Mesa ${p.mesa_id}</td>
          <td>${estadoBadge}</td>
          <td style="font-size: 0.9rem;">${detallesStr || 'Sin detalles'}</td>
          <td style="color: var(--success); font-weight: 700;">Bs ${p.total.toFixed(2)}</td>
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

function abrirModalNuevoInsumo() {
  abrirModal('modal-nuevo-insumo');
}

async function guardarNuevoInsumo(e) {
  e.preventDefault();
  const nombre = document.getElementById('ins-nombre').value;
  const unidad_base = document.getElementById('ins-unidad').value;
  const stock_minimo = parseFloat(document.getElementById('ins-minimo').value);

  try {
    const res = await apiFetch('/admin/api/insumos', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nombre, unidad_base, stock_minimo })
    });
    if (!res.ok) throw new Error(await res.text());

    showToast('Insumo guardado correctamente');
    cerrarModal('modal-nuevo-insumo');
    e.target.reset();
    cargarTodo();
  } catch (err) {
    showToast(err.message, true);
  }
}

function abrirModalCompra() {
  poblarSelectInsumos('compra-insumo-id');
  actualizarUnidadesCompraPorInsumo();
  abrirModal('modal-compra');
}

async function guardarCompra(e) {
  e.preventDefault();
  const insumo_id = parseInt(document.getElementById('compra-insumo-id').value);
  const cantidad = parseFloat(document.getElementById('compra-cantidad').value);
  const unidad_compra = document.getElementById('compra-unidad').value;
  const costo_unitario = parseFloat(document.getElementById('compra-costo').value);
  const referencia = document.getElementById('compra-ref').value;

  try {
    const res = await apiFetch('/admin/api/compras', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ insumo_id, cantidad, unidad_compra, costo_unitario, referencia })
    });
    if (!res.ok) throw new Error(await res.text());

    showToast('Entrada de compra registrada y stock actualizado');
    cerrarModal('modal-compra');
    e.target.reset();
    cargarTodo();
  } catch (err) {
    showToast(err.message, true);
  }
}

function abrirModalMerma() {
  poblarSelectInsumos('merma-insumo-id');
  actualizarUnidadesMermaPorInsumo();
  abrirModal('modal-merma');
}

async function guardarMerma(e) {
  e.preventDefault();
  const insumo_id = parseInt(document.getElementById('merma-insumo-id').value);
  const cantidad = parseFloat(document.getElementById('merma-cantidad').value);
  const unidad_medida = document.getElementById('merma-unidad').value;
  const motivo = document.getElementById('merma-motivo').value;

  try {
    const res = await apiFetch('/admin/api/mermas', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ insumo_id, cantidad, unidad_medida, motivo })
    });
    if (!res.ok) throw new Error(await res.text());

    showToast('Merma registrada y stock descontado');
    cerrarModal('modal-merma');
    e.target.reset();
    cargarTodo();
  } catch (err) {
    showToast(err.message, true);
  }
}

function abrirModalAjuste() {
  poblarSelectInsumos('ajuste-insumo-id');
  abrirModal('modal-ajuste');
}

async function guardarAjuste(e) {
  e.preventDefault();
  const insumo_id = parseInt(document.getElementById('ajuste-insumo-id').value);
  const stock_fisico = parseFloat(document.getElementById('ajuste-stock').value);
  const motivo = document.getElementById('ajuste-motivo').value;

  try {
    const res = await apiFetch('/admin/api/ajustes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ insumo_id, stock_fisico, motivo })
    });
    if (!res.ok) throw new Error(await res.text());

    showToast('Ajuste de inventario aplicado correctamente');
    cerrarModal('modal-ajuste');
    e.target.reset();
    cargarTodo();
  } catch (err) {
    showToast(err.message, true);
  }
}

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
  const ins = insumosCache.find(i => i.id === id);
  if (!ins) {
    showToast('Insumo no encontrado', true);
    return;
  }
  document.getElementById('edit-ins-id').value = ins.id;
  document.getElementById('edit-ins-nombre').value = ins.nombre;
  document.getElementById('edit-ins-unidad').value = ins.unidad_base;
  document.getElementById('edit-ins-minimo').value = ins.stock_minimo;
  abrirModal('modal-editar-insumo');
}

async function guardarEditarInsumo(e) {
  e.preventDefault();
  const id = parseInt(document.getElementById('edit-ins-id').value);
  const nombre = document.getElementById('edit-ins-nombre').value.trim();
  const unidad_base = document.getElementById('edit-ins-unidad').value;
  const stock_minimo = parseFloat(document.getElementById('edit-ins-minimo').value);

  if (!nombre) {
    showToast('El nombre del insumo es obligatorio', true);
    return;
  }

  try {
    const res = await apiFetch(`/admin/api/insumos/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nombre, unidad_base, stock_minimo })
    });

    if (!res.ok) {
      const errData = await res.json().catch(() => ({ error: 'Error al actualizar insumo' }));
      throw new Error(errData.error || 'Error al actualizar insumo');
    }

    showToast('Insumo actualizado correctamente');
    cerrarModal('modal-editar-insumo');
    cargarTodo();
  } catch (err) {
    showToast(err.message, true);
  }
}

async function eliminarInsumo(id, nombre, currentStock) {
  if (currentStock > 0) {
    alert(`No se puede eliminar el insumo "${nombre}" porque aún cuenta con stock disponible (${currentStock}). Debe ajustar o consumir el stock a 0 antes de eliminarlo.`);
    return;
  }

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

// ---- Gestión de Tasas de Cambio Oficiales (BCV) ----

async function cargarTasas() {
  try {
    const res = await apiFetch('/admin/api/tasas');
    if (!res.ok) return;
    const tasas = await res.json();
    if (Array.isArray(tasas)) {
      tasas.forEach(t => {
        if (t.moneda && t.tasa) {
          tasasCache[t.moneda] = t.tasa;
        }
      });
    }

    const usdEl = document.getElementById('rate-usd-display');
    const eurEl = document.getElementById('rate-eur-display');
    if (usdEl && tasasCache.USD) {
      usdEl.textContent = `💵 $ 1 = Bs ${tasasCache.USD.toFixed(2)}`;
    }
    if (eurEl && tasasCache.EUR) {
      eurEl.textContent = `💶 € 1 = Bs ${tasasCache.EUR.toFixed(2)}`;
    }
  } catch (err) {
    console.warn('Modo offline: no se pudieron cargar tasas de cambio:', err);
  }
}

async function sincronizarTasasBCV() {
  showToast('Consultando Banco Central de Venezuela...');
  try {
    const res = await apiFetch('/admin/api/tasas/sincronizar', { method: 'POST' });
    if (!res.ok) {
      const errData = await res.json().catch(() => ({ error: 'Error sincronizando con BCV' }));
      throw new Error(errData.error || 'Fallo de conexión');
    }
    const data = await res.json();
    showToast(data.mensaje || 'Tasas sincronizadas con éxito');
    await cargarTasas();
    cargarInsumos();
    cargarPlatos();
  } catch (err) {
    showToast(`Error BCV: ${err.message}. Puedes fijarla manualmente con ✏️`, true);
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
    cargarInsumos();
    cargarPlatos();
  } catch (err) {
    showToast(err.message, true);
  }
}