"""Dependencias reutilizables de FastAPI (BD y usuario autenticado).

Uso en un router:

    from app.deps import DbSession, CurrentUser, AdminUser

    @router.get("/me")
    def me(user: CurrentUser):
        ...
"""
from typing import Annotated

from fastapi import Cookie, Depends, HTTPException, status
from sqlalchemy.orm import Session

from app.core.config import settings
from app.core.security import InvalidTokenError, decode_access_token
from app.db import get_db
from app.models import ROLE_ADMIN, User

DbSession = Annotated[Session, Depends(get_db)]


def get_current_user(
    db: DbSession,
    session: Annotated[str | None, Cookie(alias=settings.session_cookie_name)] = None,
) -> User:
    """Lee el JWT de la cookie de sesion y devuelve el usuario. 401 si no hay sesion valida."""
    if not session:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="No hay sesion")
    try:
        user_id, _ = decode_access_token(session)
    except InvalidTokenError:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Sesion invalida o vencida")

    user = db.get(User, user_id)
    if user is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="El usuario ya no existe")
    return user


CurrentUser = Annotated[User, Depends(get_current_user)]


def require_admin(user: CurrentUser) -> User:
    if user.role != ROLE_ADMIN:
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Solo el administrador")
    return user


AdminUser = Annotated[User, Depends(require_admin)]
