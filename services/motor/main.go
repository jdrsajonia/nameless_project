// Motor de apuestas - Prototipo 1 (ArquiSoft).
//
// main solo arma las piezas y arranca el servidor. Cada modulo vive en su
// propio paquete bajo internal/ y registra sus rutas con Register(mux, ...),
// para que cada HU se agregue sin editar el codigo de las demas.
//
//   - GET /health : healthcheck (503 si MongoDB no responde)
//   - GET /ws     : WebSocket para datos en vivo (exige la cookie de sesion)
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arquisoft/motor/internal/auth"
	"github.com/arquisoft/motor/internal/config"
	"github.com/arquisoft/motor/internal/health"
	"github.com/arquisoft/motor/internal/httpx"
	"github.com/arquisoft/motor/internal/mongodb"
	"github.com/arquisoft/motor/internal/ws"
)

func main() {
	cfg, err := config.Cargar()
	if err != nil {
		log.Fatalf("configuracion invalida: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctxConexion, cancel := context.WithTimeout(ctx, 10*time.Second)
	db, err := mongodb.Conectar(ctxConexion, cfg.MongoURI, cfg.MongoDB)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("conectado a MongoDB (base %s)", cfg.MongoDB)

	hub := ws.NuevoHub()
	go hub.Run(ctx)

	origenes := httpx.NuevosOrigenes(cfg.OrigenesPermitidos)

	// El JWT lo emite Pagos; el motor solo verifica la firma con el secreto
	// compartido (RNF-06). Contrato en internal/auth/contrato.go.
	validador, err := auth.NuevoValidador(cfg.JWTSecret)
	if err != nil {
		log.Fatal(err)
	}
	autenticador := auth.NuevoAutenticador(validador)

	mux := http.NewServeMux()
	health.Register(mux, db)
	ws.Register(mux, hub, autenticador, origenes.Permite)
	// Aqui se registran los modulos de cada HU, p. ej.:
	//   posiciones.Register(mux, db.DB, hub)                 // HU-05
	//   mercados.Register(mux, db.DB, hub, autenticador, cfg) // HU-11

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           httpx.CORS(origenes, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		log.Println("apagando el motor...")
		ctxApagado, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctxApagado)
	}()

	log.Printf("motor escuchando en %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("servidor HTTP termino: %v", err)
	}

	ctxCierre, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Cerrar(ctxCierre); err != nil {
		log.Printf("error al cerrar MongoDB: %v", err)
	}
}
