package client

import (
	"context"
	"encoding/json"
	"sync"

	"pos-inventory/internal/core/ports"
)

type ClientOrderEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type ClientStore struct {
	mu          sync.RWMutex
	orders      map[int]*ClientOrder
	clients     map[int]map[chan ClientOrderEvent]bool
	pedidoSvc   ports.PedidoService
}

type ClientOrder struct {
	ID        int                    `json:"id"`
	MesaID    int                    `json:"mesa_id"`
	Estado    string                 `json:"estado"`
	Items     []ClientOrderItem      `json:"items"`
	Total     float64                `json:"total"`
	CreatedAt string                 `json:"created_at"`
}

type ClientOrderItem struct {
	PlatoID  int     `json:"plato_id"`
	Nombre   string  `json:"nombre"`
	Cantidad int     `json:"cantidad"`
	Precio   float64 `json:"precio"`
}

func NewClientStore(pedidoSvc ports.PedidoService) *ClientStore {
	return &ClientStore{
		orders:    make(map[int]*ClientOrder),
		clients:   make(map[int]map[chan ClientOrderEvent]bool),
		pedidoSvc: pedidoSvc,
	}
}

func (s *ClientStore) AddClient(mesaID int, ch chan ClientOrderEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clients[mesaID] == nil {
		s.clients[mesaID] = make(map[chan ClientOrderEvent]bool)
	}
	s.clients[mesaID][ch] = true
}

func (s *ClientStore) RemoveClient(mesaID int, ch chan ClientOrderEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if clients, ok := s.clients[mesaID]; ok {
		delete(clients, ch)
		close(ch)
		if len(clients) == 0 {
			delete(s.clients, mesaID)
		}
	}
}

func (s *ClientStore) broadcast(mesaID int, event ClientOrderEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if clients, ok := s.clients[mesaID]; ok {
		for ch := range clients {
			select {
			case ch <- event:
			default:
			}
		}
	}
}

func (s *ClientStore) LoadExistingOrders(ctx context.Context) error {
	pedidos, err := s.pedidoSvc.ListarPedidos(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range pedidos {
		s.orders[p.ID] = s.toClientOrder(p)
	}
	return nil
}

func (s *ClientStore) toClientOrder(p *ports.PedidoWithItems) *ClientOrder {
	items := make([]ClientOrderItem, len(p.Items))
	for i, item := range p.Items {
		items[i] = ClientOrderItem{
			PlatoID:  item.PlatoID,
			Nombre:   item.Nombre,
			Cantidad: item.Cantidad,
			Precio:   item.Precio,
		}
	}
	return &ClientOrder{
		ID:        p.ID,
		MesaID:    p.MesaID,
		Estado:    p.Estado,
		Items:     items,
		Total:     p.Total,
		CreatedAt: p.CreatedAt,
	}
}

func (s *ClientStore) AddOrder(order *ClientOrder) {
	s.mu.Lock()
	s.orders[order.ID] = order
	s.mu.Unlock()
	s.broadcast(order.MesaID, ClientOrderEvent{Type: "new_order", Data: order})
}

func (s *ClientStore) UpdateOrderStatus(id int, estado string, mesaID int) {
	s.mu.Lock()
	if order, ok := s.orders[id]; ok {
		order.Estado = estado
		s.broadcast(mesaID, ClientOrderEvent{Type: "status_change", Data: map[string]interface{}{
			"id":     id,
			"estado": estado,
		}})
	}
	s.mu.Unlock()
}

func (s *ClientStore) GetOrdersByMesa(mesaID int) []*ClientOrder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*ClientOrder
	for _, o := range s.orders {
		if o.MesaID == mesaID {
			result = append(result, o)
		}
	}
	return result
}

func (s *ClientStore) GetOrder(id int) (*ClientOrder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	return order, ok
}

func (s *ClientStore) ToJSON(mesaID int) ([]byte, error) {
	orders := s.GetOrdersByMesa(mesaID)
	return json.Marshal(orders)
}