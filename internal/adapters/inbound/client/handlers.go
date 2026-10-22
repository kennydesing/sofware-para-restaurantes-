package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"pos-inventory/internal/adapters/inbound/shared"
	"pos-inventory/internal/core/ports"
)

func (s *Server) loadInitialOrders() {
	if s.pedidoSvc == nil {
		return
	}
	ctx := context.Background()
	if err := s.store.LoadExistingOrders(ctx); err != nil {
	}
}

func (s *Server) handleMenu(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		shared.WriteError(w, http.StatusNotFound, "no encontrado")
		return
	}

	mesaStr := r.URL.Query().Get("mesa")
	if mesaStr == "" {
		http.Redirect(w, r, "/qr", http.StatusSeeOther)
		return
	}

	mesaID, err := strconv.Atoi(mesaStr)
	if err != nil || mesaID < 1 || mesaID > 10 {
		shared.WriteError(w, http.StatusBadRequest, "mesa inválida (1-10)")
		return
	}

	content, err := templateFS.ReadFile("templates/menu.html")
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "error cargando template")
		return
	}

	html := strings.ReplaceAll(string(content), "{{.MesaID}}", strconv.Itoa(mesaID))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (s *Server) handleCrearPedido(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		shared.WriteError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}

	var req ports.CrearPedidoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	if req.MesaID < 1 || req.MesaID > 10 {
		shared.WriteError(w, http.StatusBadRequest, "mesa_id debe estar entre 1 y 10")
		return
	}

	ctx := context.Background()
	pedido, err := s.pedidoSvc.CrearPedido(ctx, req)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.pedidoSvc.ProcesarPedido(ctx, pedido.ID); err != nil {
		shared.WriteError(w, http.StatusConflict, err.Error())
		return
	}

	order := &ClientOrder{
		ID:        pedido.ID,
		MesaID:    pedido.MesaID,
		Estado:    string(pedido.Estado),
		Items:     []ClientOrderItem{},
		Total:     pedido.Total,
		CreatedAt: pedido.CreatedAt.Format("2006-01-02T15:04:05"),
	}
	for _, d := range pedido.Detalles {
		order.Items = append(order.Items, ClientOrderItem{
			PlatoID:  d.PlatoID,
			Nombre:   d.NombrePlato,
			Cantidad: d.Cantidad,
			Precio:   d.PrecioUnitario,
		})
	}
	s.store.AddOrder(order)

	shared.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":     pedido.ID,
		"mesa":   pedido.MesaID,
		"estado": string(pedido.Estado),
		"total":  pedido.Total,
	})
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	mesaStr := r.URL.Query().Get("mesa")
	if mesaStr == "" {
		shared.WriteError(w, http.StatusBadRequest, "parámetro mesa requerido")
		return
	}

	mesaID, err := strconv.Atoi(mesaStr)
	if err != nil || mesaID < 1 || mesaID > 10 {
		shared.WriteError(w, http.StatusBadRequest, "mesa inválida (1-10)")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE no soportado", http.StatusInternalServerError)
		return
	}

	ch := make(chan ClientOrderEvent, 10)
	s.store.AddClient(mesaID, ch)
	defer s.store.RemoveClient(mesaID, ch)

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

func (s *Server) handleQR(w http.ResponseWriter, r *http.Request) {
	content, err := templateFS.ReadFile("templates/qr.html")
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "error cargando template")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	pedidoStr := r.URL.Query().Get("pedido")
	if pedidoStr == "" {
		shared.WriteError(w, http.StatusBadRequest, "parámetro pedido requerido")
		return
	}

	pedidoID, err := strconv.Atoi(pedidoStr)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "ID de pedido inválido")
		return
	}

	ctx := context.Background()
	pedido, err := s.pedidoSvc.ObtenerPedido(ctx, pedidoID)
	if err != nil {
		shared.WriteError(w, http.StatusNotFound, "pedido no encontrado")
		return
	}

	content, err := templateFS.ReadFile("templates/status.html")
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "error cargando template")
		return
	}

	html := string(content)
	html = strings.ReplaceAll(html, "{{.PedidoID}}", strconv.Itoa(pedido.ID))
	html = strings.ReplaceAll(html, "{{.MesaID}}", strconv.Itoa(pedido.MesaID))
	html = strings.ReplaceAll(html, "{{.Estado}}", string(pedido.Estado))
	html = strings.ReplaceAll(html, "{{.Total}}", strconv.FormatFloat(pedido.Total, 'f', 2, 64))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}