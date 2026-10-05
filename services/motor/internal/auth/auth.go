// Package auth valida el JWT que emite Pagos y usuarios.
//
// El motor no le pregunta a Pagos en cada peticion: verifica la firma por su
// cuenta con el secreto compartido, asi el flujo en vivo no depende de Pagos.
// El resto del motor usa la interfaz ValidarToken, de modo que cambiar luego
// a llaves publica/privada u OAuth no toca los demas paquetes.
package auth

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrSinSesion: la peticion no trae la cookie de sesion.
	ErrSinSesion = errors.New("no hay sesion")
	// ErrTokenInvalido: el token esta vencido, mal firmado o mal formado.
	ErrTokenInvalido = errors.New("token invalido")
)

// Sesion es la identidad que se saca de un token valido.
type Sesion struct {
	UserID string
	Rol    string
}

// ValidarToken es lo que el resto del motor necesita de la autenticacion.
type ValidarToken interface {
	Validar(token string) (Sesion, error)
}

var formatoUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidadorHS256 valida tokens firmados con el secreto compartido.
type ValidadorHS256 struct {
	secreto []byte
	parser  *jwt.Parser
}

func NuevoValidador(secreto string) (*ValidadorHS256, error) {
	if secreto == "" {
		return nil, errors.New("el secreto del JWT no puede estar vacio")
	}
	return &ValidadorHS256{
		secreto: []byte(secreto),
		parser: jwt.NewParser(
			// Fijar el algoritmo evita que un token con "alg": "none" u otro
			// algoritmo pase la validacion.
			jwt.WithValidMethods([]string{Algoritmo}),
			// Un token sin exp nunca venceria: se exige.
			jwt.WithExpirationRequired(),
		),
	}, nil
}

// Validar revisa firma, algoritmo y vencimiento, y devuelve el user_id y el
// rol. Cualquier error envuelve ErrTokenInvalido.
func (v *ValidadorHS256) Validar(token string) (Sesion, error) {
	claims := jwt.MapClaims{}
	if _, err := v.parser.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return v.secreto, nil
	}); err != nil {
		return Sesion{}, fmt.Errorf("%w: %w", ErrTokenInvalido, err)
	}

	userID, _ := claims[ClaimUserID].(string)
	if !formatoUUID.MatchString(userID) {
		return Sesion{}, fmt.Errorf("%w: el claim %q no es un UUID", ErrTokenInvalido, ClaimUserID)
	}
	rol, _ := claims[ClaimRol].(string)
	if rol != RolJugador && rol != RolAdmin {
		return Sesion{}, fmt.Errorf("%w: rol %q desconocido", ErrTokenInvalido, rol)
	}
	return Sesion{UserID: userID, Rol: rol}, nil
}

// DesdeRequest lee el JWT de la cookie de sesion y lo valida.
func DesdeRequest(v ValidarToken, r *http.Request) (Sesion, error) {
	cookie, err := r.Cookie(NombreCookie)
	if err != nil || cookie.Value == "" {
		return Sesion{}, ErrSinSesion
	}
	return v.Validar(cookie.Value)
}
