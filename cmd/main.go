package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"pos-inventory/internal/adapters/inbound/admin"
	"pos-inventory/internal/adapters/inbound/api"
	"pos-inventory/internal/adapters/inbound/auth"
	"pos-inventory/internal/adapters/inbound/client"
	"pos-inventory/internal/adapters/inbound/kitchen"
	"pos-inventory/internal/adapters/inbound/shared"
	"pos-inventory/internal/adapters/inbound/shared/middleware"
	"pos-inventory/internal/adapters/outbound/bcv"
	"pos-inventory/internal/adapters/outbound/notifier"
	"pos-inventory/internal/adapters/outbound/printer"
	"pos-inventory/internal/adapters/outbound/sqlite"
	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/services"
)

func main() {
	ctx := context.Background()

	fmt.Println("================================================================================")
	fmt.Println(" SISTEMA DE INVENTARIO Y POS PARA RESTAURANTES (ARQUITECTURA HEXAGONAL - GO)   ")
	fmt.Println("================================================================================")

	dbFile := "./data/restaurante_pos.db"

	db, err := sqlite.NewDB(dbFile)
	if err != nil {
		log.Fatalf("❌ Error crítico inicializando SQLite: %v", err)
	}
	defer db.Close()
	fmt.Printf("✅ Base de datos SQLite inicializada en WAL mode: %s\n", dbFile)

	for _, migrationFile := range []string{"000001_init_schema.up.sql", "000002_auth.up.sql", "000003_tasa_cambio.up.sql", "000004_unidades_medida.up.sql", "000005_precio_venta_insumo.up.sql", "000006_empaques_cervezas_bebidas.up.sql"} {
		migrationPath := filepath.Join(".", "migrations", migrationFile)
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			migrationPath = filepath.Join("..", "migrations", migrationFile)
			migrationSQL, err = os.ReadFile(migrationPath)
			if err != nil {
				log.Fatalf("❌ Error leyendo archivo de migración %s: %v", migrationFile, err)
			}
		}

		if err := db.RunNamedMigration(ctx, migrationFile, string(migrationSQL)); err != nil {
			log.Fatalf("❌ Error aplicando migración %s: %v", migrationFile, err)
		}
		fmt.Printf("✅ Migración %s aplicada correctamente.\n", migrationFile)
	}


	insumoRepo := sqlite.NewInsumoRepository(db)
	platoRepo := sqlite.NewPlatoRepository(db)
	recetaRepo := sqlite.NewRecetaRepository(db)
	pedidoRepo := sqlite.NewPedidoRepository(db)
	movimientoRepo := sqlite.NewMovimientoRepository(db)
	atomicDebitor := sqlite.NewAtomicDebitor(db)
	printerAdapter := printer.NewConsolePrinterAdapter()
	notifierAdapter := notifier.NewLogNotifierAdapter()

	// Repositorio y Servicio de Unidades de Medida (Spec 003)
	unidadRepo := sqlite.NewUnidadMedidaRepositorySQLite(db)
	conversionSvc := services.NewConversionService(unidadRepo, insumoRepo)

	// Cargar caché de unidades de medida al arranque
	if err := conversionSvc.CargarCache(ctx); err != nil {
		log.Fatalf("❌ Error cargando caché de unidades de medida: %v", err)
	}
	fmt.Printf("✅ Caché de unidades de medida cargada.\n")

	// Repositorios de Autenticación y Auditoría RBAC
	usuarioRepo := sqlite.NewUsuarioRepository(db)
	sesionRepo := sqlite.NewSesionRepository(db)
	auditoriaRepo := sqlite.NewAuditoriaRepository(db)

	inventarioService := services.NewInventarioService(insumoRepo, movimientoRepo, notifierAdapter)
	pedidoService := services.NewPedidoService(
		pedidoRepo,
		platoRepo,
		recetaRepo,
		insumoRepo,
		atomicDebitor,
		printerAdapter,
		notifierAdapter,
		conversionSvc,
	)

	// Servicios de Autenticación y Auditoría
	authService := services.NewAuthService(usuarioRepo, sesionRepo, auditoriaRepo)

	err = sqlite.SeedData(ctx, db, inventarioService, insumoRepo, platoRepo, recetaRepo)
	if err != nil {
		log.Fatalf("❌ Error en seeding de datos: %v", err)
	}
	fmt.Println("✅ Datos de demostración de catálogo e inventario cargados.")

	// Seeder de usuarios y administradores iniciales (TAREA-17)
	if err := sqlite.SeedAuth(ctx, authService); err != nil {
		log.Fatalf("❌ Error inicializando usuarios RBAC: %v", err)
	}
	fmt.Println("✅ Usuarios RBAC iniciales verificados/cargados correctamente.")

	// Rate Limiter y Auth Handler
	rateLimiter := middleware.NewRateLimiter()
	authHandler := auth.NewHandler(authService, rateLimiter)

	// Job de limpieza periódica de sesiones expiradas en segundo plano (TAREA-18)
	cleanupInterval := 60
	if val := os.Getenv("SESSION_CLEANUP_INTERVAL"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cleanupInterval = n
		}
	}
	go func() {
		ticker := time.NewTicker(time.Duration(cleanupInterval) * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			_, _ = authService.LimpiarSesionesExpiradas(context.Background())
		}
	}()

	// Repositorio y Servicio de Tasas de Cambio Oficiales (BCV)
	tasaRepo := sqlite.NewTasaCambioRepository(db)
	bcvProvider := bcv.NewDolarAPIProvider()
	tasaService := services.NewTasaCambioService(tasaRepo, bcvProvider)

	// Sincronización inicial asíncrona de tasas BCV al arrancar
	go func() {
		tasas, err := tasaService.SincronizarTasasBCV(context.Background())
		if err != nil {
			log.Printf("⚠️ Modo offline de tasas activo (no se pudo conectar con BCV en arranque): %v", err)
		} else {
			for _, t := range tasas {
				log.Printf("💵 Tasa oficial %s: %.2f Bs (Fuente: %s)", t.Moneda, t.Tasa, t.Fuente)
			}
		}
	}()

	// Job en segundo plano para sincronizar tasas BCV cada 10 minutos
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			_, _ = tasaService.SincronizarTasasBCV(context.Background())
		}
	}()

	apiServer := api.NewServer(inventarioService, pedidoService, insumoRepo, movimientoRepo, platoRepo, notifierAdapter)
	kitchenServer := kitchen.NewServer(pedidoService, printerAdapter, notifierAdapter)
	clientServer := client.NewServer(pedidoService, platoRepo, insumoRepo)
	adminServer := admin.NewServer(inventarioService, pedidoService, insumoRepo, movimientoRepo, platoRepo, recetaRepo, unidadRepo, authService, tasaService)

	mux := http.NewServeMux()

	// Archivos estáticos públicos (sin autenticación) - DEBEN ir ANTES de las rutas protegidas
	mux.Handle("/static/admin/", middleware.SecurityHeaders(http.StripPrefix("/static/admin/", adminServer.StaticFileServer())))
	mux.Handle("/static/kds/", middleware.SecurityHeaders(http.StripPrefix("/static/kds/", kitchenServer.StaticFileServer())))
	mux.Handle("/static/client/", middleware.SecurityHeaders(http.StripPrefix("/static/client/", clientServer.StaticFileServer())))

	// Rutas Públicas de Autenticación
	mux.Handle("/login", middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			authHandler.HandleGetLogin(w, r)
		case http.MethodPost:
			authHandler.HandlePostLogin(w, r)
		default:
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/logout", middleware.SecurityHeaders(http.HandlerFunc(authHandler.HandleLogout)))

	// Helper para envolver rutas con toda la cadena perimetral de seguridad y RBAC
	protect := func(h http.Handler, roles ...domain.Rol) http.Handler {
		return middleware.SecurityHeaders(
			shared.LoggingMiddleware(
				shared.RecoveryMiddleware(
					middleware.AuthMiddleware(authService)(
						middleware.CSRFMiddleware(
							middleware.RequireRole(roles...)(h),
						),
					),
				),
			),
		)
	}

	// Rutas Protegidas según la Matriz de Permisos (§3.2 de spec.md)
	// Admin Panel: Admin, Gerente, Cajero
	mux.Handle("/admin/", protect(http.StripPrefix("/admin", adminServer), domain.RolAdmin, domain.RolGerente, domain.RolCajero))
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
	})

	// Cocina (KDS): Admin, Cocinero
	mux.Handle("/kds/", protect(http.StripPrefix("/kds", kitchenServer), domain.RolAdmin, domain.RolCocinero))
	mux.HandleFunc("/kds", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/kds/", http.StatusMovedPermanently)
	})

	// API Gateway: Todos los roles autenticados
	mux.Handle("/api/", protect(http.StripPrefix("/api", apiServer),
		domain.RolAdmin, domain.RolGerente, domain.RolCajero, domain.RolMesero, domain.RolCocinero))

	// Cliente / Mesero / Mesa POS: Admin, Gerente, Cajero, Mesero
	mux.Handle("/", protect(clientServer, domain.RolAdmin, domain.RolGerente, domain.RolCajero, domain.RolMesero))

	fmt.Println("\n================================================================================")
	fmt.Println(" 🚀 SERVIDOR WEB INICIADO — 3 Interfaces POS + Seguridad RBAC Multirol")
	fmt.Println("================================================================================")
	fmt.Printf("🔐 Login Perimetral:      http://localhost:8080/login\n")
	fmt.Printf("🌐 Cliente (QR Mesa):     http://localhost:8080/?mesa=1\n")
	fmt.Printf("🍳 Cocina (KDS):          http://localhost:8080/kds/\n")
	fmt.Printf("⚙️ Admin (Dashboard):     http://localhost:8080/admin/\n")
	fmt.Printf("🔧 API Endpoints:         http://localhost:8080/api/...\n")
	fmt.Println("================================================================================")
	fmt.Println("🔑 Credenciales de Demostración:")
	fmt.Println("   • Admin:    admin@restaurante.com    | Admin1234!")
	fmt.Println("   • Gerente:  gerente@restaurante.com  | Gerente1234!")
	fmt.Println("   • Cocinero: cocinero@restaurante.com | Cocina1234!")
	fmt.Println("   • Mesero:   mesero@restaurante.com   | Mesero1234!")
	fmt.Println("   • Cajero:   cajero@restaurante.com   | Cajero1234!")
	fmt.Println("================================================================================")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("❌ Error iniciando servidor web: %v", err)
	}
}