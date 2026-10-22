const API = '/api';

const statusConfig = {
    PENDIENTE: { 
        icon: '⏳', 
        title: 'PENDIENTE', 
        desc: 'Tu pedido ha sido recibido y está en cola',
        step: 1
    },
    EN_COCINA: { 
        icon: '🔥', 
        title: 'EN COCINA', 
        desc: 'Los chefs están preparando tu pedido',
        step: 2
    },
    LISTO: { 
        icon: '✅', 
        title: 'LISTO', 
        desc: '¡Tu pedido está listo para ser servido!',
        step: 3
    },
    ENTREGADO: { 
        icon: '🎉', 
        title: 'ENTREGADO', 
        desc: '¡Disfruta tu comida!',
        step: 4
    },
    CANCELADO: { 
        icon: '❌', 
        title: 'CANCELADO', 
        desc: 'El pedido ha sido cancelado',
        step: 0
    }
};

async function init() {
    const urlParams = new URLSearchParams(window.location.search);
    const pedidoID = urlParams.get('pedido');
    const mesaID = urlParams.get('mesa');

    if (!pedidoID) {
        window.location.href = '/qr';
        return;
    }

    if (mesaID) {
        connectSSE(mesaID);
    }

    await loadOrderStatus(pedidoID);
    setInterval(() => loadOrderStatus(pedidoID), 10000);
}

async function loadOrderStatus(pedidoID) {
    try {
        const res = await fetch(`${API}/pedidos/${pedidoID}`);
        if (!res.ok) throw new Error('Pedido no encontrado');
        const pedido = await res.json();
        updateUI(pedido);
    } catch (err) {
        console.error('Error:', err);
    }
}

function updateUI(pedido) {
    const config = statusConfig[pedido.estado] || statusConfig.PENDIENTE;
    
    document.getElementById('status-icon').textContent = config.icon;
    document.getElementById('status-title').textContent = config.title;
    document.getElementById('status-desc').textContent = config.desc;
    document.getElementById('status-badge').textContent = pedido.estado;
    document.getElementById('status-badge').className = 'status-badge ' + pedido.estado;

    updateProgress(config.step);
}

function updateProgress(currentStep) {
    document.querySelectorAll('.progress-step').forEach((step, index) => {
        const stepNum = index + 1;
        step.classList.remove('active', 'completed');
        
        if (stepNum < currentStep) {
            step.classList.add('completed');
        } else if (stepNum === currentStep) {
            step.classList.add('active');
        }
    });
}

function connectSSE(mesaID) {
    const eventSource = new EventSource(`/ws/client?mesa=${mesaID}`);
    
    eventSource.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            if (data.type === 'status_change') {
                loadOrderStatus(data.data.id);
            }
        } catch (e) {
            console.error('SSE error:', e);
        }
    };

    eventSource.onerror = () => {
        setTimeout(() => connectSSE(mesaID), 5000);
    };
}

document.addEventListener('DOMContentLoaded', init);