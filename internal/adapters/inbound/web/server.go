package web

import (
	"embed"
	"io/fs"
	"net/http"

	"pos-inventory/internal/core/ports"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	mux            http.ServeMux
	inventario     ports.InventarioService
	pedido         ports.PedidoService
	insumoRepo     ports.InsumoRepository
	movimientoRepo ports.MovimientoRepository
	platoRepo      ports.PlatoRepository
}

func NewServer(
	inventario ports.InventarioService,
	pedido ports.PedidoService,
	insumoRepo ports.InsumoRepository,
	movimientoRepo ports.MovimientoRepository,
	platoRepo ports.PlatoRepository,
) *Server {
	s := &Server{
		inventario:     inventario,
		pedido:         pedido,
		insumoRepo:     insumoRepo,
		movimientoRepo: movimientoRepo,
		platoRepo:      platoRepo,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Serve index.html at the root
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(staticFS, "static/index.html")
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	s.mux.HandleFunc("/api/insumos", s.handleInsumos)
	s.mux.HandleFunc("/api/insumos/", s.handleInsumoByID)
	s.mux.HandleFunc("/api/platos", s.handlePlatos)
	s.mux.HandleFunc("/api/pedidos", s.handlePedidos)
	s.mux.HandleFunc("/api/pedidos/", s.handlePedidoByID)
	s.mux.HandleFunc("/api/compras", s.handleCompras)
	s.mux.HandleFunc("/api/mermas", s.handleMermas)
	s.mux.HandleFunc("/api/ajustes", s.handleAjustes)
}

// StaticFileHandler returns a handler that serves static files (CSS, JS, etc.)
// from the embedded static directory. This handler is intended to be used
// without authentication middleware.
func (s *Server) StaticFileHandler() http.Handler {
	staticDir, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("error al acceder a archivos estáticos: " + err.Error())
	}
	return http.FileServer(http.FS(staticDir))
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, &s.mux)
}