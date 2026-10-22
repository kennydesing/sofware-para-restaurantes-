package client

import (
	"embed"
	"io/fs"
	"net/http"

	"pos-inventory/internal/core/ports"
)

//go:embed templates/*
var templateFS embed.FS

type Server struct {
	mux        *http.ServeMux
	store      *ClientStore
	pedidoSvc  ports.PedidoService
	platoRepo  ports.PlatoRepository
	insumoRepo ports.InsumoRepository
}

func NewServer(pedidoSvc ports.PedidoService, platoRepo ports.PlatoRepository, insumoRepo ports.InsumoRepository) *Server {
	store := NewClientStore(pedidoSvc)
	s := &Server{
		mux:        http.NewServeMux(),
		store:      store,
		pedidoSvc:  pedidoSvc,
		platoRepo:  platoRepo,
		insumoRepo: insumoRepo,
	}
	s.registerRoutes()
	go s.loadInitialOrders()
	return s
}

func (s *Server) registerRoutes() {
	staticDir, err := fs.Sub(templateFS, "templates")
	if err != nil {
		panic("error al acceder a templates: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(staticDir))
	s.mux.Handle("/static/client/", http.StripPrefix("/static/client/", fileServer))
	s.mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	s.mux.HandleFunc("/", s.handleMenu)
	s.mux.HandleFunc("/api/pedidos", s.handleCrearPedido)
	s.mux.HandleFunc("/ws/client", s.handleSSE)
	s.mux.HandleFunc("/qr", s.handleQR)
	s.mux.HandleFunc("/status", s.handleStatus)
}

// StaticFileServer returns a handler that serves static files (CSS, JS, etc.)
// from the embedded templates directory. This handler is intended to be used
// without authentication middleware.
func (s *Server) StaticFileServer() http.Handler {
	staticDir, err := fs.Sub(templateFS, "templates")
	if err != nil {
		panic("error al acceder a templates: " + err.Error())
	}
	return http.FileServer(http.FS(staticDir))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}