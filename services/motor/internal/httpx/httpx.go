// Package httpx reune utilidades HTTP compartidas por los modulos del motor.
package httpx

import (
	"encoding/json"
	"net/http"
)

// EscribirJSON responde con el cuerpo serializado como JSON.
func EscribirJSON(w http.ResponseWriter, status int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(cuerpo)
}

// Origenes es el conjunto de origenes del front permitidos (ALLOWED_ORIGINS).
type Origenes map[string]struct{}

func NuevosOrigenes(lista []string) Origenes {
	o := make(Origenes, len(lista))
	for _, s := range lista {
		o[s] = struct{}{}
	}
	return o
}

func (o Origenes) Permite(origen string) bool {
	_, ok := o[origen]
	return ok
}

// CORS permite que el front (servido en otro puerto) llame al motor
// enviando la cookie de sesion. Con credenciales el navegador no acepta
// "Access-Control-Allow-Origin: *", por eso se responde con el origen exacto
// y solo si esta en la lista.
func CORS(origenes Origenes, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origen := r.Header.Get("Origin")
		if origen != "" && origenes.Permite(origen) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origen)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
			h.Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
