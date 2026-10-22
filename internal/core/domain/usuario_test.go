package domain

import (
	"testing"
	"time"
)

func TestUsuario_PuedeAcceder(t *testing.T) {
	tests := []struct {
		name           string
		usuarioRol     Rol
		rolesPermitidos []Rol
		expected       bool
	}{
		{
			name:           "admin puede acceder a ruta admin",
			usuarioRol:     RolAdmin,
			rolesPermitidos: []Rol{RolAdmin},
			expected:       true,
		},
		{
			name:           "admin puede acceder a ruta multi-rol",
			usuarioRol:     RolAdmin,
			rolesPermitidos: []Rol{RolAdmin, RolGerente},
			expected:       true,
		},
		{
			name:           "gerente no puede acceder a ruta solo admin",
			usuarioRol:     RolGerente,
			rolesPermitidos: []Rol{RolAdmin},
			expected:       false,
		},
		{
			name:           "cocinero puede acceder a KDS",
			usuarioRol:     RolCocinero,
			rolesPermitidos: []Rol{RolAdmin, RolCocinero},
			expected:       true,
		},
		{
			name:           "mesero no puede acceder a admin",
			usuarioRol:     RolMesero,
			rolesPermitidos: []Rol{RolAdmin, RolGerente},
			expected:       false,
		},
		{
			name:           "lista vacía de roles permitidos",
			usuarioRol:     RolAdmin,
			rolesPermitidos: []Rol{},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &Usuario{Rol: tt.usuarioRol}
			result := u.PuedeAcceder(tt.rolesPermitidos)
			if result != tt.expected {
				t.Errorf("PuedeAcceder() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUsuario_EsActivo(t *testing.T) {
	activo := &Usuario{Activo: true}
	if !activo.EsActivo() {
		t.Error("EsActivo() should return true for active user")
	}

	inactivo := &Usuario{Activo: false}
	if inactivo.EsActivo() {
		t.Error("EsActivo() should return false for inactive user")
	}
}

func TestSesion_EstaVigente(t *testing.T) {
	futura := &Sesion{ExpiresAt: time.Now().Add(1 * time.Hour)}
	if !futura.EstaVigente() {
		t.Error("EstaVigente() should return true for future expiry")
	}

	pasada := &Sesion{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	if pasada.EstaVigente() {
		t.Error("EstaVigente() should return false for past expiry")
	}
}