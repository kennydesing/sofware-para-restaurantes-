package kitchen

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"pos-inventory/internal/adapters/inbound/shared"
)

//go:embed templates/* static/*
var templateFS embed.FS

func (s *Server) loadInitialOrders() {
	if s.pedidoSvc == nil {
		return
	}
	ctx := context.Background()
	if err := s.store.LoadExistingOrders(ctx); err != nil {
	}
}

func (s *Server) handleKDS(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "" && r.URL.Path != "/kds" && r.URL.Path != "/kds/" {
		shared.WriteError(w, http.StatusNotFound, "no encontrado")
		return
	}
	content, err := templateFS.ReadFile("templates/kds.html")
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "error cargando template")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

func (s *Server) handleAPIOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}
	data, err := s.store.ToJSON()
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *Server) handlePedidoEstado(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/pedidos/")
	parts := strings.Split(idStr, "/")
	if len(parts) < 2 || parts[1] != "estado" {
		shared.WriteError(w, http.StatusNotFound, "endpoint no encontrado")
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "ID inválido")
		return
	}

	var req struct {
		Estado string `json:"estado"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	ctx := context.Background()
	if err := s.pedidoSvc.ActualizarEstadoPedido(ctx, id, req.Estado); err != nil {
		shared.WriteError(w, http.StatusConflict, err.Error())
		return
	}

	s.store.UpdateOrderStatus(id, req.Estado)

	shared.WriteJSON(w, http.StatusOK, map[string]string{"status": "actualizado"})
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE no soportado", http.StatusInternalServerError)
		return
	}

	ch := make(chan OrderEvent, 10)
	s.store.AddClient(ch)
	defer s.store.RemoveClient(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			data, _ := json.Marshal(event)
			w.Write([]byte("data: "))
			w.Write(data)
			w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}