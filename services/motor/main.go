// Motor de apuestas - Prototipo 1 (ArquiSoft).
//
// Esqueleto base (HU-10). Arranca un servidor HTTP con:
//   - GET /health : healthcheck (servicio vivo + MongoDB responde)
//   - GET /ws     : endpoint WebSocket base (hub) para datos en vivo
//
// La logica de negocio (ingesta OpenF1, mercados, apuestas, liquidacion)
// se construye encima de esta base en tareas posteriores.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type config struct {
	mongoURI       string
	mongoDB        string
	httpPort       string
	jwtSecret      string
	pagosBaseURL   string
	bettingWindow  string
	observationWin string
}

func loadConfig() config {
	return config{
		mongoURI:       getenv("MONGO_URI", "mongodb://arqui:arqui_pass@localhost:27017/arqui_motor?authSource=admin"),
		mongoDB:        getenv("MONGO_DB", "arqui_motor"),
		httpPort:       getenv("HTTP_PORT", "8080"),
		jwtSecret:      getenv("JWT_SECRET", "dev-secret"),
		pagosBaseURL:   getenv("PAGOS_BASE_URL", "http://localhost:8000"),
		bettingWindow:  getenv("BETTING_WINDOW_SECONDS", "30"),
		observationWin: getenv("OBSERVATION_WINDOW_SECONDS", "60"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type server struct {
	cfg   config
	mongo *mongo.Client
	hub   *Hub
}

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.mongoURI))
	if err != nil {
		log.Fatalf("no se pudo conectar a MongoDB: %v", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("MongoDB no responde al ping: %v", err)
	}
	log.Println("conectado a MongoDB")

	hub := newHub()
	go hub.run()

	s := &server{cfg: cfg, mongo: client, hub: hub}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/", s.handleRoot)

	addr := ":" + cfg.httpPort
	log.Printf("motor escuchando en %s", addr)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatalf("servidor HTTP termino: %v", err)
	}
}

// withCORS agrega cabeceras CORS a las respuestas REST para que el front
// (servido en otro origen/puerto) pueda consumir la API desde el navegador.
// El WebSocket no pasa por aqui porque no esta sujeto a la politica CORS.
// Para el prototipo se permite cualquier origen; restringir en entregas futuras.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	mongoOK := s.mongo.Ping(ctx, readpref.Primary()) == nil

	writeJSON(w, http.StatusOK, map[string]any{
		"service": "motor",
		"status":  "ok",
		"mongo":   mongoOK,
	})
}

func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "motor",
		"message": "Motor de apuestas - Prototipo 1",
	})
}

// upgrader para la conexion WebSocket. En el prototipo se acepta cualquier
// origen; mas adelante se valida el JWT al abrir la conexion (RNF-06).
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error al abrir WebSocket: %v", err)
		return
	}
	client := &Client{hub: s.hub, conn: conn, send: make(chan []byte, 16)}
	s.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
