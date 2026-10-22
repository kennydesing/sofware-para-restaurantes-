package kitchen

import (
	"io/fs"
	"net/http"

	"pos-inventory/internal/core/ports"
)

type Server struct {
	mux               *http.ServeMux
	store             *OrdersStore
	printer           ports.PrinterRepository
	notifier          ports.NotifierRepository
	pedidoSvc         ports.PedidoService
	staticFileServer  http.Handler
}

func NewServer(pedidoSvc ports.PedidoService, printer ports.PrinterRepository, notifier ports.NotifierRepository) *Server {
	store := NewOrdersStore(pedidoSvc)
	s := &Server{
		mux:       http.NewServeMux(),
		store:     store,
		printer:   printer,
		notifier:  notifier,
		pedidoSvc: pedidoSvc,
	}
	s.registerRoutes()
	go s.loadInitialOrders()
	return s
}

func (s *Server) registerRoutes() {
	staticDir, err := fs.Sub(templateFS, "static")
	if err != nil {
		panic("error al acceder a archivos estáticos de kitchen: " + err.Error())
	}

	s.staticFileServer = http.FileServer(http.FS(staticDir))
	s.mux.Handle("/static/", http.StripPrefix("/static/", s.staticFileServer))
	s.mux.HandleFunc("/", s.handleKDS)
	s.mux.HandleFunc("/api/orders", s.handleAPIOrders)
	s.mux.HandleFunc("/api/pedidos/", s.handlePedidoEstado)
	s.mux.HandleFunc("/ws", s.handleSSE)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) StaticFileServer() http.Handler {
	return s.staticFileServer
}