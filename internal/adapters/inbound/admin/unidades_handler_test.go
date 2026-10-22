package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"pos-inventory/internal/adapters/inbound/admin"
	"pos-inventory/internal/adapters/outbound/sqlite"
	"pos-inventory/internal/core/domain"
)

func setupTestServer(t *testing.T) (*admin.Server, *sqlite.DB) {
	t.Helper()
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("error creando DB en memoria: %v", err)
	}

	ctx := context.Background()

	// Cargar migraciones base y de unidades
	findMigration := func(name string) string {
		paths := []string{
			filepath.Join("..", "..", "..", "..", "migrations", name),
			filepath.Join("migrations", name),
		}
		for _, p := range paths {
			if content, err := os.ReadFile(p); err == nil {
				return string(content)
			}
		}
		t.Fatalf("no se encontró archivo de migración %s", name)
		return ""
	}

	for _, m := range []string{"000001_init_schema.up.sql", "000004_unidades_medida.up.sql"} {
		sql := findMigration(m)
		if err := db.RunMigrations(ctx, sql); err != nil {
			t.Fatalf("error ejecutando migración %s: %v", m, err)
		}
	}

	insumoRepo := sqlite.NewInsumoRepository(db)
	movimientoRepo := sqlite.NewMovimientoRepository(db)
	platoRepo := sqlite.NewPlatoRepository(db)
	recetaRepo := sqlite.NewRecetaRepository(db)
	unidadRepo := sqlite.NewUnidadMedidaRepositorySQLite(db)

	server := admin.NewServer(nil, nil, insumoRepo, movimientoRepo, platoRepo, recetaRepo, unidadRepo, nil, nil)
	return server, db
}

func TestUnidadesAPI_GET_Todas(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/unidades", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status esperado 200, obtenido %d", rec.Code)
	}

	var unidades []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &unidades); err != nil {
		t.Fatalf("error al parsear JSON: %v", err)
	}

	if len(unidades) < 18 {
		t.Errorf("se esperaban al menos 18 unidades estándar, obtenidas %d", len(unidades))
	}
}

func TestUnidadesAPI_GET_FiltrarPorCategoria(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/unidades?categoria=PESO", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status esperado 200, obtenido %d", rec.Code)
	}

	var unidades []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &unidades); err != nil {
		t.Fatalf("error al parsear JSON: %v", err)
	}

	if len(unidades) == 0 {
		t.Fatalf("no se obtuvieron unidades de categoría PESO")
	}

	for _, u := range unidades {
		if u["categoria"] != string(domain.CategoriaPeso) {
			t.Errorf("unidad con categoría inesperada: %v", u["categoria"])
		}
	}
}

func TestUnidadesAPI_POST_CrearValida(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	payload := map[string]interface{}{
		"nombre":      "Caja de 10 Kilogramos",
		"abreviatura": "caja10kg",
		"categoria":   "PESO",
		"factor_n":    10000,
		"factor_d":    1,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/unidades", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status esperado 201, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUnidadesAPI_POST_FactorDenominadorInvalido(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	payload := map[string]interface{}{
		"nombre":      "Unidad Inválida",
		"abreviatura": "inv1",
		"categoria":   "PESO",
		"factor_n":    1000,
		"factor_d":    0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/unidades", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status esperado 400 Bad Request, obtenido %d", rec.Code)
	}
}

func TestUnidadesAPI_POST_CategoriaInvalida(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	payload := map[string]interface{}{
		"nombre":      "Unidad Inválida",
		"abreviatura": "inv2",
		"categoria":   "INEXISTENTE",
		"factor_n":    1,
		"factor_d":    1,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/unidades", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status esperado 400 Bad Request, obtenido %d", rec.Code)
	}
}
