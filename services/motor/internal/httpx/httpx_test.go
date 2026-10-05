package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	origenes := NuevosOrigenes([]string{"http://localhost:5173"})
	h := CORS(origenes, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	casos := []struct {
		nombre, metodo, origen string
		status                 int
		allowOrigin            string
	}{
		{"origen permitido", http.MethodGet, "http://localhost:5173", http.StatusOK, "http://localhost:5173"},
		{"origen no permitido", http.MethodGet, "http://malo.com", http.StatusOK, ""},
		{"preflight permitido", http.MethodOptions, "http://localhost:5173", http.StatusNoContent, "http://localhost:5173"},
		{"sin origen", http.MethodGet, "", http.StatusOK, ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			req := httptest.NewRequest(c.metodo, "/", nil)
			if c.origen != "" {
				req.Header.Set("Origin", c.origen)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != c.status {
				t.Errorf("status = %d, se esperaba %d", rec.Code, c.status)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != c.allowOrigin {
				t.Errorf("Allow-Origin = %q, se esperaba %q", got, c.allowOrigin)
			}
			if c.allowOrigin != "" && rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Error("falta Allow-Credentials")
			}
		})
	}
}
