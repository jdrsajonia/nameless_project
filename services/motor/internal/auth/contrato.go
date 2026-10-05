package auth

// Contrato del JWT con Pagos y usuarios (HU-02). Pagos emite el token y el
// motor solo lo valida con el secreto compartido JWT_SECRET (RNF-06).
//
// PENDIENTE: valores por defecto, falta confirmarlos con Misael (dueño de
// Pagos). Si alguno cambia, se cambia solo en este archivo.
const (
	// Algoritmo de firma. Cualquier otro (incluido "none") se rechaza.
	Algoritmo = "HS256"

	// Cookie HttpOnly donde Pagos guarda el JWT al iniciar sesion.
	NombreCookie = "session"

	// ClaimUserID es el id del usuario: un UUID como string.
	ClaimUserID = "sub"
	// ClaimRol es el rol del usuario; debe ser RolJugador o RolAdmin.
	ClaimRol = "rol"

	// Valores del rol, como los define el modelo de datos de Pagos.
	RolJugador = "player"
	RolAdmin   = "admin"
)
