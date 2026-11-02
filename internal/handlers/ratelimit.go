package handlers

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tiar/telegram-sender/internal/services"
	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientLimiter
	r       rate.Limit
	b       int
	logs    *services.LogService
}

// NewRateLimiter membuat in-memory rate limiter per-client (berdasarkan API key atau IP).
// r = request per second yang diperbolehkan.
// b = burst allowance.
func NewRateLimiter(r rate.Limit, b int, logs *services.LogService) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientLimiter),
		r:       r,
		b:       b,
		logs:    logs,
	}

	// Background worker untuk membersihkan klien yang tidak aktif setiap 3 menit
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			rl.mu.Lock()
			for key, client := range rl.clients {
				if time.Since(client.lastSeen) > 5*time.Minute {
					delete(rl.clients, key)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, exists := rl.clients[key]
	if !exists {
		client = &clientLimiter{
			limiter:  rate.NewLimiter(rl.r, rl.b),
			lastSeen: time.Now(),
		}
		rl.clients[key] = client
		return client.limiter
	}

	client.lastSeen = time.Now()
	return client.limiter
}

func getClientIdentifier(r *http.Request) string {
	// 1. Prioritaskan API Key jika ada
	apiKey := r.Header.Get("X-Api-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}
	if apiKey != "" {
		return "key:" + apiKey
	}

	// 2. Fallback ke IP
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return "ip:" + strings.TrimSpace(parts[0])
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return "ip:" + strings.TrimSpace(realIP)
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return "ip:" + ip
	}
	return "ip:" + r.RemoteAddr
}

// GetClientIP mengekstrak IP pemanggil HTTP dengan mempertimbangkan reverse proxy
func GetClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return strings.TrimSpace(realIP)
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}
	return r.RemoteAddr
}

// Middleware membungkus handler HTTP dengan pengecekan rate limit
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := getClientIdentifier(r)
		limiter := rl.getLimiter(key)

		if !limiter.Allow() {
			if rl.logs != nil {
				rl.logs.WriteDetail("warn", "api.rate_limit", fmt.Sprintf("Rate limit terlampaui untuk %s", key), GetClientIP(r), "", nil)
			}
			w.Header().Set("Retry-After", "1")
			jsonErrorTyped(w, "Terlalu banyak permintaan (Rate limit exceeded). Harap perlambat pengiriman.", "rate_limit_exceeded", map[string]any{
				"retry_after_seconds": 1,
			}, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
