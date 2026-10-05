// Package ws mantiene las conexiones WebSocket de los jugadores y les envia
// mensajes en vivo: a todos (posiciones, estado del mercado) o a un usuario
// concreto (resultado de su apuesta).
//
// Todo el estado del hub lo toca una sola goroutine (Run); el resto del
// motor solo le habla por canales, asi que no hace falta ningun mutex.
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

// ErrHubDetenido se devuelve si se intenta enviar despues de apagar el hub.
var ErrHubDetenido = errors.New("hub de WebSocket detenido")

type envio struct {
	userID string
	datos  []byte
}

type Hub struct {
	clientes map[*cliente]bool
	// porUsuario permite enviar a un usuario concreto. Un usuario puede
	// tener varias conexiones (p. ej. dos pestañas abiertas).
	porUsuario map[string]map[*cliente]bool

	registrar chan *cliente
	retirar   chan *cliente
	difundir  chan []byte
	directo   chan envio
	consultar chan chan int
	terminado chan struct{}
}

func NuevoHub() *Hub {
	return &Hub{
		clientes:   make(map[*cliente]bool),
		porUsuario: make(map[string]map[*cliente]bool),
		registrar:  make(chan *cliente),
		retirar:    make(chan *cliente),
		difundir:   make(chan []byte),
		directo:    make(chan envio),
		consultar:  make(chan chan int),
		terminado:  make(chan struct{}),
	}
}

// Run atiende el hub hasta que se cancela ctx. Al terminar cierra todas las
// conexiones.
func (h *Hub) Run(ctx context.Context) {
	defer close(h.terminado)
	for {
		select {
		case <-ctx.Done():
			for c := range h.clientes {
				h.quitar(c)
			}
			return
		case c := <-h.registrar:
			h.agregar(c)
		case c := <-h.retirar:
			h.quitar(c)
		case datos := <-h.difundir:
			for c := range h.clientes {
				h.entregar(c, datos)
			}
		case e := <-h.directo:
			for c := range h.porUsuario[e.userID] {
				h.entregar(c, e.datos)
			}
		case resp := <-h.consultar:
			resp <- len(h.clientes)
		}
	}
}

// Difundir envia el mensaje a todos los clientes conectados.
func (h *Hub) Difundir(m Mensaje) error {
	datos, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("no se pudo serializar el mensaje %q: %w", m.Type, err)
	}
	select {
	case h.difundir <- datos:
		return nil
	case <-h.terminado:
		return ErrHubDetenido
	}
}

// EnviarA envia el mensaje solo a las conexiones del usuario. Si el usuario
// no esta conectado el mensaje se descarta (no es un error).
func (h *Hub) EnviarA(userID string, m Mensaje) error {
	if userID == "" {
		return errors.New("EnviarA necesita un user_id")
	}
	datos, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("no se pudo serializar el mensaje %q: %w", m.Type, err)
	}
	select {
	case h.directo <- envio{userID: userID, datos: datos}:
		return nil
	case <-h.terminado:
		return ErrHubDetenido
	}
}

// Conectados devuelve cuantas conexiones hay abiertas. Sirve para metricas
// en las pruebas de carga y en los tests.
func (h *Hub) Conectados() int {
	resp := make(chan int, 1)
	select {
	case h.consultar <- resp:
		return <-resp
	case <-h.terminado:
		return 0
	}
}

func (h *Hub) entrar(c *cliente) error {
	select {
	case h.registrar <- c:
		return nil
	case <-h.terminado:
		return ErrHubDetenido
	}
}

func (h *Hub) salir(c *cliente) {
	select {
	case h.retirar <- c:
	case <-h.terminado:
	}
}

func (h *Hub) agregar(c *cliente) {
	h.clientes[c] = true
	if c.userID != "" {
		if h.porUsuario[c.userID] == nil {
			h.porUsuario[c.userID] = make(map[*cliente]bool)
		}
		h.porUsuario[c.userID][c] = true
	}
	log.Printf("cliente WS conectado user_id=%q (total=%d)", c.userID, len(h.clientes))
}

func (h *Hub) quitar(c *cliente) {
	if !h.clientes[c] {
		return
	}
	delete(h.clientes, c)
	if conexiones := h.porUsuario[c.userID]; conexiones != nil {
		delete(conexiones, c)
		if len(conexiones) == 0 {
			delete(h.porUsuario, c.userID)
		}
	}
	// Cerrar send hace que writePump cierre la conexion.
	close(c.send)
	log.Printf("cliente WS desconectado user_id=%q (total=%d)", c.userID, len(h.clientes))
}

// entregar no bloquea: si el buffer del cliente esta lleno (cliente lento o
// colgado) se desconecta para no frenar a los demas (RNF-04). El front se
// reconecta solo (HU-05).
func (h *Hub) entregar(c *cliente, datos []byte) {
	select {
	case c.send <- datos:
	default:
		log.Printf("cliente WS lento, se desconecta user_id=%q", c.userID)
		h.quitar(c)
	}
}
