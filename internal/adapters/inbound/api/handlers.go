package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"pos-inventory/internal/adapters/inbound/shared"
	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

type createInsumoRequest struct {
	Nombre      string  `json:"nombre"`
	UnidadBase  string  `json:"unidad_base"`
	StockMinimo float64 `json:"stock_minimo"`
}

func (s *Server) handleInsumos(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		insumos, err := s.inventario.ListarInsumos(ctx)
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusOK, insumos)

	case http.MethodPost:
		var req createInsumoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
			return
		}
		if req.Nombre == "" || req.UnidadBase == "" {
			shared.WriteError(w, http.StatusBadRequest, "nombre y unidad_base son obligatorios")
			return
		}
		insumo := &domain.Insumo{
			Nombre:      req.Nombre,
			UnidadBase:  req.UnidadBase,
			StockMinimo: req.StockMinimo,
		}
		if err := s.insumoRepo.Create(ctx, insumo); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusCreated, insumo)

	default:
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
	}
}

func (s *Server) handleInsumoByID(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(r.URL.Path, "/insumos/")
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "ID inválido")
		return
	}

	ctx := context.Background()

	if strings.Contains(r.URL.Path, "/movimientos") {
		movimientos, err := s.movimientoRepo.GetByInsumoID(ctx, id)
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusOK, movimientos)
		return
	}

	insumo, err := s.inventario.ObtenerInsumo(ctx, id)
	if err != nil {
		shared.WriteError(w, http.StatusNotFound, "insumo no encontrado")
		return
	}
	shared.WriteJSON(w, http.StatusOK, insumo)
}

func (s *Server) handlePlatos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}
	ctx := context.Background()
	platos, err := s.platoRepo.GetAll(ctx)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, platos)
}

func (s *Server) handlePedidos(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodPost:
		var req ports.CrearPedidoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
			return
		}
		pedido, err := s.pedido.CrearPedido(ctx, req)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		shared.WriteJSON(w, http.StatusCreated, pedido)

	default:
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
	}
}

func (s *Server) handlePedidoByID(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(r.URL.Path, "/pedidos/")
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "ID inválido")
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		pedido, err := s.pedido.ObtenerPedido(ctx, id)
		if err != nil {
			shared.WriteError(w, http.StatusNotFound, "pedido no encontrado")
			return
		}
		shared.WriteJSON(w, http.StatusOK, pedido)

	case http.MethodPost:
		if strings.Contains(r.URL.Path, "/procesar") {
			if err := s.pedido.ProcesarPedido(ctx, id); err != nil {
				shared.WriteError(w, http.StatusConflict, err.Error())
				return
			}
			shared.WriteJSON(w, http.StatusOK, map[string]string{"status": "procesado"})
			return
		}
		if strings.Contains(r.URL.Path, "/estado") {
			var req struct {
				Estado string `json:"estado"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
				return
			}
			if err := s.pedido.ActualizarEstadoPedido(ctx, id, req.Estado); err != nil {
				shared.WriteError(w, http.StatusConflict, err.Error())
				return
			}
			if s.notifier != nil {
				_ = s.notifier.NotificarCambioEstado(ctx, id, req.Estado)
			}
			shared.WriteJSON(w, http.StatusOK, map[string]string{"status": "actualizado"})
			return
		}
		shared.WriteError(w, http.StatusNotFound, "endpoint no encontrado")

	default:
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
	}
}

func (s *Server) handleCompras(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}
	var req ports.CompraRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	ctx := context.Background()
	insumo, err := s.inventario.RegistrarCompra(ctx, req)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, insumo)
}

func (s *Server) handleMermas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}
	var req ports.MermaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	ctx := context.Background()
	mov, err := s.inventario.RegistrarMerma(ctx, req)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, mov)
}

func (s *Server) handleAjustes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}
	var req ports.AjusteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	ctx := context.Background()
	mov, err := s.inventario.AjustarStock(ctx, req)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, mov)
}