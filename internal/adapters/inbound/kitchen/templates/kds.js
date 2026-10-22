const API_BASE = '/kds/api';
const SSE_ENDPOINT = '/kds/ws';

const statusColumns = {
    'PENDIENTE': document.getElementById('column-PENDIENTE'),
    'EN_COCINA': document.getElementById('column-EN_COCINA'),
    'LISTO': document.getElementById('column-LISTO'),
};

const statusCounts = {
    'PENDIENTE': document.getElementById('count-PENDIENTE'),
    'EN_COCINA': document.getElementById('count-EN_COCINA'),
    'LISTO': document.getElementById('count-LISTO'),
};

const connectionStatus = document.getElementById('connection-status');
const orderCountEl = document.getElementById('order-count');

let eventSource = null;

function init() {
    loadOrders();
    connectSSE();
    setInterval(loadOrders, 30000);
}

async function loadOrders() {
    try {
        const res = await fetch(API_BASE + '/orders');
        if (!res.ok) throw new Error('Error cargando órdenes');
        const orders = await res.json();
        renderOrders(orders);
    } catch (err) {
        console.error('Error loading orders:', err);
        setConnectionStatus(false);
    }
}

function renderOrders(orders) {
    Object.values(statusColumns).forEach(col => col.innerHTML = '');
    const counts = { PENDIENTE: 0, EN_COCINA: 0, LISTO: 0 };

    orders.forEach(order => {
        const status = order.estado || 'PENDIENTE';
        if (counts[status] !== undefined) counts[status]++;
        const card = createOrderCard(order);
        if (statusColumns[status]) {
            statusColumns[status].appendChild(card);
        }
    });

    Object.keys(counts).forEach(key => {
        if (statusCounts[key]) statusCounts[key].textContent = counts[key];
    });

    orderCountEl.textContent = `Órdenes: ${orders.length}`;
    setConnectionStatus(true);
}

function createOrderCard(order) {
    const card = document.createElement('div');
    card.className = `order-card ${getStatusClass(order.estado)}`;
    card.dataset.orderId = order.id;

    const itemsHtml = order.items.map(item => `
        <div class="order-item">
            <span class="item-name">${item.nombre}</span>
            <span class="item-qty">x${item.cantidad}</span>
        </div>
    `).join('');

    const actions = getActionsForStatus(order.estado, order.id);

    card.innerHTML = `
        <div class="order-header">
            <span class="order-id">#${order.id}</span>
            <span class="order-mesa">Mesa ${order.mesa_id}</span>
        </div>
        <div class="order-items">
            ${itemsHtml}
        </div>
        <div class="order-total">Total: $${order.total.toFixed(2)}</div>
        <div class="order-actions">
            ${actions}
        </div>
    `;

    return card;
}

function getStatusClass(status) {
    switch (status) {
        case 'PENDIENTE': return 'pending';
        case 'EN_COCINA': return 'cooking';
        case 'LISTO': return 'ready';
        default: return '';
    }
}

function getActionsForStatus(status, orderId) {
    switch (status) {
        case 'PENDIENTE':
            return `<button class="btn btn-primary" onclick="updateStatus(${orderId}, 'EN_COCINA')">Iniciar</button>`;
        case 'EN_COCINA':
            return `<button class="btn btn-success" onclick="updateStatus(${orderId}, 'LISTO')">Listo</button>
                    <button class="btn btn-secondary" onclick="updateStatus(${orderId}, 'PENDIENTE')">Volver</button>`;
        case 'LISTO':
            return `<button class="btn btn-secondary" onclick="updateStatus(${orderId}, 'ENTREGADO')">Entregar</button>
                    <button class="btn btn-secondary" onclick="updateStatus(${orderId}, 'EN_COCINA')">Volver</button>`;
        default:
            return `<span style="color: #4ade80; font-weight: 600;">✓ Entregado</span>`;
    }
}

async function updateStatus(orderId, newStatus) {
    try {
        const res = await fetch(`/kds/api/pedidos/${orderId}/estado`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ estado: newStatus })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'Error actualizando estado');
        }
        loadOrders();
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

function connectSSE() {
    if (eventSource) {
        eventSource.close();
    }

    eventSource = new EventSource(SSE_ENDPOINT);

    eventSource.onopen = () => {
        setConnectionStatus(true);
    };

    eventSource.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            handleSSEEvent(data);
        } catch (e) {
            console.error('Error parsing SSE:', e);
        }
    };

    eventSource.onerror = () => {
        setConnectionStatus(false);
        setTimeout(connectSSE, 5000);
    };
}

function handleSSEEvent(event) {
    switch (event.type) {
        case 'new_order':
            addOrderToColumn(event.data);
            break;
        case 'status_change':
            moveOrderToColumn(event.data.id, event.data.estado);
            break;
    }
    loadOrders();
}

function addOrderToColumn(order) {
    const status = order.estado || 'PENDIENTE';
    if (statusColumns[status]) {
        const card = createOrderCard(order);
        card.classList.add('new-order');
        statusColumns[status].prepend(card);
        updateCounts();
    }
}

function moveOrderToColumn(orderId, newStatus) {
    const card = document.querySelector(`.order-card[data-order-id="${orderId}"]`);
    if (card && statusColumns[newStatus]) {
        card.className = `order-card ${getStatusClass(newStatus)}`;
        card.querySelector('.order-actions').innerHTML = getActionsForStatus(newStatus, orderId);
        statusColumns[newStatus].prepend(card);
        updateCounts();
    }
}

function updateCounts() {
    const counts = { PENDIENTE: 0, EN_COCINA: 0, LISTO: 0 };
    document.querySelectorAll('.order-card').forEach(card => {
        const status = card.dataset.status || 
            (card.classList.contains('pending') ? 'PENDIENTE' :
             card.classList.contains('cooking') ? 'EN_COCINA' :
             card.classList.contains('ready') ? 'LISTO' : 'PENDIENTE');
        if (counts[status] !== undefined) counts[status]++;
    });
    Object.keys(counts).forEach(key => {
        if (statusCounts[key]) statusCounts[key].textContent = counts[key];
    });
    const total = counts.PENDIENTE + counts.EN_COCINA + counts.LISTO;
    orderCountEl.textContent = `Órdenes: ${total}`;
}

function setConnectionStatus(connected) {
    connectionStatus.className = 'status ' + (connected ? 'connected' : 'disconnected');
    connectionStatus.textContent = connected ? 'Conectado' : 'Desconectado';
}

document.addEventListener('DOMContentLoaded', init);