package domain

import "testing"

func TestUnidadesPorDefectoEmpaque_Gavera36(t *testing.T) {
	if n := UnidadesPorDefectoEmpaque(EmpaqueGavera36); n != 36 {
		t.Fatalf("esperado 36 unidades para gavera, obtuvo %d", n)
	}
}

func TestUnidadesPorDefectoEmpaque_Caja24Bot(t *testing.T) {
	if n := UnidadesPorDefectoEmpaque(EmpaqueCaja24Bot); n != 24 {
		t.Fatalf("esperado 24 unidades para caja 24 botellas, obtuvo %d", n)
	}
}

func TestUnidadesPorDefectoEmpaque_Caja24Lata(t *testing.T) {
	if n := UnidadesPorDefectoEmpaque(EmpaqueCaja24Lata); n != 24 {
		t.Fatalf("esperado 24 unidades para caja 24 latas, obtuvo %d", n)
	}
}

func TestUnidadesPorDefectoEmpaque_SixPack6(t *testing.T) {
	if n := UnidadesPorDefectoEmpaque(EmpaqueSixPack6); n != 6 {
		t.Fatalf("esperado 6 unidades para six-pack, obtuvo %d", n)
	}
}

func TestUnidadesPorDefectoEmpaque_Caja12Art(t *testing.T) {
	if n := UnidadesPorDefectoEmpaque(EmpaqueCaja12Art); n != 12 {
		t.Fatalf("esperado 12 unidades para caja artesanal, obtuvo %d", n)
	}
}

func TestUnidadesPorDefectoEmpaque_Unitario(t *testing.T) {
	if n := UnidadesPorDefectoEmpaque(EmpaqueUnitario); n != 1 {
		t.Fatalf("esperado 1 unidad para unitario, obtuvo %d", n)
	}
}

func TestUnidadesPorDefectoEmpaque_Personalizado(t *testing.T) {
	if n := UnidadesPorDefectoEmpaque(EmpaquePersonalizado); n != 1 {
		t.Fatalf("esperado 1 unidad por defecto para personalizado, obtuvo %d", n)
	}
}

func TestConvertirEmpaquesABase_Gavera36(t *testing.T) {
	insumo := &Insumo{
		UnidadesPorEmpaque: 36,
	}

	// 1 gavera = 36 unidades
	resultado := insumo.ConvertirEmpaquesABase(1)
	if resultado != 36 {
		t.Fatalf("esperado 36 unidades para 1 gavera, obtuvo %v", resultado)
	}

	// 3 gaveras = 108 unidades
	resultado = insumo.ConvertirEmpaquesABase(3)
	if resultado != 108 {
		t.Fatalf("esperado 108 unidades para 3 gaveras, obtuvo %v", resultado)
	}
}

func TestConvertirEmpaquesABase_Caja24(t *testing.T) {
	insumo := &Insumo{
		UnidadesPorEmpaque: 24,
	}

	// 2 cajas = 48 unidades
	resultado := insumo.ConvertirEmpaquesABase(2)
	if resultado != 48 {
		t.Fatalf("esperado 48 unidades para 2 cajas, obtuvo %v", resultado)
	}
}

func TestConvertirEmpaquesABase_CustomPack18(t *testing.T) {
	insumo := &Insumo{
		UnidadesPorEmpaque: 18,
	}

	// 5 packs personalizados de 18 = 90 unidades
	resultado := insumo.ConvertirEmpaquesABase(5)
	if resultado != 90 {
		t.Fatalf("esperado 90 unidades para 5 packs de 18, obtuvo %v", resultado)
	}
}

func TestConvertirEmpaquesABase_FactorCero(t *testing.T) {
	insumo := &Insumo{
		UnidadesPorEmpaque: 0,
	}

	// Cuando el factor es 0, debe retornar la cantidad tal cual (factor = 1)
	resultado := insumo.ConvertirEmpaquesABase(10)
	if resultado != 10 {
		t.Fatalf("esperado 10 unidades cuando factor es 0 (usa 1 por defecto), obtuvo %v", resultado)
	}
}
