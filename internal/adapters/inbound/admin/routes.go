package admin

import (
	"io/fs"
	"net/http"
	"os"
	"strings"

	"pos-inventory/internal/adapters/inbound/shared"
	"pos-inventory/internal/core/ports"
)

type Server struct {
	mux             *http.ServeMux
	inventarioSvc   ports.InventarioService
	pedidoSvc       ports.PedidoService
	insumoRepo      ports.InsumoRepository
	movimientoRepo  ports.MovimientoRepository
	platoRepo       ports.PlatoRepository
	recetaRepo      ports.RecetaRepository
	unidadRepo      ports.UnidadMedidaRepository
	authSvc         ports.AuthService
	tasaSvc         ports.TasaCambioService
	staticFileServer http.Handler
}

func NewServer(
	inventarioSvc ports.InventarioService,
	pedidoSvc ports.PedidoService,
	insumoRepo ports.InsumoRepository,
	movimientoRepo ports.MovimientoRepository,
	platoRepo ports.PlatoRepository,
	recetaRepo ports.RecetaRepository,
	unidadRepo ports.UnidadMedidaRepository,
	authSvc ports.AuthService,
	tasaSvc ports.TasaCambioService,
) *Server {
	s := &Server{
		mux:            http.NewServeMux(),
		inventarioSvc:  inventarioSvc,
		pedidoSvc:      pedidoSvc,
		insumoRepo:     insumoRepo,
		movimientoRepo: movimientoRepo,
		platoRepo:      platoRepo,
		recetaRepo:     recetaRepo,
		unidadRepo:     unidadRepo,
		authSvc:        authSvc,
		tasaSvc:        tasaSvc,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	staticDir, err := fs.Sub(templateFS, "static")
	if err != nil {
		panic("error al acceder a archivos estáticos de admin: " + err.Error())
	}

	// Servidor de archivos estáticos para exposición pública
	s.staticFileServer = http.FileServer(http.FS(staticDir))

	// Rutas internas del admin (protegidas por su propio middleware)
	s.mux.Handle("/static/", http.StripPrefix("/static/", s.staticFileServer))

	s.mux.HandleFunc("/", s.authMiddleware(s.handleAdminPage))

	s.mux.HandleFunc("/api/stats", s.authMiddleware(s.handleStatsAPI))
	s.mux.HandleFunc("/api/insumos", s.authMiddleware(s.handleInsumosAPI))
	s.mux.HandleFunc("/api/insumos/", s.authMiddleware(s.handleInsumoByIDAPI))
	s.mux.HandleFunc("/api/compras", s.authMiddleware(s.handleComprasAPI))
	s.mux.HandleFunc("/api/mermas", s.authMiddleware(s.handleMermasAPI))
	s.mux.HandleFunc("/api/mermas/historial", s.authMiddleware(s.handleMermasHistorialAPI))
	s.mux.HandleFunc("/api/uso-personal", s.authMiddleware(s.handleUsoPersonalAPI))
	s.mux.HandleFunc("/api/uso-personal/historial", s.authMiddleware(s.handleUsoPersonalHistorialAPI))
	s.mux.HandleFunc("/api/ajustes", s.authMiddleware(s.handleAjustesAPI))
	s.mux.HandleFunc("/api/auditoria-stock", s.authMiddleware(s.handleAuditoriaStockAPI))
	s.mux.HandleFunc("/api/platos", s.authMiddleware(s.handlePlatosAPI))
	s.mux.HandleFunc("/api/pedidos", s.authMiddleware(s.handlePedidosAPI))
	s.mux.HandleFunc("/api/auditoria", s.authMiddleware(s.handleAuditoriaMovimientosAPI))
	s.mux.HandleFunc("/api/usuarios", s.authMiddleware(s.handleUsuariosAPI))
	s.mux.HandleFunc("/api/usuarios/modificar", s.authMiddleware(s.handleModificarUsuarioAPI))
	s.mux.HandleFunc("/api/perfil", s.authMiddleware(s.handlePerfilAPI))

	// Rutas de Tasas Oficiales de Cambio (BCV)
	s.mux.HandleFunc("/api/tasas", s.authMiddleware(s.handleTasasAPI))
	s.mux.HandleFunc("/api/tasas/sincronizar", s.authMiddleware(s.handleSincronizarTasasAPI))
	s.mux.HandleFunc("/api/tasas/manual", s.authMiddleware(s.handleActualizarTasaManualAPI))

	// Unidades de Medida (Spec 003)
	s.mux.HandleFunc("/api/unidades", s.authMiddleware(s.handleUnidadesAPI))
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminToken := os.Getenv("ADMIN_TOKEN")
		if adminToken != "" {
			authHeader := r.Header.Get("Authorization")
			tokenParam := r.URL.Query().Get("token")
			providedToken := strings.TrimPrefix(authHeader, "Bearer ")
			if providedToken == "" {
				providedToken = tokenParam
			}
			if providedToken != adminToken {
				w.Header().Set("WWW-Authenticate", `Basic realm="Admin POS"`)
				user, pass, ok := r.BasicAuth()
				if !ok || (pass != adminToken && user != "admin") {
					shared.WriteError(w, http.StatusUnauthorized, "Acceso no autorizado al Panel Admin")
					return
				}
			}
		}
		next(w, r)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) StaticFileServer() http.Handler {
	return s.staticFileServer
}
