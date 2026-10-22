const API = '/api';
let dishes = [];
let cart = [];
let currentCategory = 'all';

const cartBtn = document.getElementById('cart-btn');
const cartSlide = document.getElementById('cart-slide');
const overlay = document.getElementById('overlay');
const cartCount = document.getElementById('cart-count');
const cartItems = document.getElementById('cart-items');
const cartTotal = document.getElementById('cart-total');
const checkoutBtn = document.getElementById('checkout-btn');
const dishesGrid = document.getElementById('dishes-grid');
const categoriesEl = document.getElementById('categories');
const toast = document.getElementById('toast');

const mesaID = new URLSearchParams(window.location.search).get('mesa');

async function init() {
    await loadDishes();
    renderCategories();
    renderDishes();
    updateCartUI();
}

async function loadDishes() {
    try {
        const res = await fetch(API + '/platos');
        if (!res.ok) throw new Error('Error cargando platos');
        dishes = await res.json();
    } catch (err) {
        console.error('Error:', err);
        showToast('Error cargando el menú', 'error');
    }
}

function renderCategories() {
    const cats = ['all', ...new Set(dishes.map(d => d.categoria || 'Principal'))];
    categoriesEl.innerHTML = cats.map(cat => `
        <button class="category-btn ${cat === currentCategory ? 'active' : ''}" 
                onclick="filterCategory('${cat}')">
            ${cat === 'all' ? 'Todos' : cat}
        </button>
    `).join('');
}

function filterCategory(cat) {
    currentCategory = cat;
    document.querySelectorAll('.category-btn').forEach(btn => {
        btn.classList.toggle('active', btn.textContent === (cat === 'all' ? 'Todos' : cat));
    });
    renderDishes();
}

function renderDishes() {
    const filtered = currentCategory === 'all' 
        ? dishes 
        : dishes.filter(d => (d.categoria || 'Principal') === currentCategory);
    
    if (filtered.length === 0) {
        dishesGrid.innerHTML = `
            <div class="empty-state" style="grid-column: 1 / -1;">
                <div class="empty-icon">🍽️</div>
                <p>No hay platos en esta categoría</p>
            </div>
        `;
        return;
    }

    dishesGrid.innerHTML = filtered.map(dish => `
        <article class="dish-card" onclick="addToCart(${dish.id}, '${escapeHtml(dish.nombre)}', ${dish.precio_venta})">
            <div class="dish-image">🍽️</div>
            <div class="dish-content">
                <h3 class="dish-name">${escapeHtml(dish.nombre)}</h3>
                <div class="dish-price">$${dish.precio_venta.toFixed(2)}</div>
            </div>
        </article>
    `).join('');
}

function addToCart(id, name, price) {
    const existing = cart.find(item => item.id === id);
    if (existing) {
        existing.qty++;
    } else {
        cart.push({ id, name, price, qty: 1 });
    }
    updateCartUI();
    showToast(`${name} agregado al carrito`, 'success');
}

function removeFromCart(id) {
    cart = cart.filter(item => item.id !== id);
    updateCartUI();
}

function changeQty(id, delta) {
    const item = cart.find(i => i.id === id);
    if (item) {
        item.qty += delta;
        if (item.qty <= 0) removeFromCart(id);
        updateCartUI();
    }
}

function updateCartUI() {
    const totalItems = cart.reduce((sum, item) => sum + item.qty, 0);
    const totalPrice = cart.reduce((sum, item) => sum + item.price * item.qty, 0);

    cartCount.textContent = totalItems;
    cartCount.style.display = totalItems > 0 ? 'flex' : 'none';
    
    cartTotal.textContent = `$${totalPrice.toFixed(2)}`;
    checkoutBtn.disabled = cart.length === 0;

    if (cart.length === 0) {
        cartItems.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">🛒</div>
                <p>Tu carrito está vacío</p>
            </div>
        `;
    } else {
        cartItems.innerHTML = cart.map(item => `
            <div class="cart-item">
                <div class="cart-item-info">
                    <div class="cart-item-name">${escapeHtml(item.name)}</div>
                    <div class="cart-item-price">$${item.price.toFixed(2)} c/u</div>
                </div>
                <div class="cart-item-qty">
                    <button class="qty-btn" onclick="changeQty(${item.id}, -1)">−</button>
                    <span class="qty-value">${item.qty}</span>
                    <button class="qty-btn" onclick="changeQty(${item.id}, 1)">+</button>
                </div>
            </div>
        `).join('');
    }
}

function toggleCart() {
    cartSlide.classList.toggle('open');
    overlay.classList.toggle('open');
    document.body.style.overflow = cartSlide.classList.contains('open') ? 'hidden' : '';
}

async function checkout() {
    if (cart.length === 0) return;

    checkoutBtn.disabled = true;
    checkoutBtn.textContent = 'Procesando...';

    const items = cart.map(item => ({
        plato_id: item.id,
        cantidad: item.qty
    }));

    try {
        const res = await fetch(API + '/pedidos', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ mesa_id: parseInt(mesaID), items })
        });

        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'Error al crear pedido');
        }

        const data = await res.json();
        showToast(`¡Pedido #${data.id} confirmado! Total: $${data.total.toFixed(2)}`, 'success');
        cart = [];
        updateCartUI();
        toggleCart();
        
        setTimeout(() => {
            window.location.href = `/status?pedido=${data.id}&mesa=${mesaID}`;
        }, 2000);

    } catch (err) {
        showToast('Error: ' + err.message, 'error');
        checkoutBtn.disabled = false;
        checkoutBtn.textContent = 'Confirmar Pedido';
    }
}

function showToast(message, type = 'success') {
    toast.textContent = message;
    toast.className = 'toast ' + type + ' show';
    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

document.addEventListener('DOMContentLoaded', init);