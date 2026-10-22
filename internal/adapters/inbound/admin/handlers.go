package admin

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"pos-inventory/internal/adapters/inbound/shared"
	"pos-inventory/internal/adapters/inbound/shared/middleware"
	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

//go:embed templates/* static/*
var templateFS embed.FS

var adminTemplate = template.Must(template.New("admin.html").Funcs(template.FuncMap{
	"csrfField": middleware.CSRFField,
}).ParseFS(templateFS, "templates/admin.html"))

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "" && r.URL.Path != "/admin" && r.URL.Path != "/admin/" {
		shared.WriteError(w, http.StatusNotFound, "Página no encontrada")
		return
	}
	
	usuario := middleware.UsuarioFromContext(r.Context())
	csrfToken := middleware.CSRFToken(r)
	
	data := map[string]interface{}{
		"UsuarioActual": usuario,
		"CSRFToken":     csrfToken,
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := adminTemplate.Execute(w, data); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Error al renderizar plantilla: "+err.Error())
	}
}

func (s *Server) handleStatsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	ctx := context.Background()

	insumos, err := s.inventarioSvc.ListarInsumos(ctx)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lowStockCount := 0
	for _, ins := range insumos {
		if ins.Stock <= ins.StockMinimo {
			lowStockCount++
		}
	}

	platos, err := s.platoRepo.GetAll(ctx)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pedidos, err := s.pedidoSvc.ListarPedidos(ctx)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var totalVentas float64
	for _, p := range pedidos {
		if p.Estado == string(domain.EstadoListo) || p.Estado == string(domain.EstadoEntregado) || p.Estado == string(domain.EstadoEnCocina) {
			totalVentas += p.Total
		}
	}

	shared.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"total_insumos":   len(insumos),
		"stock_bajo":      lowStockCount,
		"total_platos":    len(platos),
		"total_pedidos":   len(pedidos),
		"total_ventas_bs": totalVentas,
	})
}

func (s *Server) handleInsumosAPI(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		insumos, err := s.insumoRepo.GetAllEnriquecido(ctx)
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// InsumoResponseDTO es el DTO de salida para la vista admin (Spec 004 y 005).
		// Extiende los datos de dominio con campos calculados sin modificar el core.
		type InsumoResponseDTO struct {
			ID                        int     `json:"id"`
			Nombre                    string  `json:"nombre"`
			UnidadBase                string  `json:"unidad_base"`
			CostoPromedio             float64 `json:"costo_promedio"`
			PrecioVentaUSD            float64 `json:"precio_venta_usd"`
			StockMinimo               float64 `json:"stock_minimo"`
			Stock                     float64 `json:"stock"`
			MermaAcumulada            float64 `json:"merma_acumulada"`
			PresentacionNombre        string  `json:"presentacion_nombre"`
			CantidadBasePorPresentacion float64 `json:"cantidad_base_por_presentacion"`
			UnidadCompraAbrev          string  `json:"unidad_compra_abrev"`
			StockMinimoEnCompra        float64 `json:"stock_minimo_en_compra"`
			EmpaqueTipo                string  `json:"empaque_tipo"`
			EmpaqueNombre              string  `json:"empaque_nombre"`
			UnidadesPorEmpaque         int     `json:"unidades_por_empaque"`
			ContenidoMlPorUnidad       float64 `json:"contenido_ml_por_unidad"`
			EsRetornable               bool    `json:"es_retornable"`
			CantidadEnPresentacion     float64 `json:"cantidad_en_presentacion"`
			PesoValor                  float64 `json:"peso_valor"`
			PesoUnidad                 string  `json:"peso_unidad"`
			StockMinimoEnUnidades      float64 `json:"stock_minimo_en_unidades"`
			EsEmpaqueCompuesto         bool    `json:"es_empaque_compuesto"`
			UnidadesTotalesStock       float64 `json:"unidades_totales_stock"`
		}
		result := make([]InsumoResponseDTO, 0, len(insumos))
		for _, ins := range insumos {
			pesoValor, pesoUnidad := domain.FormatearPeso(ins.Stock, ins.UnidadBase)
			esEmpaqueCompuesto := ins.UnidadesPorEmpaque > 1 && ins.ContenidoMlPorUnidad > 0
			stockMinimoEnUnidades := ins.StockMinimo
			unidadesTotalesStock := ins.Stock
			if esEmpaqueCompuesto {
				stockMinimoEnUnidades = ins.StockMinimo / ins.ContenidoMlPorUnidad
				unidadesTotalesStock = ins.Stock / ins.ContenidoMlPorUnidad
			}
			result = append(result, InsumoResponseDTO{
				ID:                         ins.ID,
				Nombre:                     ins.Nombre,
				UnidadBase:                 ins.UnidadBase,
				CostoPromedio:              ins.CostoPromedio,
				PrecioVentaUSD:             ins.PrecioVentaUSD,
				StockMinimo:                ins.StockMinimo,
				Stock:                      ins.Stock,
				MermaAcumulada:             ins.MermaAcumulada,
				PresentacionNombre:         ins.PresentacionNombre,
				CantidadBasePorPresentacion: ins.CantidadBasePorPresentacion,
				UnidadCompraAbrev:          ins.UnidadCompraAbrev,
				StockMinimoEnCompra:        ins.StockMinimoEnCompra,
				EmpaqueTipo:                string(ins.EmpaqueTipo),
				EmpaqueNombre:              ins.EmpaqueNombre,
				UnidadesPorEmpaque:         ins.UnidadesPorEmpaque,
				ContenidoMlPorUnidad:       ins.ContenidoMlPorUnidad,
				EsRetornable:               ins.EsRetornable,
				CantidadEnPresentacion:     ins.CantidadEnPresentacion(),
				PesoValor:                  pesoValor,
				PesoUnidad:                 pesoUnidad,
				StockMinimoEnUnidades:      stockMinimoEnUnidades,
				EsEmpaqueCompuesto:         esEmpaqueCompuesto,
				UnidadesTotalesStock:       unidadesTotalesStock,
			})
		}
		shared.WriteJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var req struct {
			Nombre                      string  `json:"nombre"`
			UnidadBase                  string  `json:"unidad_base"`
			StockMinimo                 float64 `json:"stock_minimo"`
			PrecioVentaUSD              float64 `json:"precio_venta_usd"`
			PresentacionNombre          string  `json:"presentacion_nombre"`
			CantidadBasePorPresentacion float64 `json:"cantidad_base_por_presentacion"`
			EmpaqueTipo                 string  `json:"empaque_tipo"`
			EmpaqueNombre               string  `json:"empaque_nombre"`
			UnidadesPorEmpaque          int     `json:"unidades_por_empaque"`
			ContenidoMlPorUnidad        float64 `json:"contenido_ml_por_unidad"`
			EsRetornable                bool    `json:"es_retornable"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
			return
		}
		if req.Nombre == "" || req.UnidadBase == "" {
			shared.WriteError(w, http.StatusBadRequest, "El nombre y la unidad base son requeridos")
			return
		}

		insumo := &domain.Insumo{
			Nombre:                      req.Nombre,
			UnidadBase:                  req.UnidadBase,
			StockMinimo:                 req.StockMinimo,
			Stock:                       0,
			PrecioVentaUSD:              req.PrecioVentaUSD,
			PresentacionNombre:          req.PresentacionNombre,
			CantidadBasePorPresentacion: req.CantidadBasePorPresentacion,
			EmpaqueTipo:                 domain.TipoEmpaque(req.EmpaqueTipo),
			EmpaqueNombre:               req.EmpaqueNombre,
			UnidadesPorEmpaque:          req.UnidadesPorEmpaque,
			ContenidoMlPorUnidad:        req.ContenidoMlPorUnidad,
			EsRetornable:                req.EsRetornable,
		}
		insumo.NormalizarEmpaque()
		insumo.NormalizarPresentacion()
		if err := s.insumoRepo.Create(ctx, insumo); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusCreated, insumo)

	default:
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
	}
}

func (s *Server) handleComprasAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	var req ports.CompraRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return
	}
	ctx := context.Background()
	insumo, err := s.inventarioSvc.RegistrarCompra(ctx, req)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, insumo)
}

func (s *Server) handleMermasAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	var req ports.MermaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return
	}
	ctx := context.Background()
	mov, err := s.inventarioSvc.RegistrarMerma(ctx, req)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, mov)
}

func (s *Server) handleMermasHistorialAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	ctx := context.Background()
	historial, err := s.inventarioSvc.ListarHistorialMermas(ctx, 100)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, historial)
}

func (s *Server) handleUsoPersonalAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	var req ports.UsoPersonalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return
	}
	ctx := context.Background()
	mov, err := s.inventarioSvc.RegistrarUsoPersonal(ctx, req)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, mov)
}

func (s *Server) handleUsoPersonalHistorialAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	ctx := context.Background()
	historial, err := s.inventarioSvc.ListarHistorialUsoPersonal(ctx, 100)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, historial)
}

func (s *Server) handleAjustesAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	var req ports.AjusteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return
	}
	ctx := context.Background()
	mov, err := s.inventarioSvc.AjustarStock(ctx, req)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, mov)
}

func (s *Server) handleAuditoriaStockAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	var req struct {
		InsumoID    int     `json:"insumo_id"`
		StockFisico float64 `json:"stock_fisico"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	ctx := context.Background()
	dto, err := s.inventarioSvc.AuditarStock(ctx, req.InsumoID, req.StockFisico)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, dto)
}

type PlatoConEscandallo struct {
	ID          int                 `json:"id"`
	Nombre      string              `json:"nombre"`
	PrecioVenta float64             `json:"precio_venta"`
	Activo      bool                `json:"activo"`
	Receta      []domain.RecetaItem `json:"receta"`
}

type CrearPlatoConRecetaRequest struct {
	Nombre      string `json:"nombre"`
	PrecioVenta float64 `json:"precio_venta"`
	Receta      []struct {
		InsumoID          int     `json:"insumo_id"`
		CantidadRequerida float64 `json:"cantidad_requerida"`
		PorcentajeMerma   float64 `json:"porcentaje_merma"`
	} `json:"receta"`
}

func (s *Server) handlePlatosAPI(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		platos, err := s.platoRepo.GetAll(ctx)
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}

		var resultado []PlatoConEscandallo
		for _, p := range platos {
			receta, err := s.recetaRepo.GetByPlatoID(ctx, p.ID)
			if err != nil {
				receta = []domain.RecetaItem{}
			}
			resultado = append(resultado, PlatoConEscandallo{
				ID:          p.ID,
				Nombre:      p.Nombre,
				PrecioVenta: p.PrecioVenta,
				Activo:      p.Activo,
				Receta:      receta,
			})
		}
		shared.WriteJSON(w, http.StatusOK, resultado)

	case http.MethodPost:
		var req CrearPlatoConRecetaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
			return
		}
		if req.Nombre == "" || req.PrecioVenta <= 0 {
			shared.WriteError(w, http.StatusBadRequest, "El nombre y precio de venta válido son requeridos")
			return
		}

		plato := &domain.Plato{
			Nombre:      req.Nombre,
			PrecioVenta: req.PrecioVenta,
			Activo:      true,
		}

		if err := s.platoRepo.Create(ctx, plato); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, "Error al crear plato: "+err.Error())
			return
		}

		var recetaItems []domain.RecetaItem
		for _, rItem := range req.Receta {
			item := &domain.RecetaItem{
				PlatoID:           plato.ID,
				InsumoID:          rItem.InsumoID,
				CantidadRequerida: rItem.CantidadRequerida,
				PorcentajeMerma:   rItem.PorcentajeMerma,
			}
			if err := s.recetaRepo.Create(ctx, item); err != nil {
				shared.WriteError(w, http.StatusInternalServerError, "Error al guardar ingrediente de receta: "+err.Error())
				return
			}
			recetaItems = append(recetaItems, *item)
		}

		shared.WriteJSON(w, http.StatusCreated, PlatoConEscandallo{
			ID:          plato.ID,
			Nombre:      plato.Nombre,
			PrecioVenta: plato.PrecioVenta,
			Activo:      plato.Activo,
			Receta:      recetaItems,
		})

	default:
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
	}
}

func (s *Server) handlePedidosAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	ctx := context.Background()
	pedidos, err := s.pedidoSvc.ListarPedidos(ctx)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, pedidos)
}

func (s *Server) handleAuditoriaMovimientosAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	ctx := context.Background()

	insumoIDStr := r.URL.Query().Get("insumo_id")
	if insumoIDStr != "" {
		insumoID, err := strconv.Atoi(insumoIDStr)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, "ID de insumo inválido")
			return
		}
		movs, err := s.movimientoRepo.GetByInsumoID(ctx, insumoID)
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusOK, movs)
		return
	}

	insumos, err := s.inventarioSvc.ListarInsumos(ctx)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var todosMovimientos []*domain.MovimientoInventario
	for _, ins := range insumos {
		movs, err := s.movimientoRepo.GetByInsumoID(ctx, ins.ID)
		if err == nil {
			todosMovimientos = append(todosMovimientos, movs...)
		}
	}

	shared.WriteJSON(w, http.StatusOK, todosMovimientos)
}

func (s *Server) handleUsuariosAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		usuarios, err := s.authSvc.ListarUsuarios(ctx)
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusOK, usuarios)

	case http.MethodPost:
		var req ports.CrearUsuarioRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "Cuerpo de solicitud inválido: "+err.Error())
			return
		}

		usuario, err := s.authSvc.CrearUsuario(ctx, req)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		shared.WriteJSON(w, http.StatusCreated, usuario)

	default:
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
	}
}

type modificarUsuarioPayload struct {
	ID     int         `json:"id"`
	Rol    *domain.Rol `json:"rol,omitempty"`
	Activo *bool       `json:"activo,omitempty"`
}

func (s *Server) handleModificarUsuarioAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}

	var payload modificarUsuarioPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Cuerpo de solicitud inválido: "+err.Error())
		return
	}

	if payload.ID <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "ID de usuario inválido")
		return
	}

	usuario, err := s.authSvc.ModificarUsuario(r.Context(), payload.ID, ports.ModificarUsuarioRequest{
		Rol:    payload.Rol,
		Activo: payload.Activo,
	})
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	shared.WriteJSON(w, http.StatusOK, usuario)
}

func (s *Server) handlePerfilAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}

	u := middleware.UsuarioFromContext(r.Context())
	if u != nil {
		shared.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"id":     u.ID,
			"nombre": u.Nombre,
			"email":  u.Email,
			"rol":    u.Rol,
			"activo": u.Activo,
		})
		return
	}

	shared.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"id":     1,
		"nombre": "Administrador General",
		"email":  "admin@restaurante.com",
		"rol":    "admin",
		"activo": true,
	})
}

func (s *Server) handleInsumoByIDAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/insumos/")
	path = strings.TrimPrefix(path, "/")

	if strings.Contains(path, "unidades-compra") {
		parts := strings.Split(path, "/")
		id, err := strconv.Atoi(parts[0])
		if err != nil || id <= 0 {
			shared.WriteError(w, http.StatusBadRequest, "ID inválido")
			return
		}
		s.handleInsumoUnidadesCompraAPI(w, r, int64(id))
		return
	}

	idStr := path
	if i := strings.Index(idStr, "/"); i != -1 {
		idStr = idStr[:i]
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "ID inválido")
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		insumo, err := s.insumoRepo.GetByID(ctx, id)
		if err != nil {
			shared.WriteError(w, http.StatusNotFound, "Insumo no encontrado")
			return
		}
		shared.WriteJSON(w, http.StatusOK, insumo)

	case http.MethodPut:
		var req struct {
			Nombre                      string  `json:"nombre"`
			UnidadBase                  string  `json:"unidad_base"`
			StockMinimo                 float64 `json:"stock_minimo"`
			Stock                       float64 `json:"stock"`
			CostoPromedio               float64 `json:"costo_promedio"`
			PrecioVentaUSD              float64 `json:"precio_venta_usd"`
			PresentacionNombre          string  `json:"presentacion_nombre"`
			CantidadBasePorPresentacion float64 `json:"cantidad_base_por_presentacion"`
			EmpaqueTipo                 string  `json:"empaque_tipo"`
			EmpaqueNombre               string  `json:"empaque_nombre"`
			UnidadesPorEmpaque          int     `json:"unidades_por_empaque"`
			ContenidoMlPorUnidad        float64 `json:"contenido_ml_por_unidad"`
			EsRetornable                bool    `json:"es_retornable"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
			return
		}
		if req.Nombre == "" || req.UnidadBase == "" {
			shared.WriteError(w, http.StatusBadRequest, "Nombre y unidad_base son requeridos")
			return
		}
		if req.StockMinimo < 0 {
			shared.WriteError(w, http.StatusBadRequest, "El stock mínimo no puede ser negativo")
			return
		}

		actual, err := s.insumoRepo.GetByID(ctx, id)
		if err != nil {
			shared.WriteError(w, http.StatusNotFound, "Insumo no encontrado")
			return
		}

		actual.Nombre = req.Nombre
		actual.UnidadBase = req.UnidadBase
		actual.StockMinimo = req.StockMinimo
		actual.Stock = req.Stock
		actual.CostoPromedio = req.CostoPromedio
		actual.PrecioVentaUSD = req.PrecioVentaUSD
		if req.PresentacionNombre != "" {
			actual.PresentacionNombre = req.PresentacionNombre
		}
		if req.CantidadBasePorPresentacion > 0 {
			actual.CantidadBasePorPresentacion = req.CantidadBasePorPresentacion
		}
		if req.EmpaqueTipo != "" {
			actual.EmpaqueTipo = domain.TipoEmpaque(req.EmpaqueTipo)
		}
		if req.UnidadesPorEmpaque >= 1 {
			actual.UnidadesPorEmpaque = req.UnidadesPorEmpaque
		}
		if req.EmpaqueNombre != "" {
			actual.EmpaqueNombre = req.EmpaqueNombre
		}
		if req.ContenidoMlPorUnidad > 0 {
			actual.ContenidoMlPorUnidad = req.ContenidoMlPorUnidad
		}
		actual.EsRetornable = req.EsRetornable
		actual.NormalizarEmpaque()
		actual.NormalizarPresentacion()

		if err := s.insumoRepo.Update(ctx, actual); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusOK, actual)

	case http.MethodDelete:
		if err := s.insumoRepo.Delete(ctx, id); err != nil {
			if err == domain.ErrInsumoNoEncontrado {
				shared.WriteError(w, http.StatusNotFound, "Insumo no encontrado")
				return
			}
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
	}
}

// ---- Tasas Oficiales de Cambio (BCV) ----

func (s *Server) handleTasasAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	ctx := context.Background()
	tasas, err := s.tasaSvc.ObtenerTasas(ctx)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, tasas)
}

func (s *Server) handleSincronizarTasasAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	ctx := context.Background()
	tasas, err := s.tasaSvc.SincronizarTasasBCV(ctx)
	if err != nil {
		shared.WriteError(w, http.StatusBadGateway, fmt.Sprintf("Error conectando con BCV: %v", err))
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"mensaje": "Tasas sincronizadas exitosamente con el BCV",
		"tasas":   tasas,
	})
}

func (s *Server) handleActualizarTasaManualAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}
	var req struct {
		Moneda string  `json:"moneda"`
		Tasa   float64 `json:"tasa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if req.Tasa <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "La tasa debe ser mayor a 0")
		return
	}

	ctx := context.Background()
	if err := s.tasaSvc.ActualizarTasaManual(ctx, req.Moneda, req.Tasa); err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	tasas, _ := s.tasaSvc.ObtenerTasas(ctx)
	shared.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"mensaje": fmt.Sprintf("Tasa de %s actualizada manualmente", req.Moneda),
		"tasas":   tasas,
	})
}

// handleInsumoUnidadesCompraAPI gestiona las presentaciones de compra específicas de un insumo (Spec 003 T-12).
func (s *Server) handleInsumoUnidadesCompraAPI(w http.ResponseWriter, r *http.Request, insumoID int64) {
	ctx := context.Background()
	switch r.Method {
	case http.MethodGet:
		unidades, err := s.unidadRepo.ObtenerUnidadesCompraInsumo(ctx, insumoID)
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"data":  unidades,
			"error": nil,
		})

	case http.MethodPost:
		var req struct {
			UnidadID    int64  `json:"unidad_id"`
			FactorN     int64  `json:"factor_n"`
			FactorD     int64  `json:"factor_d"`
			EsPrincipal bool   `json:"es_principal"`
			Descripcion string `json:"descripcion"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
			return
		}
		if req.UnidadID <= 0 || req.FactorN <= 0 || req.FactorD <= 0 {
			shared.WriteError(w, http.StatusBadRequest, "unidad_id, factor_n y factor_d deben ser mayores a 0")
			return
		}

		iuc := domain.InsumoUnidadCompra{
			InsumoID:    insumoID,
			UnidadID:    req.UnidadID,
			FactorN:     req.FactorN,
			FactorD:     req.FactorD,
			EsPrincipal: req.EsPrincipal,
			Description: req.Descripcion,
		}
		if err := s.unidadRepo.GuardarUnidadCompraInsumo(ctx, iuc); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusCreated, map[string]interface{}{
			"data":  iuc,
			"error": nil,
		})

	default:
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
	}
}

// handleUnidadesAPI gestiona el catálogo de unidades de medida (Spec 003 T-12).
// GET: lista todas las unidades o filtra por categoría (?categoria=PESO).
// POST: registra una nueva unidad personalizada.
func (s *Server) handleUnidadesAPI(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		cat := r.URL.Query().Get("categoria")
		var unidades []domain.UnidadMedida
		var err error

		if cat != "" {
			catUpper := domain.Categoria(strings.ToUpper(cat))
			unidades, err = s.unidadRepo.ObtenerPorCategoria(ctx, catUpper)
		} else {
			unidades, err = s.unidadRepo.ObtenerTodas(ctx)
		}
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}

		type unidadLite struct {
			ID          int64  `json:"id"`
			Nombre      string `json:"nombre"`
			Abreviatura string `json:"abreviatura"`
			Categoria   string `json:"categoria"`
			EsBase      bool   `json:"es_base"`
			FactorN     int64  `json:"factor_n"`
			FactorD     int64  `json:"factor_d"`
		}
		result := make([]unidadLite, 0, len(unidades))
		for _, u := range unidades {
			result = append(result, unidadLite{
				ID:          u.ID,
				Nombre:      u.Nombre,
				Abreviatura: u.Abreviatura,
				Categoria:   string(u.Categoria),
				EsBase:      u.EsBase,
				FactorN:     u.FactorN,
				FactorD:     u.FactorD,
			})
		}
		shared.WriteJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var req struct {
			Nombre      string `json:"nombre"`
			Abreviatura string `json:"abreviatura"`
			Categoria   string `json:"categoria"`
			FactorN     int64  `json:"factor_n"`
			FactorD     int64  `json:"factor_d"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
			return
		}

		req.Nombre = strings.TrimSpace(req.Nombre)
		req.Abreviatura = strings.TrimSpace(req.Abreviatura)
		catUpper := domain.Categoria(strings.ToUpper(strings.TrimSpace(req.Categoria)))

		if req.Nombre == "" || req.Abreviatura == "" {
			shared.WriteError(w, http.StatusBadRequest, "Nombre y abreviatura son requeridos")
			return
		}
		if catUpper != domain.CategoriaVolumen && catUpper != domain.CategoriaPeso &&
			catUpper != domain.CategoriaConteo && catUpper != domain.CategoriaCulinaria {
			shared.WriteError(w, http.StatusBadRequest, "Categoría inválida")
			return
		}
		if req.FactorN <= 0 || req.FactorD <= 0 {
			shared.WriteError(w, http.StatusBadRequest, "factor_n y factor_d deben ser mayores a 0")
			return
		}

		nuevaUnidad := domain.UnidadMedida{
			Nombre:      req.Nombre,
			Abreviatura: req.Abreviatura,
			Categoria:   catUpper,
			EsBase:      false,
			FactorN:     req.FactorN,
			FactorD:     req.FactorD,
		}

		id, err := s.unidadRepo.Crear(ctx, nuevaUnidad)
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		nuevaUnidad.ID = id

		shared.WriteJSON(w, http.StatusCreated, map[string]interface{}{
			"data":  nuevaUnidad,
			"error": nil,
		})

	default:
		shared.WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
	}
}


