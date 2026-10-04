package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const origenFront = "http://localhost:5173"

// identificadorPrueba toma el user_id del parametro ?u= (en produccion sale
// del JWT). Sin parametro rechaza la conexion.
var identificadorPrueba = IdentificadorFunc(func(r *http.Request) (string, error) {
	if u := r.URL.Query().Get("u"); u != "" {
		return u, nil
	}
	return "", errors.New("sin sesion")
})

func soloFront(origen string) bool { return origen == origenFront }

func iniciar(t *testing.T) (*Hub, *httptest.Server) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	hub := NuevoHub()
	go hub.Run(ctx)

	mux := http.NewServeMux()
	Register(mux, hub, identificadorPrueba, soloFront)
	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
		cancel()
	})
	return hub, srv
}

func conectar(t *testing.T, srv *httptest.Server, userID string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?u=" + userID
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {origenFront}})
	if err != nil {
		t.Fatalf("no se pudo conectar como %q: %v", userID, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// esperarConectados espera a que el hub registre n conexiones: el Dial del
// cliente puede volver antes de que el hub procese el registro.
func esperarConectados(t *testing.T, hub *Hub, n int) {
	t.Helper()
	limite := time.Now().Add(2 * time.Second)
	for hub.Conectados() != n {
		if time.Now().After(limite) {
			t.Fatalf("conectados = %d, se esperaba %d", hub.Conectados(), n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func leer(t *testing.T, conn *websocket.Conn) Mensaje {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, datos, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("no llego mensaje: %v", err)
	}
	var m Mensaje
	if err := json.Unmarshal(datos, &m); err != nil {
		t.Fatalf("mensaje no es JSON valido: %s", datos)
	}
	return m
}

func TestDifundirLlegaATodos(t *testing.T) {
	hub, srv := iniciar(t)
	a := conectar(t, srv, "1")
	b := conectar(t, srv, "2")
	esperarConectados(t, hub, 2)

	if err := hub.Difundir(Mensaje{Type: TipoPosiciones, Payload: []string{"VER", "HAM"}}); err != nil {
		t.Fatal(err)
	}
	for _, conn := range []*websocket.Conn{a, b} {
		if m := leer(t, conn); m.Type != TipoPosiciones {
			t.Errorf("type = %q, se esperaba %q", m.Type, TipoPosiciones)
		}
	}
}

func TestEnviarASoloLlegaAlUsuario(t *testing.T) {
	hub, srv := iniciar(t)
	pestana1 := conectar(t, srv, "7")
	pestana2 := conectar(t, srv, "7")
	otro := conectar(t, srv, "8")
	esperarConectados(t, hub, 3)

	if err := hub.EnviarA("7", Mensaje{Type: TipoResultadoApuesta, Payload: map[string]any{"resultado": "ganada"}}); err != nil {
		t.Fatal(err)
	}
	for _, conn := range []*websocket.Conn{pestana1, pestana2} {
		if m := leer(t, conn); m.Type != TipoResultadoApuesta {
			t.Errorf("type = %q, se esperaba %q", m.Type, TipoResultadoApuesta)
		}
	}

	// El otro usuario no debe recibir el resultado: el siguiente mensaje que
	// le llega es la difusion.
	if err := hub.Difundir(Mensaje{Type: TipoMercadoAbierto}); err != nil {
		t.Fatal(err)
	}
	if m := leer(t, otro); m.Type != TipoMercadoAbierto {
		t.Errorf("el usuario 8 recibio %q", m.Type)
	}
}

func TestDesconexionLiberaAlCliente(t *testing.T) {
	hub, srv := iniciar(t)
	conn := conectar(t, srv, "3")
	esperarConectados(t, hub, 1)

	_ = conn.Close()
	esperarConectados(t, hub, 0)

	// Enviar a un usuario desconectado no es error ni bloquea.
	if err := hub.EnviarA("3", Mensaje{Type: TipoResultadoApuesta}); err != nil {
		t.Errorf("error inesperado: %v", err)
	}
}

func TestRechazos(t *testing.T) {
	_, srv := iniciar(t)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	casos := []struct {
		nombre, query, origen string
		status                int
	}{
		{"sin sesion", "", origenFront, http.StatusUnauthorized},
		{"origen ajeno", "?u=1", "http://sitio-malicioso.com", http.StatusForbidden},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, resp, err := websocket.DefaultDialer.Dial(url+c.query, http.Header{"Origin": {c.origen}})
			if err == nil {
				t.Fatal("se esperaba que la conexion fallara")
			}
			if resp == nil || resp.StatusCode != c.status {
				t.Errorf("respuesta = %v, se esperaba status %d", resp, c.status)
			}
		})
	}
}

func TestHubDetenido(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := NuevoHub()
	hecho := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(hecho)
	}()
	cancel()
	<-hecho

	if err := hub.Difundir(Mensaje{Type: TipoPosiciones}); !errors.Is(err, ErrHubDetenido) {
		t.Errorf("Difundir = %v, se esperaba ErrHubDetenido", err)
	}
	if err := hub.EnviarA("1", Mensaje{Type: TipoResultadoApuesta}); !errors.Is(err, ErrHubDetenido) {
		t.Errorf("EnviarA = %v, se esperaba ErrHubDetenido", err)
	}
	if err := hub.EnviarA("", Mensaje{}); err == nil {
		t.Error("EnviarA sin user_id deberia fallar")
	}
}
