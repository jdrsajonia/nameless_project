package apuestas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RutaDebito es la "puerta" de Pagos que usa el motor. PENDIENTE: confirmar
// la ruta, el cuerpo y los codigos con Juan Diego. Si cambia, se cambia aqui.
const RutaDebito = "/internal/debit"

// PagosHTTP llama al servicio Pagos por REST.
//
// Codigos esperados de Pagos (contrato propuesto):
//
//	200/201 -> debitado
//	409     -> esa clave de idempotencia ya se proceso: se trata como exito
//	422     -> saldo insuficiente
//	404     -> el usuario no existe
//	otro    -> Pagos no esta disponible
type PagosHTTP struct {
	BaseURL string
	Cliente *http.Client
}

func NuevoPagosHTTP(baseURL string) *PagosHTTP {
	return &PagosHTTP{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Cliente: &http.Client{Timeout: 5 * time.Second},
	}
}

func (p *PagosHTTP) Debitar(ctx context.Context, d Debito) error {
	cuerpo, err := json.Marshal(d)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+RutaDebito, bytes.NewReader(cuerpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", d.IdempotencyKey)

	resp, err := p.Cliente.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPagosNoDisponible, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusConflict:
		return nil
	case http.StatusUnprocessableEntity:
		return ErrSaldoInsuficiente
	case http.StatusNotFound:
		return ErrUsuarioNoExiste
	default:
		return fmt.Errorf("%w: respondio %d", ErrPagosNoDisponible, resp.StatusCode)
	}
}

// PagosMock simula a Pagos mientras el endpoint real no exista. Cada usuario
// empieza con SaldoInicial tokens, el saldo nunca baja de 0 y repetir una
// clave de idempotencia no debita dos veces. Se activa con PAGOS_MOCK=true.
type PagosMock struct {
	SaldoInicial int64

	mu      sync.Mutex
	saldos  map[string]int64
	hechas  map[string]bool
	Debitos []Debito // historial, util para pruebas
}

func NuevoPagosMock(saldoInicial int64) *PagosMock {
	return &PagosMock{
		SaldoInicial: saldoInicial,
		saldos:       map[string]int64{},
		hechas:       map[string]bool{},
	}
}

func (m *PagosMock) Debitar(_ context.Context, d Debito) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.hechas[d.IdempotencyKey] {
		return nil
	}
	saldo, existe := m.saldos[d.UserID]
	if !existe {
		saldo = m.SaldoInicial
	}
	if saldo < d.Amount {
		return ErrSaldoInsuficiente
	}
	m.saldos[d.UserID] = saldo - d.Amount
	m.hechas[d.IdempotencyKey] = true
	m.Debitos = append(m.Debitos, d)
	return nil
}

// Saldo devuelve el saldo simulado de un usuario.
func (m *PagosMock) Saldo(userID string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.saldos[userID]; ok {
		return s
	}
	return m.SaldoInicial
}
