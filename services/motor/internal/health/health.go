// Package health expone el healthcheck del motor que usa Docker Compose.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/arquisoft/motor/internal/httpx"
)

// Pinger es lo minimo que el healthcheck necesita de la base de datos.
type Pinger interface {
	Ping(ctx context.Context) error
}

func Register(mux *http.ServeMux, db Pinger) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		// Si MongoDB no responde el motor no puede servir mercados, asi que
		// se reporta como no saludable (503) para que el Compose lo note.
		if err := db.Ping(ctx); err != nil {
			httpx.EscribirJSON(w, http.StatusServiceUnavailable, map[string]any{
				"service": "motor",
				"status":  "degradado",
				"mongo":   false,
			})
			return
		}
		httpx.EscribirJSON(w, http.StatusOK, map[string]any{
			"service": "motor",
			"status":  "ok",
			"mongo":   true,
		})
	})

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		httpx.EscribirJSON(w, http.StatusOK, map[string]any{
			"service": "motor",
			"message": "Motor de apuestas - Prototipo 1",
		})
	})
}
