package middleware

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ipRecord struct {
	attempts    int
	lastAttempt time.Time
	blockedTill time.Time
}

// RateLimiter gestiona límites de tasa de peticiones en memoria por dirección IP.
type RateLimiter struct {
	mu           sync.RWMutex
	records      map[string]*ipRecord
	maxAttempts  int
	window       time.Duration
	blockTimeout time.Duration
}

// NewRateLimiter crea un nuevo limitador de tasa configurable por variables de entorno.
func NewRateLimiter() *RateLimiter {
	maxAttempts := getEnvInt("RATE_LIMIT_MAX_ATTEMPTS", 5)
	windowMinutes := getEnvInt("RATE_LIMIT_WINDOW_MINUTES", 15)

	rl := &RateLimiter{
		records:      make(map[string]*ipRecord),
		maxAttempts:  maxAttempts,
		window:       time.Duration(windowMinutes) * time.Minute,
		blockTimeout: time.Duration(windowMinutes) * time.Minute,
	}

	// Goroutine en segundo plano para limpiar entradas viejas
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, rec := range rl.records {
		if now.Sub(rec.lastAttempt) > rl.window && now.After(rec.blockedTill) {
			delete(rl.records, ip)
		}
	}
}

// Check valida si la IP está permitida o si ha superado el umbral.
func (rl *RateLimiter) Check(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rec, exists := rl.records[ip]
	if !exists {
		return true
	}

	if now.Before(rec.blockedTill) {
		return false
	}

	// Si expiró la ventana, reiniciar contador
	if now.Sub(rec.lastAttempt) > rl.window {
		rec.attempts = 0
	}

	return rec.attempts < rl.maxAttempts
}

// RecordFailedAttempt registra un intento fallido y bloquea la IP si supera el máximo.
func (rl *RateLimiter) RecordFailedAttempt(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rec, exists := rl.records[ip]
	if !exists {
		rec = &ipRecord{}
		rl.records[ip] = rec
	}

	if now.Sub(rec.lastAttempt) > rl.window {
		rec.attempts = 1
	} else {
		rec.attempts++
	}
	rec.lastAttempt = now

	if rec.attempts >= rl.maxAttempts {
		rec.blockedTill = now.Add(rl.blockTimeout)
	}
}

// Reset limpia el historial de una IP tras un login exitoso.
func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.records, ip)
}

// ExtractIP obtiene la IP remota del cliente sanitizada.
func ExtractIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// LimitHandler middleware que bloquea con 429 si se excede el rate limit.
func (rl *RateLimiter) LimitHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ExtractIP(r)
		if !rl.Check(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Demasiados intentos de acceso fallidos. Por favor espere antes de reintentar.",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultVal
}
