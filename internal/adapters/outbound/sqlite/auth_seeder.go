package sqlite

import (
	"context"
	"log"
	"os"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// SeedAuth realiza el bootstrap de usuarios iniciales si la tabla usuarios está vacía.
func SeedAuth(ctx context.Context, authSvc ports.AuthService) error {
	usuarios, err := authSvc.ListarUsuarios(ctx)
	if err != nil {
		return err
	}
	if len(usuarios) > 0 {
		return nil // Ya existen usuarios, omitir silenciosamente
	}

	adminEmail := os.Getenv("ADMIN_BOOTSTRAP_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@restaurante.com"
	}
	adminPass := os.Getenv("ADMIN_BOOTSTRAP_PASSWORD")
	if adminPass == "" {
		adminPass = "Admin1234!"
	}

	_, err = authSvc.CrearUsuario(ctx, ports.CrearUsuarioRequest{
		Nombre:   "Administrador General",
		Email:    adminEmail,
		Password: adminPass,
		Rol:      domain.RolAdmin,
	})
	if err != nil {
		return err
	}
	log.Printf("✅ Usuario administrador inicial creado: %s", adminEmail)

	// Sembrar operadores de demostración para pruebas de RBAC
	rolesDemo := []struct {
		nombre string
		email  string
		pass   string
		rol    domain.Rol
	}{
		{"Gerente de Turno", "gerente@restaurante.com", "Gerente1234!", domain.RolGerente},
		{"Chef de Cocina", "cocinero@restaurante.com", "Cocina1234!", domain.RolCocinero},
		{"Mesero Principal", "mesero@restaurante.com", "Mesero1234!", domain.RolMesero},
		{"Cajero Central", "cajero@restaurante.com", "Cajero1234!", domain.RolCajero},
	}

	for _, demo := range rolesDemo {
		_, err := authSvc.CrearUsuario(ctx, ports.CrearUsuarioRequest{
			Nombre:   demo.nombre,
			Email:    demo.email,
			Password: demo.pass,
			Rol:      demo.rol,
		})
		if err != nil {
			log.Printf("⚠️ No se pudo crear usuario demo %s: %v", demo.email, err)
		} else {
			log.Printf("✅ Usuario demo creado: %s (%s)", demo.email, demo.rol)
		}
	}

	return nil
}
