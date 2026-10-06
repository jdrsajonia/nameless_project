"""Utilidades de seguridad: hash de contrasenas y JWT de sesion.

El contrato del JWT es el mismo que valida el motor de apuestas en Go
(services/motor/internal/auth/contrato.go):

  - Algoritmo: HS256 con el secreto compartido JWT_SECRET.
  - Viaja en una cookie HttpOnly llamada "session".
  - Claims: "sub" = id del usuario (UUID en texto), "rol" = "player" | "admin",
    "exp" obligatorio.

Si cambia algo del contrato, hay que cambiarlo aqui y en contrato.go.
"""
import uuid
from datetime import datetime, timedelta, timezone

import bcrypt
import jwt

from app.core.config import settings
from app.models import ROLE_ADMIN, ROLE_PLAYER

CLAIM_USER_ID = "sub"
CLAIM_ROLE = "rol"
VALID_ROLES = {ROLE_PLAYER, ROLE_ADMIN}


class InvalidTokenError(Exception):
    """El token esta vencido, mal firmado o no cumple el contrato."""


# ---------------------------------------------------------------------------
# Contrasenas
# ---------------------------------------------------------------------------
def hash_password(password: str) -> str:
    return bcrypt.hashpw(password.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")


def verify_password(password: str, password_hash: str) -> bool:
    try:
        return bcrypt.checkpw(password.encode("utf-8"), password_hash.encode("utf-8"))
    except ValueError:
        return False


# ---------------------------------------------------------------------------
# JWT
# ---------------------------------------------------------------------------
def create_access_token(user_id: uuid.UUID | str, role: str) -> str:
    if role not in VALID_ROLES:
        raise ValueError(f"rol desconocido: {role!r}")
    now = datetime.now(timezone.utc)
    payload = {
        CLAIM_USER_ID: str(user_id),
        CLAIM_ROLE: role,
        "iat": now,
        "exp": now + timedelta(minutes=settings.jwt_expire_minutes),
    }
    return jwt.encode(payload, settings.jwt_secret, algorithm=settings.jwt_algorithm)


def decode_access_token(token: str) -> tuple[uuid.UUID, str]:
    """Valida el token y devuelve (user_id, rol)."""
    try:
        claims = jwt.decode(
            token,
            settings.jwt_secret,
            algorithms=[settings.jwt_algorithm],  # fija el algoritmo: rechaza "none"
            options={"require": ["exp", CLAIM_USER_ID, CLAIM_ROLE]},
        )
        user_id = uuid.UUID(claims[CLAIM_USER_ID])
    except (jwt.PyJWTError, ValueError, TypeError) as exc:
        raise InvalidTokenError(str(exc)) from exc

    role = claims[CLAIM_ROLE]
    if role not in VALID_ROLES:
        raise InvalidTokenError(f"rol desconocido: {role!r}")
    return user_id, role
