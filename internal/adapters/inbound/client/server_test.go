package client_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pos-inventory/internal/adapters/inbound/client"
)

func TestClientServer_StaticAndMenu(t *testing.T) {
	server := client.NewServer(nil, nil, nil)

	mux := http.NewServeMux()
	mux.Handle("/static/client/", http.StripPrefix("/static/client/", server.StaticFileServer()))
	mux.Handle("/", server)

	// 1. Probar carga de menú con mesa=1
	t.Run("GET /?mesa=1 carga template con link a CSS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?mesa=1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("se esperaba status 200, se obtuvo %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `<link rel="stylesheet" href="/static/client/menu.css">`) {
			t.Errorf("el HTML no contiene la referencia a /static/client/menu.css")
		}
		if !strings.Contains(body, "Mesa 1") {
			t.Errorf("el HTML no contiene la interpolación de Mesa 1")
		}
	})

	// 2. Probar que el archivo CSS de cliente se sirve correctamente y no retorna 404
	t.Run("GET /static/client/menu.css retorna 200 OK y CSS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/client/menu.css", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("se esperaba status 200 para menu.css, se obtuvo %d (posible 404 de prefijo)", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/css") {
			t.Errorf("Content-Type esperado text/css, obtenido %s", contentType)
		}

		if !strings.Contains(rec.Body.String(), ".menu-header") {
			t.Errorf("el contenido CSS no contiene reglas esperadas de menu.css")
		}
	})

	// 3. Probar que el archivo JS de cliente se sirve correctamente
	t.Run("GET /static/client/menu.js retorna 200 OK y JS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/client/menu.js", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("se esperaba status 200 para menu.js, se obtuvo %d", rec.Code)
		}

		if !strings.Contains(rec.Body.String(), "init") {
			t.Errorf("el contenido JS no contiene funciones esperadas")
		}
	})
}
