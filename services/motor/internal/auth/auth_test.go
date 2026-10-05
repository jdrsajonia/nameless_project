package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	secretoPrueba = "secreto-de-prueba-con-mas-de-32-bytes!!"
	userPrueba    = "3f2b8c1e-9a4d-4e6f-8b1a-2c3d4e5f6a7b"
)

// firmar crea un token como lo emitiria Pagos.
func firmar(t *testing.T, metodo jwt.SigningMethod, secreto string, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(metodo, claims).SignedString([]byte(secreto))
	if err != nil {
		t.Fatalf("no se pudo firmar: %v", err)
	}
	return token
}

func claimsValidos() jwt.MapClaims {
	return jwt.MapClaims{
		ClaimUserID: userPrueba,
		ClaimRol:    RolJugador,
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
}

func validador(t *testing.T) *ValidadorHS256 {
	t.Helper()
	v, err := NuevoValidador(secretoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestTokenValido(t *testing.T) {
	token := firmar(t, jwt.SigningMethodHS256, secretoPrueba, claimsValidos())

	s, err := validador(t).Validar(token)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if s.UserID != userPrueba || s.Rol != RolJugador {
		t.Errorf("sesion = %+v", s)
	}
}

func TestTokensRechazados(t *testing.T) {
	con := func(cambiar func(jwt.MapClaims)) jwt.MapClaims {
		c := claimsValidos()
		cambiar(c)
		return c
	}
	casos := []struct {
		nombre string
		token  string
	}{
		{"vencido", firmar(t, jwt.SigningMethodHS256, secretoPrueba,
			con(func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() }))},
		{"firma mala", firmar(t, jwt.SigningMethodHS256, "otro-secreto", claimsValidos())},
		{"algoritmo distinto", firmar(t, jwt.SigningMethodHS384, secretoPrueba, claimsValidos())},
		{"sin exp", firmar(t, jwt.SigningMethodHS256, secretoPrueba,
			con(func(c jwt.MapClaims) { delete(c, "exp") }))},
		{"sub no es UUID", firmar(t, jwt.SigningMethodHS256, secretoPrueba,
			con(func(c jwt.MapClaims) { c[ClaimUserID] = "42" }))},
		{"rol desconocido", firmar(t, jwt.SigningMethodHS256, secretoPrueba,
			con(func(c jwt.MapClaims) { c[ClaimRol] = "superusuario" }))},
		{"basura", "esto.no.es-un-jwt"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := validador(t).Validar(c.token)
			if !errors.Is(err, ErrTokenInvalido) {
				t.Errorf("error = %v, se esperaba ErrTokenInvalido", err)
			}
		})
	}
}

func TestAlgoritmoNone(t *testing.T) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claimsValidos()).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validador(t).Validar(token); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("un token sin firma no debe pasar: %v", err)
	}
}

func TestDesdeRequest(t *testing.T) {
	v := validador(t)

	t.Run("con cookie valida", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: NombreCookie, Value: firmar(t, jwt.SigningMethodHS256, secretoPrueba, claimsValidos())})
		if s, err := DesdeRequest(v, r); err != nil || s.UserID != userPrueba {
			t.Errorf("sesion = %+v, err = %v", s, err)
		}
	})

	t.Run("sin cookie", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if _, err := DesdeRequest(v, r); !errors.Is(err, ErrSinSesion) {
			t.Errorf("error = %v, se esperaba ErrSinSesion", err)
		}
	})

	t.Run("cookie con otro nombre", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "token", Value: firmar(t, jwt.SigningMethodHS256, secretoPrueba, claimsValidos())})
		if _, err := DesdeRequest(v, r); !errors.Is(err, ErrSinSesion) {
			t.Errorf("error = %v, se esperaba ErrSinSesion", err)
		}
	})
}

func TestSecretoVacio(t *testing.T) {
	if _, err := NuevoValidador(""); err == nil {
		t.Error("un secreto vacio deberia fallar")
	}
}
