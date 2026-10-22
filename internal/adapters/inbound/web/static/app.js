const API = '/api';

function formatCurrency(v) {
    return new Intl.NumberFormat('es-US', {style: 'currency', currency: 'USD'}).format(v);
}

function formatDate(s) {
    if (!s) return '-';
    try {
        return new Date(s).toLocaleString('es-ES');
    } catch { return s; }
}

async function fetchAPI(path, opts = {}) {
    const res = await fetch(API + path, {
        headers: {'Content-Type': 'application/json'},
        ...opts
    });
    const data = await res.json().catch(() => null);
    if (!res.ok) {
        alert(data?.error || 'Error en la operación');
        throw new Error(data?.error || 'Error');
    }
    return data;
}

// Navegacion
document.querySelectorAll('.nav-link').forEach(link => {
    link.addEventListener('click', e => {
        // Only prevent default for links that switch views internally (have data-view)
        if (link.dataset.view) {
            e.preventDefault();
            document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));
            link.classList.add('active');
            const view = link.dataset.view;
            document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
            document.getElementById('view-' + view).classList.add('active');
            if (view === 'dashboard') cargarDashboard();
            if (view === 'insumos') cargarInsumos();
            if (view === 'platos') cargarPlatos();
            if (view === 'movimientos') cargarMovimientos();
        }
        // else: let the link navigate normally (e.g., to /admin, /kds, /logout)
    });
});

// Modales
function openModal(id) {
    document.getElementById(id).classList.add('active');
}
function closeModal(id) {
    document.getElementById(id).classList.remove('active');
}

// DASHBOARD
async function cargarDashboard() {
    const insumos = await fetchAPI('/insumos');
    const platos = await fetchAPI('/platos');
    document.getElementById('card-total-insumos').textContent = insumos.length;
    document.getElementById('card-total-platos').textContent = platos.length;
    const bajos = insumos.filter(i => i.stock <= i.stock_minimo);
    document.getElementById('card-alerta-baja').textContent = bajos.length;
    if (bajos.length > 0) document.getElementById('card-alerta-baja').style.color = 'var(--red)';

    const tbody = document.querySelector('#table-stock tbody');
    tbody.innerHTML = insumos.map(i => {
        const bajo = i.stock <= i.stock_minimo;
        const badge = bajo ? 'bg' : (i.stock < i.stock_minimo * 2 ? 'warning' : 'ok');
        const label = bajo ? 'CRÍTICO' : (i.stock < i.stock_minimo * 2 ? 'BAJO' : 'OK');
        return `<tr>
            <td>${i.nombre}</td>
            <td>${i.unidad_base}</td>
            <td>${i.stock.toFixed(i.unidad_base === 'u' ? 0 : 2)} ${i.unidad_base}</td>
            <td>${i.stock_minimo}</td>
            <td>${i.costo_promedio.toFixed(5)}</td>
            <td><span class="status-badge status-${badge}">${label}</span></td>
        </tr>`;
    }).join('');
}

// INSUMOS
async function cargarInsumos() {
    const insumos = await fetchAPI('/insumos');
    const tbody = document.querySelector('#table-insumos tbody');
    tbody.innerHTML = insumos.map(i => {
        const bajo = i.stock <= i.stock_minimo;
        return `<tr>
            <td>${i.id}</td>
            <td>${i.nombre}</td>
            <td>${i.unidad_base}</td>
            <td>${i.stock.toFixed(2)}</td>
            <td>${i.stock_minimo}</td>
            <td>${i.costo_promedio.toFixed(5)}</td>
            <td class="actions-cell">
                <button class="btn btn-sm btn-secondary" onclick="abrirCompra(${i.id})">📥 Compra</button>
                <button class="btn btn-sm" style="background:var(--yellow-bg);color:var(--yellow)" onclick="abrirMerma(${i.id})">📤 Merma</button>
                <button class="btn btn-sm btn-info" onclick="abrirEditarInsumo(${i.id})">✏️ Editar</button>
                <button class="btn btn-sm btn-danger" onclick="borrarInsumo(${i.id})">🗑️ Borrar</button>
            </td>
        </tr>`;
    }).join('');
}

function abrirCompra(id) {
    document.getElementById('compra-insumo-id').value = id;
    openModal('modal-compra');
}
function abrirMerma(id) {
    document.getElementById('merma-insumo-id').value = id;
    openModal('modal-merma');
}
function abrirEditarInsumo(id) {
    // Fetch the insumo data to populate the edit form
    fetchAPI(`/insumos/${id}`)
        .then(insumo => {
            document.getElementById('editar-insumo-id').value = insumo.id;
            document.getElementById('editar-insumo-nombre').value = insumo.nombre;
            document.getElementById('editar-insumo-unidad').value = insumo.unidad_base;
            document.getElementById('editar-insumo-stock-min').value = insumo.stock_minimo || 0;
            openModal('modal-editar-insumo');
        })
        .catch(err => {
            alert('Error al cargar los datos del insumo: ' + err.message);
        });
}
function borrarInsumo(id) {
    if (!confirm('¿Está seguro de que desea eliminar este insumo? Esta acción no se puede deshacer.')) {
        return;
    }
    
    fetchAPI(`/insumos/${id}`, {method: 'DELETE'})
        .then(() => {
            alert('Insumo eliminado exitosamente');
            cargarInsumos();
            cargarDashboard();
        })
        .catch(err => {
            alert('Error al eliminar el insumo: ' + err.message);
        });
}

document.getElementById('form-insumo').addEventListener('submit', async e => {
    e.preventDefault();
    const req = {
        nombre: document.getElementById('insumo-nombre').value,
        unidad_base: document.getElementById('insumo-unidad').value,
        stock_minimo: parseFloat(document.getElementById('insumo-stock-min').value) || 0
    };
    await fetchAPI('/insumos', {method: 'POST', body: JSON.stringify(req)});
    e.target.reset();
    closeModal('modal-nuevo-insumo');
    cargarInsumos();
    cargarDashboard();
});

document.getElementById('form-editar-insumo').addEventListener('submit', async e => {
    e.preventDefault();
    const id = document.getElementById('editar-insumo-id').value;
    if (!id) {
        alert('Error: ID de insumo no válido');
        return;
    }
    const req = {
        nombre: document.getElementById('editar-insumo-nombre').value,
        unidad_base: document.getElementById('editar-insumo-unidad').value,
        stock_minimo: parseFloat(document.getElementById('editar-insumo-stock-min').value) || 0
    };
    await fetchAPI(`/insumos/${id}`, {method: 'PUT', body: JSON.stringify(req)});
    e.target.reset();
    closeModal('modal-editar-insumo');
    cargarInsumos();
    cargarDashboard();
});

document.getElementById('form-compra').addEventListener('submit', async e => {
    e.preventDefault();
    const req = {
        insumo_id: parseInt(document.getElementById('compra-insumo-id').value),
        cantidad: parseFloat(document.getElementById('compra-cantidad').value),
        unidad_compra: document.getElementById('compra-unidad').value,
        costo_unitario: parseFloat(document.getElementById('compra-costo').value),
        referencia: document.getElementById('compra-referencia').value
    };
    await fetchAPI('/compras', {method: 'POST', body: JSON.stringify(req)});
    e.target.reset();
    closeModal('modal-compra');
    cargarInsumos();
    cargarDashboard();
});

document.getElementById('form-merma').addEventListener('submit', async e => {
    e.preventDefault();
    const req = {
        insumo_id: parseInt(document.getElementById('merma-insumo-id').value),
        cantidad: parseFloat(document.getElementById('merma-cantidad').value),
        unidad_medida: document.getElementById('merma-unidad').value,
        motivo: document.getElementById('merma-motivo').value
    };
    await fetchAPI('/mermas', {method: 'POST', body: JSON.stringify(req)});
    e.target.reset();
    closeModal('modal-merma');
    cargarInsumos();
    cargarDashboard();
});

// PLATOS
async function cargarPlatos() {
    const platos = await fetchAPI('/platos');
    const tbody = document.querySelector('#table-platos tbody');
    const select = document.getElementById('pedido-plato');
    tbody.innerHTML = platos.map(p => `
        <tr>
            <td>${p.id}</td>
            <td>${p.nombre}</td>
            <td>${formatCurrency(p.precio_venta)}</td>
            <td>${p.activo ? '✅ Sí' : '❌ No'}</td>
        </tr>
    `).join('');
    select.innerHTML = platos.filter(p => p.activo).map(p =>
        `<option value="${p.id}">${p.nombre} - ${formatCurrency(p.precio_venta)}</option>`
    ).join('');
}

// PEDIDOS
document.getElementById('form-pedido').addEventListener('submit', async e => {
    e.preventDefault();
    const mesaId = parseInt(document.getElementById('pedido-mesa').value);
    const platoId = parseInt(document.getElementById('pedido-plato').value);
    const cantidad = parseInt(document.getElementById('pedido-cantidad').value);

    const pedido = await fetchAPI('/pedidos', {
        method: 'POST',
        body: JSON.stringify({mesa_id: mesaId, items: [{plato_id: platoId, cantidad}]})
    });

    const result = await fetchAPI(`/pedidos/${pedido.id}/procesar`, {method: 'POST'});
    if (result.status === 'procesado') {
        alert(`✅ Pedido #${pedido.id} procesado. Total: ${formatCurrency(pedido.total)}`);
        cargarInsumos();
        cargarDashboard();
    }
});

async function cargarPedidosRecientes() {
    // Simplificado: no hay endpoint de listado, usamos insumos como indicador
    const tbody = document.querySelector('#table-pedidos-recientes tbody');
    tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:var(--text-muted)">Usa el formulario de arriba para crear pedidos</td></tr>';
}

// MOVIMIENTOS
async function cargarMovimientos() {
    const insumoId = document.getElementById('mov-insumo-id').value;
    const movs = await fetchAPI(`/insumos/${insumoId}/movimientos`);
    const tbody = document.querySelector('#table-movimientos tbody');
    tbody.innerHTML = movs.map(m => `
        <tr>
            <td>${m.id}</td>
            <td>${m.tipo}</td>
            <td>${m.cantidad.toFixed(2)}</td>
            <td>${m.costo_unitario.toFixed(5)}</td>
            <td>${m.referencia}</td>
            <td>${formatDate(m.created_at)}</td>
        </tr>
    `).join('');
}

// Inicializar
document.addEventListener('DOMContentLoaded', () => {
    cargarDashboard();
    cargarInsumos();
    cargarPlatos();
    cargarMovimientos();
});