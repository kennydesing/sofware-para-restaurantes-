package bcv

import (
	"context"
	"testing"
	"time"
)

func TestBCVOficialLiveOrFallback(t *testing.T) {
	provider := NewDolarAPIProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	usd, eur, err := provider.ObtenerTasasBCV(ctx)
	if err != nil {
		t.Fatalf("ObtenerTasasBCV falló: %v", err)
	}

	t.Logf("USD obtenido: %.4f, EUR obtenido: %.4f", usd, eur)

	if usd <= 0 {
		t.Errorf("Se esperaba tasa USD > 0, obtenido: %f", usd)
	}
	if eur <= 0 {
		t.Errorf("Se esperaba tasa EUR > 0, obtenido: %f", eur)
	}
}
