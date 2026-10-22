package kitchen

import (
	"context"
	"encoding/json"
	"sync"

	"pos-inventory/internal/core/ports"
)

type OrderEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type OrdersStore struct {
	mu       sync.RWMutex
	orders   map[int]*KitchenOrder
	clients  map[chan OrderEvent]bool
	pedidoSvc ports.PedidoService
}

type KitchenOrder struct {
	ID        int                    `json:"id"`
	MesaID    int                    `json:"mesa_id"`
	Estado    string                 `json:"estado"`
	Items     []KitchenOrderItem     `json:"items"`
	Total     float64                `json:"total"`
	CreatedAt string                 `json:"created_at"`
}

type KitchenOrderItem struct {
	PlatoID   int     `json:"plato_id"`
	Nombre    string  `json:"nombre"`
	Cantidad  int     `json:"cantidad"`
	Precio    float64 `json:"precio"`
}

func NewOrdersStore(pedidoSvc ports.PedidoService) *OrdersStore {
	return &OrdersStore{
		orders:    make(map[int]*KitchenOrder),
		clients:   make(map[chan OrderEvent]bool),
		pedidoSvc: pedidoSvc,
	}
}

func (s *OrdersStore) AddClient(ch chan OrderEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[ch] = true
}

func (s *OrdersStore) RemoveClient(ch chan OrderEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, ch)
	close(ch)
}

func (s *OrdersStore) broadcast(event OrderEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *OrdersStore) LoadExistingOrders(ctx context.Context) error {
	pedidos, err := s.pedidoSvc.ListarPedidos(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range pedidos {
		s.orders[p.ID] = s.toKitchenOrder(p)
	}
	return nil
}

func (s *OrdersStore) toKitchenOrder(p *ports.PedidoWithItems) *KitchenOrder {
	items := make([]KitchenOrderItem, len(p.Items))
	for i, item := range p.Items {
		items[i] = KitchenOrderItem{
			PlatoID:  item.PlatoID,
			Nombre:   item.Nombre,
			Cantidad: item.Cantidad,
			Precio:   item.Precio,
		}
	}
	return &KitchenOrder{
		ID:        p.ID,
		MesaID:    p.MesaID,
		Estado:    p.Estado,
		Items:     items,
		Total:     p.Total,
		CreatedAt: p.CreatedAt,
	}
}

func (s *OrdersStore) AddOrder(order *KitchenOrder) {
	s.mu.Lock()
	s.orders[order.ID] = order
	s.mu.Unlock()
	s.broadcast(OrderEvent{Type: "new_order", Data: order})
}

func (s *OrdersStore) UpdateOrderStatus(id int, estado string) {
	s.mu.Lock()
	if order, ok := s.orders[id]; ok {
		order.Estado = estado
		s.broadcast(OrderEvent{Type: "status_change", Data: map[string]interface{}{
			"id":     id,
			"estado": estado,
		}})
	}
	s.mu.Unlock()
}

func (s *OrdersStore) GetAllOrders() []*KitchenOrder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]*KitchenOrder, 0, len(s.orders))
	for _, o := range s.orders {
		orders = append(orders, o)
	}
	return orders
}

func (s *OrdersStore) GetOrder(id int) (*KitchenOrder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	return order, ok
}

func (s *OrdersStore) ToJSON() ([]byte, error) {
	orders := s.GetAllOrders()
	return json.Marshal(orders)
}