package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type pingerFalso struct{ err error }

func (p pingerFalso) Ping(context.Context) error { return p.err }

func TestHealth(t *testing.T) {
	casos := []struct {
		nombre string
		err    error
		status int
	}{
		{"mongo responde", nil, http.StatusOK},
		{"mongo caido", errors.New("sin conexion"), http.StatusServiceUnavailable},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			mux := http.NewServeMux()
			Register(mux, pingerFalso{c.err})

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
			if rec.Code != c.status {
				t.Errorf("status = %d, se esperaba %d", rec.Code, c.status)
			}
		})
	}
}

func TestRutaDesconocidaDa404(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, pingerFalso{})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/no-existe", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, se esperaba 404", rec.Code)
	}
}
