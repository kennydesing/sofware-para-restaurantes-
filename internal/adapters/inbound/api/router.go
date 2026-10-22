package api

import (
	"embed"
	"io/fs"
	"net/http"

	"pos-inventory/internal/core/ports"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	mux            *http.ServeMux
	inventario     ports.InventarioService
	pedido         ports.PedidoService
	insumoRepo     ports.InsumoRepository
	movimientoRepo ports.MovimientoRepository
	platoRepo      ports.PlatoRepository
	notifier       ports.NotifierRepository
}

func NewServer(
	inventario ports.InventarioService,
	pedido ports.PedidoService,
	insumoRepo ports.InsumoRepository,
	movimientoRepo ports.MovimientoRepository,
	platoRepo ports.PlatoRepository,
	notifier ports.NotifierRepository,
) *Server {
	s := &Server{
		mux:            http.NewServeMux(),
		inventario:     inventario,
		pedido:         pedido,
		insumoRepo:     insumoRepo,
		movimientoRepo: movimientoRepo,
		platoRepo:      platoRepo,
		notifier:       notifier,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	staticDir, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("error al acceder a archivos estáticos: " + err.Error())
	}

	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticDir))))

	s.mux.HandleFunc("/insumos", s.handleInsumos)
	s.mux.HandleFunc("/insumos/", s.handleInsumoByID)
	s.mux.HandleFunc("/platos", s.handlePlatos)
	s.mux.HandleFunc("/pedidos", s.handlePedidos)
	s.mux.HandleFunc("/pedidos/", s.handlePedidoByID)
	s.mux.HandleFunc("/compras", s.handleCompras)
	s.mux.HandleFunc("/mermas", s.handleMermas)
	s.mux.HandleFunc("/ajustes", s.handleAjustes)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s)
}