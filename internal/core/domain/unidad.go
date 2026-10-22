package domain

//Categoria representa la clasificación de una unidad de medida.
type Categoria string

const (
	CategoriaVolumen    Categoria = "VOLUMEN"
	CategoriaPeso       Categoria = "PESO"
	CategoriaConteo     Categoria = "CONTEO"
	CategoriaCulinaria  Categoria = "CULINARIA"
)

// UnidadMedida representa una unidad de medida en el catálogo.
// 1 unidad de esta = (FactorN / FactorD) unidades base de su categoría.
// Sin dependencias de I/O, base de datos ni frameworks externos.
type UnidadMedida struct {
	ID          int64
	Nombre      string
	Abreviatura string
	Categoria  	Categoria
	EsBase      bool
	FactorN     int64 // > 0
	FactorD     int64 // > 0, nunca 0
}

// InsumoUnidadCompra representa la equivalencia entre una presentación de compra
// y la unidad base del insumo.
type InsumoUnidadCompra struct {
	ID          int64
	InsumoID    int64
	UnidadID    int64
	FactorN     int64
	FactorD     int64
	EsPrincipal bool
	Description string
}