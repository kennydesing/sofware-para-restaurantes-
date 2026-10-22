package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

type createInsumoRequest struct {
	Nombre      string  `json:"nombre"`
	UnidadBase  string  `json:"unidad_base"`
	StockMinimo float64 `json:"stock_minimo"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) handleInsumos(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Handling %s request for /api/insumos\n", r.Method)
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		insumos, err := s.inventario.ListarInsumos(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, insumos)

	case http.MethodPost:
		var req createInsumoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "JSON inválido")
			return
		}
		if req.Nombre == "" || req.UnidadBase == "" {
			writeError(w, http.StatusBadRequest, "nombre y unidad_base son obligatorios")
			return
		}
		insumo := &domain.Insumo{
			Nombre:      req.Nombre,
			UnidadBase:  req.UnidadBase,
			StockMinimo: req.StockMinimo,
		}
		if err := s.insumoRepo.Create(ctx, insumo); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, insumo)

	default:
		writeError(w, http.StatusMethodNotAllowed, "método no permitido")
	}
}

func (s *Server) handleInsumoByID(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Handling request for insumo ID from path: %s, method: %s\n", r.URL.Path, r.Method)
	idStr := strings.TrimPrefix(r.URL.Path, "/api/insumos/")
	parts := strings.Split(idStr, "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return
	}

	ctx := context.Background()

	// Check if the request is for the movimientos sub-resource
	if len(parts) > 1 && parts[1] == "movimientos" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		movimientos, err := s.movimientoRepo.GetByInsumoID(ctx, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, movimientos)
		return
	}

	// Handle the insumo resource itself
	switch r.Method {
	case http.MethodGet:
		insumo, err := s.inventario.ObtenerInsumo(ctx, id)
		if err != nil {
			writeError(w, http.StatusNotFound, "insumo no encontrado")
			return
		}
		writeJSON(w, http.StatusOK, insumo)
	case http.MethodPut:
		fmt.Printf("Handling PUT request for insumo %d\n", id)
		var req domain.Insumo
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "JSON inválido")
			return
		}
		// Validate required fields
		if req.Nombre == "" || req.UnidadBase == "" {
			writeError(w, http.StatusBadRequest, "nombre y unidad_base son obligatorios")
			return
		}
		if req.StockMinimo < 0 {
			writeError(w, http.StatusBadRequest, "stock_minimo debe ser >= 0")
			return
		}
		// Ensure the ID in the URL matches the ID in the body (if present)
		if req.ID != 0 && req.ID != id {
			writeError(w, http.StatusBadRequest, "ID de insumo no coincide")
			return
		}
		req.ID = id
		if err := s.insumoRepo.Update(ctx, &req); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, req)
	case http.MethodDelete:
		fmt.Printf("Handling DELETE request for insumo %d\n", id)
		if err := s.insumoRepo.Delete(ctx, id); err != nil {
			if err == domain.ErrInsumoNoEncontrado {
				writeError(w, http.StatusNotFound, "insumo no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "Insumo eliminado exitosamente"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "método no permitido")
	}
}

func (s *Server) handlePlatos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}
	ctx := context.Background()
	platos, err := s.platoRepo.GetAll(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, platos)
}

func (s *Server) handlePedidos(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodPost:
		var req ports.CrearPedidoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "JSON inválido")
			return
		}
		pedido, err := s.pedido.CrearPedido(ctx, req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, pedido)

	default:
		writeError(w, http.StatusMethodNotAllowed, "método no permitido")
	}
}

func (s *Server) handlePedidoByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/pedidos/")
	parts := strings.Split(idStr, "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID invalid")
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		pedido, err := s.pedido.ObtenerPedido(ctx, id)
		if err != nil {
			writeError(w, http.StatusNotFound, "pedido no encontrado")
			return
		}
		writeJSON(w, http.StatusOK, pedido)

	case http.MethodPost:
		if len(parts) > 1 && parts[1] == "procesar" {
			if err := s.pedido.ProcesarPedido(ctx, id); err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "procesado"})
			return
		}
		writeError(w, http.StatusNotFound, "endpoint no encontrado")

	default:
		writeError(w, http.StatusMethodNotAllowed, "método no permitido")
	}
}

func (s *Server) handleCompras(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}
	var req ports.CompraRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	ctx := context.Background()
	insumo, err := s.inventario.RegistrarCompra(ctx, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, insumo)
}

func (s *Server) handleMermas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}
	var req ports.MermaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	ctx := context.Background()
	mov, err := s.inventario.RegistrarMerma(ctx, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mov)
}

func (s *Server) handleAjustes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}
	var req ports.AjusteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	ctx := context.Background()
	mov, err := s.inventario.AjustarStock(ctx, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mov)
}