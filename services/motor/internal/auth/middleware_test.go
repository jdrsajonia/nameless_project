package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/arquisoft/motor/internal/ws"
)

// Si Autenticador deja de cumplir ws.Identificador, esto no compila.
var _ ws.Identificador = (*Autenticador)(nil)

// peticion arma un GET con la cookie de sesion del rol dado ("" = sin cookie).
func peticion(t *testing.T, rol string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if rol != "" {
		c := claimsValidos()
		c[ClaimRol] = rol
		r.AddCookie(&http.Cookie{Name: NombreCookie, Value: firmar(t, jwt.SigningMethodHS256, secretoPrueba, c)})
	}
	return r
}

// handlerOK responde 200 y guarda la sesion que vio en el contexto.
func handlerOK(vista *Sesion) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*vista, _ = SesionDe(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

func TestExigirSesion(t *testing.T) {
	a := NuevoAutenticador(validador(t))

	t.Run("sin cookie da 401", func(t *testing.T) {
		var vista Sesion
		rec := httptest.NewRecorder()
		a.ExigirSesion(handlerOK(&vista)).ServeHTTP(rec, peticion(t, ""))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, se esperaba 401", rec.Code)
		}
	})

	t.Run("token invalido da 401", func(t *testing.T) {
		var vista Sesion
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: NombreCookie, Value: "token-falso"})
		rec := httptest.NewRecorder()
		a.ExigirSesion(handlerOK(&vista)).ServeHTTP(rec, r)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, se esperaba 401", rec.Code)
		}
	})

	t.Run("con sesion deja user_id y rol en el contexto", func(t *testing.T) {
		var vista Sesion
		rec := httptest.NewRecorder()
		a.ExigirSesion(handlerOK(&vista)).ServeHTTP(rec, peticion(t, RolJugador))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, se esperaba 200", rec.Code)
		}
		if vista.UserID != userPrueba || vista.Rol != RolJugador {
			t.Errorf("sesion en el contexto = %+v", vista)
		}
	})
}

func TestExigirRol(t *testing.T) {
	a := NuevoAutenticador(validador(t))
	casos := []struct {
		nombre, rolUsuario string
		status             int
	}{
		{"admin entra", RolAdmin, http.StatusOK},
		{"jugador da 403", RolJugador, http.StatusForbidden},
		{"sin sesion da 401", "", http.StatusUnauthorized},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var vista Sesion
			rec := httptest.NewRecorder()
			a.ExigirRol(RolAdmin, handlerOK(&vista)).ServeHTTP(rec, peticion(t, c.rolUsuario))
			if rec.Code != c.status {
				t.Errorf("status = %d, se esperaba %d", rec.Code, c.status)
			}
		})
	}
}

func TestIdentificar(t *testing.T) {
	a := NuevoAutenticador(validador(t))

	userID, err := a.Identificar(peticion(t, RolJugador))
	if err != nil || userID != userPrueba {
		t.Errorf("Identificar = %q, %v", userID, err)
	}

	if _, err := a.Identificar(peticion(t, "")); !errors.Is(err, ErrSinSesion) {
		t.Errorf("sin cookie: error = %v, se esperaba ErrSinSesion", err)
	}
}

func TestSesionDeSinMiddleware(t *testing.T) {
	if _, ok := SesionDe(httptest.NewRequest(http.MethodGet, "/", nil).Context()); ok {
		t.Error("sin middleware no deberia haber sesion")
	}
}
