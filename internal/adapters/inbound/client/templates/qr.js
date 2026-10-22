const BASE_URL = window.location.origin;

function init() {
    renderQRCodes();
}

function renderQRCodes() {
    const grid = document.getElementById('qr-grid');
    const codes = [];

    for (let i = 1; i <= 10; i++) {
        const url = `${BASE_URL}/?mesa=${i}`;
        codes.push({ mesa: i, url });
    }

    grid.innerHTML = codes.map(code => `
        <article class="qr-card">
            <div class="qr-number">${code.mesa}</div>
            <div class="qr-label">Mesa ${code.mesa}</div>
            <div class="qr-image">
                <img src="https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(code.url)}" 
                     alt="QR Mesa ${code.mesa}" 
                     onerror="this.src='data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAwIiBoZWlnaHQ9IjIwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iMTAwJSIgaGVpZ2h0PSIxMDAlIiBmaWxsPSIjZTJlOGYwIi8+PHRleHQgeD0iNTAlIiB5PSI1MCUiIGZvbnQtZmFtaWx5PSJBcmlhbCIgZm9udC1zaXplPSIxNCIgZmlsbD0iIzk5OSIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZHk9Ii4zZW0iPkJSIENvZGU8L3RleHQ+PC9zdmc+'">
            </div>
            <div class="qr-url">${code.url}</div>
        </article>
    `).join('');
}

window.addEventListener('DOMContentLoaded', init);