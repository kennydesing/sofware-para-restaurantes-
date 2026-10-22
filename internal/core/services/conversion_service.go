package services

import (
	"context"
	"fmt"
	"sync"

	"pos-inventory/internal/core/domain"
	"pos-inventory/internal/core/ports"
)

// ConversionServiceImpl implementa el puerto de entrada ports.ConversionService.
// Utiliza aritmética de fracciones exactas (INT64) internamente y materializa
// el resultado a float64 solo en la capa de presentación.
// Mantiene una caché en memoria de unidades de medida para resolución O(1).
type ConversionServiceImpl struct {
	repo      ports.UnidadMedidaRepository
	insumoRepo ports.InsumoRepository
	cache     map[string]domain.UnidadMedida
	mu        sync.RWMutex
	cargado   bool
}

// NewConversionService construye una nueva instancia del servicio de conversión.
func NewConversionService(repo ports.UnidadMedidaRepository, insumoRepo ports.InsumoRepository) *ConversionServiceImpl {
	return &ConversionServiceImpl{
		repo:      repo,
		insumoRepo: insumoRepo,
		cache:     make(map[string]domain.UnidadMedida),
		cargado:   false,
	}
}

// CargarCache carga todas las unidades de medida desde el repositorio y construye
// el mapa abreviatura → UnidadMedida para resolución O(1) durante la vida del proceso.
// Debe llamarse al arranque de la aplicación.
func (s *ConversionServiceImpl) CargarCache(ctx context.Context) error {
	unidades, err := s.repo.ObtenerTodas(ctx)
	if err != nil {
		return fmt.Errorf("error al cargar caché de unidades de medida: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache = make(map[string]domain.UnidadMedida, len(unidades))
	for _, u := range unidades {
		s.cache[u.Abreviatura] = u
	}
	s.cargado = true

	return nil
}

// InvalidarCache resetea el mapa de caché. Debe llamarse tras crear una nueva unidad
// para forzar una recarga en la próxima operación.
func (s *ConversionServiceImpl) InvalidarCache() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache = make(map[string]domain.UnidadMedida)
	s.cargado = false
}

// obtenerUnidad busca una unidad por abreviatura en la caché con bloqueo de lectura.
func (s *ConversionServiceImpl) obtenerUnidad(abrev string) (domain.UnidadMedida, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.cache[abrev]
	return u, ok
}

// Convertir convierte una cantidad entre dos unidades de la misma categoría usando
// aritmética de fracciones exactas (INT64) y materializa el resultado con 4 decimales.
func (s *ConversionServiceImpl) Convertir(ctx context.Context, cantidad float64, origenAbrev, destinoAbrev string) (float64, error) {
	if !s.cargado {
		return 0, fmt.Errorf("caché de unidades de medida no cargada: llame a CargarCache() al arranque")
	}

	origen, ok := s.obtenerUnidad(origenAbrev)
	if !ok {
		return 0, fmt.Errorf("unidad de origen '%s' no encontrada", origenAbrev)
	}
	destino, ok := s.obtenerUnidad(destinoAbrev)
	if !ok {
		return 0, fmt.Errorf("unidad de destino '%s' no encontrada", destinoAbrev)
	}

	// Representar la cantidad como fracción exacta: cantidad = cantidadN / 1e9
	cantidadN := int64(cantidad * 1e9)
	cantidadD := int64(1e9)

	rN, rD, err := domain.ConvertirEntreMismaCategoria(cantidadN, cantidadD, origen, destino)
	if err != nil {
		return 0, err
	}

	return domain.ConvertirAFloat64(rN, rD, 4), nil
}

// CostoUnitarioBase calcula el costo por unidad base mínima a partir del total de la compra,
// la cantidad comprada y la unidad de compra registrada para el insumo.
// Retorna el costo con 4 decimales de precisión.
func (s *ConversionServiceImpl) CostoUnitarioBase(ctx context.Context, costoTotal, cantidadComprada float64, unidadCompraAbrev string, insumoID int64) (float64, error) {
	if !s.cargado {
		return 0, fmt.Errorf("caché de unidades de medida no cargada: llame a CargarCache() al arranque")
	}
	if cantidadComprada <= 0 {
		return 0, fmt.Errorf("la cantidad comprada debe ser mayor a cero")
	}
	if costoTotal < 0 {
		return 0, fmt.Errorf("el costo total no puede ser negativo")
	}

	unidadCompra, ok := s.obtenerUnidad(unidadCompraAbrev)
	if !ok {
		return 0, fmt.Errorf("unidad de compra '%s' no encontrada", unidadCompraAbrev)
	}

	// Obtener equivalencias de compra del insumo para validar la unidad
	unidadesCompra, err := s.repo.ObtenerUnidadesCompraInsumo(ctx, insumoID)
	if err != nil {
		return 0, fmt.Errorf("error al consultar unidades de compra del insumo %d: %w", insumoID, err)
	}

	// Buscar la unidad de compra entre las registradas para este insumo
	encontrada := false
	for _, iuc := range unidadesCompra {
		if iuc.UnidadID == unidadCompra.ID {
			encontrada = true
			break
		}
	}
	if !encontrada && len(unidadesCompra) > 0 {
		return 0, fmt.Errorf("la unidad de compra '%s' no está registrada para el insumo %d", unidadCompraAbrev, insumoID)
	}

	// Convertir cantidad comprada a unidad base mínima
	// Fórmula: cantidad_base = cantidad_compra × (factorN / factorD) de la unidad de compra
	cantidadN := int64(cantidadComprada * 1e9)
	cantidadD := int64(1e9)

	// Usar ConvertirEntreMismaCategoria con la unidad de compra como origen y la
	// unidad base de su categoría como destino. Buscamos la unidad base de la categoría
	// de unidadCompra.
	unidadesCategoria, err := s.repo.ObtenerPorCategoria(ctx, unidadCompra.Categoria)
	if err != nil {
		return 0, fmt.Errorf("error al obtener unidades de la categoría %s: %w", unidadCompra.Categoria, err)
	}

	var unidadBase domain.UnidadMedida
	for _, u := range unidadesCategoria {
		if u.EsBase {
			unidadBase = u
			break
		}
	}

	rN, rD, err := domain.ConvertirEntreMismaCategoria(cantidadN, cantidadD, unidadCompra, unidadBase)
	if err != nil {
		return 0, err
	}

	cantidadBase := domain.ConvertirAFloat64(rN, rD, 4)
	if cantidadBase <= 0 {
		return 0, fmt.Errorf("la cantidad convertida a unidad base es cero o negativa")
	}

	costoUnitarioBase := costoTotal / cantidadBase
	return domain.ConvertirAFloat64(int64(costoUnitarioBase*1e4), 1e4, 4), nil
}

// ConsumoReceta calcula la cantidad en unidad base del insumo a descontar del stock,
// aplicando la fórmula de merma del negocio.
// Si unidadRecetaAbrev está vacía, se asume que la cantidad ya está en unidad base del insumo.
func (s *ConversionServiceImpl) ConsumoReceta(ctx context.Context, cantidadReceta float64, unidadRecetaAbrev string, insumoID int64, porcentajeMerma float64, numPlatos int) (float64, error) {
	if !s.cargado {
		return 0, fmt.Errorf("caché de unidades de medida no cargada: llame a CargarCache() al arranque")
	}
	if numPlatos <= 0 {
		return 0, fmt.Errorf("el número de platos debe ser mayor a cero")
	}

	var cantidadBase float64

	if unidadRecetaAbrev == "" {
		// Ruta legacy: la cantidad ya está en unidad base del insumo
		cantidadBase = cantidadReceta
	} else {
		unidadReceta, ok := s.obtenerUnidad(unidadRecetaAbrev)
		if !ok {
			return 0, fmt.Errorf("unidad de receta '%s' no encontrada", unidadRecetaAbrev)
		}

		// Obtener el insumo para conocer su unidad base
		insumo, err := s.insumoRepo.GetByID(ctx, int(insumoID))
		if err != nil {
			return 0, fmt.Errorf("error al obtener insumo %d: %w", insumoID, err)
		}

		// Convertir de unidad de receta a unidad base del insumo
		unidadBaseInsumo, ok := s.obtenerUnidad(insumo.UnidadBase)
		if !ok {
			return 0, fmt.Errorf("unidad base '%s' del insumo no encontrada en el catálogo", insumo.UnidadBase)
		}

		if unidadReceta.Categoria != unidadBaseInsumo.Categoria {
			return 0, domain.ErrCategoriaIncompatible
		}

		cantidadN := int64(cantidadReceta * 1e9)
		cantidadD := int64(1e9)

		rN, rD, err := domain.ConvertirEntreMismaCategoria(cantidadN, cantidadD, unidadReceta, unidadBaseInsumo)
		if err != nil {
			return 0, err
		}

		cantidadBase = domain.ConvertirAFloat64(rN, rD, 4)
	}

	return domain.ConsumoBaseConMerma(
		int64(cantidadBase*1e9), 1e9,
		1, 1, // factor de conversión identidad (ya está en unidad base del insumo)
		porcentajeMerma,
		int64(numPlatos),
	), nil
}
