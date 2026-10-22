package domain

import "errors"

var (
	// ErrStockInsuficiente indica que un insumo no tiene suficiente cantidad para satisfacer la receta.
	ErrStockInsuficiente = errors.New("stock insuficiente para cubrir los requerimientos del pedido")

	// ErrInsumoNoEncontrado indica que el insumo especificado no existe en el sistema.
	ErrInsumoNoEncontrado = errors.New("insumo no encontrado")

	// ErrPlatoNoEncontrado indica que el plato del menú no existe en el sistema.
	ErrPlatoNoEncontrado = errors.New("plato no encontrado")

	// ErrPedidoNoEncontrado indica que la comanda/pedido solicitado no existe.
	ErrPedidoNoEncontrado = errors.New("pedido no encontrado")

	// ErrRecetaVacia indica que el plato no tiene escandallo ni ingredientes asociados.
	ErrRecetaVacia = errors.New("el plato no tiene receta configurada para descuento de inventario")

	// ErrUnidadInvalida indica que la unidad de compra no se puede convertir a la unidad base.
	ErrUnidadInvalida = errors.New("unidad de medida inválida o incompatible")

	// ErrEstadoPedidoInvalido indica que la orden no está en un estado que permita la acción solicitada.
	ErrEstadoPedidoInvalido = errors.New("el estado actual del pedido no permite esta operación")

	// ErrCantidadInvalida indica que la cantidad provista es cero o negativa.
	ErrCantidadInvalida = errors.New("la cantidad debe ser mayor a cero")

	// ErrNotFound indica que el recurso solicitado no existe.
	ErrNotFound = errors.New("recurso no encontrado")

	// --- Errores de Autenticación y Autorización (Spec 002) ---

	// ErrCredencialesInvalidas se devuelve cuando email o contraseña son incorrectos.
	ErrCredencialesInvalidas = errors.New("credenciales inválidas")

	// ErrCuentaInactiva se devuelve cuando el usuario existe pero está desactivado.
	ErrCuentaInactiva = errors.New("cuenta desactivada")

	// ErrSesionExpirada se devuelve cuando el token de sesión ha vencido.
	ErrSesionExpirada = errors.New("sesión expirada")

	// ErrSesionInvalida se devuelve cuando el token no existe o es inválido.
	ErrSesionInvalida = errors.New("sesión inválida o no encontrada")

	// ErrEmailDuplicado se devuelve al intentar registrar un email ya existente.
	ErrEmailDuplicado = errors.New("el email ya está registrado")

	// ErrRolInvalido se devuelve cuando el rol proporcionado no está en el conjunto permitido.
	ErrRolInvalido = errors.New("rol no reconocido")

	// ErrAccesoDenegado se devuelve cuando el rol del usuario no tiene permisos para la acción.
	ErrAccesoDenegado = errors.New("acceso denegado: rol insuficiente")

	// ErrPasswordInsegura se devuelve cuando la contraseña no cumple la política mínima.
	ErrPasswordInsegura = errors.New("la contraseña no cumple los requisitos mínimos")
)