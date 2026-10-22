package bcv

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"pos-inventory/internal/core/ports"
)

type DolarAPIProvider struct {
	client    *http.Client
	bcvClient *http.Client
	baseURL   string
}

func NewDolarAPIProvider() ports.BCVRateProvider {
	return &DolarAPIProvider{
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
		bcvClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		baseURL: "https://ve.dolarapi.com/v1",
	}
}

var _ ports.BCVRateProvider = (*DolarAPIProvider)(nil)

type DolarAPIResponse struct {
	Moneda   string  `json:"moneda"`
	Promedio float64 `json:"promedio"`
}

var (
	reDolar = regexp.MustCompile(`(?s)id=["']dolar["'].*?<strong[^>]*>\s*([0-9.,]+)\s*</strong>`)
	reEuro  = regexp.MustCompile(`(?s)id=["']euro["'].*?<strong[^>]*>\s*([0-9.,]+)\s*</strong>`)
)

func (p *DolarAPIProvider) ObtenerTasasBCV(ctx context.Context) (float64, float64, error) {
	// 1. Intentar consultar directamente el portal oficial del Banco Central de Venezuela
	// para obtener la tasa oficial más reciente publicada para el día de hoy
	usd, eur, err := p.consultarBCVOficial(ctx)
	if err == nil && usd > 0 && eur > 0 {
		return usd, eur, nil
	}

	// 2. Si falla el portal oficial del BCV (ej. caída de servidor estatal), recurrir a DolarAPI como respaldo
	usdFallback, errUSD := p.consultarTasa(ctx, "/dolares/oficial")
	if errUSD != nil {
		if err != nil {
			return 0, 0, fmt.Errorf("fallo BCV oficial (%v) y respaldo DolarAPI (%w)", err, errUSD)
		}
		return 0, 0, fmt.Errorf("error obteniendo USD oficial: %w", errUSD)
	}

	eurFallback, errEUR := p.consultarTasa(ctx, "/euros/oficial")
	if errEUR != nil {
		if err != nil {
			return 0, 0, fmt.Errorf("fallo BCV oficial (%v) y respaldo DolarAPI (%w)", err, errEUR)
		}
		return 0, 0, fmt.Errorf("error obteniendo EUR oficial: %w", errEUR)
	}

	return usdFallback, eurFallback, nil
}

func (p *DolarAPIProvider) consultarBCVOficial(ctx context.Context) (float64, float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.bcv.org.ve", nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := p.bcvClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("error conectando a bcv.org.ve: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("código de respuesta bcv.org.ve: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("error leyendo respuesta bcv.org.ve: %w", err)
	}

	bodyStr := string(body)

	matchUSD := reDolar.FindStringSubmatch(bodyStr)
	if len(matchUSD) < 2 {
		return 0, 0, fmt.Errorf("no se encontró patrón USD en portal BCV")
	}
	usdStr := strings.TrimSpace(matchUSD[1])
	usdStr = strings.ReplaceAll(usdStr, ".", "")
	usdStr = strings.ReplaceAll(usdStr, ",", ".")
	usd, err := strconv.ParseFloat(usdStr, 64)
	if err != nil || usd <= 0 {
		return 0, 0, fmt.Errorf("tasa USD inválida extraída del BCV: %s", usdStr)
	}

	matchEUR := reEuro.FindStringSubmatch(bodyStr)
	if len(matchEUR) < 2 {
		return 0, 0, fmt.Errorf("no se encontró patrón EUR en portal BCV")
	}
	eurStr := strings.TrimSpace(matchEUR[1])
	eurStr = strings.ReplaceAll(eurStr, ".", "")
	eurStr = strings.ReplaceAll(eurStr, ",", ".")
	eur, err := strconv.ParseFloat(eurStr, 64)
	if err != nil || eur <= 0 {
		return 0, 0, fmt.Errorf("tasa EUR inválida extraída del BCV: %s", eurStr)
	}

	return usd, eur, nil
}

func (p *DolarAPIProvider) consultarTasa(ctx context.Context, endpoint string) (float64, error) {
	url := p.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "POSCore-Venezuela/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("conexión fallida con API de tasas (%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("respuesta no exitosa de API de tasas: código %d", resp.StatusCode)
	}

	var data DolarAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("error decodificando respuesta de tasa: %w", err)
	}

	if data.Promedio <= 0 {
		return 0, fmt.Errorf("la tasa recibida (%v) es inválida", data.Promedio)
	}

	return data.Promedio, nil
}
