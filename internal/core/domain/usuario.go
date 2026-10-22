package domain

import "time"

// Rol representa el conjunto de permisos de un usuario en el sistema POS.
type Rol string

const (
	RolAdmin    Rol = "admin"
	RolGerente  Rol = "gerente"
	RolCocinero Rol = "cocinero"
	RolMesero   Rol = "mesero"
	RolCajero   Rol = "cajero"
)

// RolesValidos es el conjunto cerrado de roles permitidos en el sistema.
var RolesValidos = map[Rol]bool{
	RolAdmin:    true,
	RolGerente:  true,
	RolCocinero: true,
	RolMesero:   true,
	RolCajero:   true,
}

// Usuario es la entidad pura de dominio para un operador del sistema.
// No contiene lógica de persistencia ni de HTTP.
type Usuario struct {
	ID           int
	Nombre       string
	Email        string
	PasswordHash string // Formato PHC Argon2id — NUNCA se expone en respuestas HTTP
	Rol          Rol
	Activo       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// EsActivo valida que la cuenta puede iniciar sesión.
func (u *Usuario) EsActivo() bool { return u.Activo }

// TieneRol comprueba si el usuario posee exactamente el rol indicado.
func (u *Usuario) TieneRol(rol Rol) bool { return u.Rol == rol }

// PuedeAcceder verifica si el usuario tiene alguno de los roles permitidos.
func (u *Usuario) PuedeAcceder(rolesPermitidos []Rol) bool {
	for _, r := range rolesPermitidos {
		if u.Rol == r {
			return true
		}
	}
	return false
}