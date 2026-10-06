import uuid
from datetime import datetime, timedelta, timezone

import jwt
import pytest

from app.core.config import settings
from app.core.security import (
    InvalidTokenError,
    create_access_token,
    decode_access_token,
    hash_password,
    verify_password,
)


def test_hash_y_verificacion_de_contrasena():
    h = hash_password("clave123")
    assert h != "clave123"
    assert verify_password("clave123", h)
    assert not verify_password("otra", h)


def test_token_cumple_contrato_con_el_motor():
    uid = uuid.uuid4()
    token = create_access_token(uid, "player")
    header = jwt.get_unverified_header(token)
    claims = jwt.decode(token, settings.jwt_secret, algorithms=["HS256"])
    assert header["alg"] == "HS256"
    assert claims["sub"] == str(uid)
    assert claims["rol"] == "player"
    assert "exp" in claims
    assert decode_access_token(token) == (uid, "player")


def test_rol_desconocido_no_se_emite():
    with pytest.raises(ValueError):
        create_access_token(uuid.uuid4(), "superusuario")


@pytest.mark.parametrize(
    "token",
    [
        jwt.encode({"sub": str(uuid.uuid4()), "rol": "player", "exp": datetime.now(timezone.utc) - timedelta(minutes=1)}, settings.jwt_secret, algorithm="HS256"),
        jwt.encode({"sub": str(uuid.uuid4()), "rol": "player", "exp": datetime.now(timezone.utc) + timedelta(minutes=5)}, "otro-secreto", algorithm="HS256"),
        jwt.encode({"sub": str(uuid.uuid4()), "rol": "player"}, settings.jwt_secret, algorithm="HS256"),
        jwt.encode({"sub": "no-es-uuid", "rol": "player", "exp": datetime.now(timezone.utc) + timedelta(minutes=5)}, settings.jwt_secret, algorithm="HS256"),
        jwt.encode({"sub": str(uuid.uuid4()), "rol": "hacker", "exp": datetime.now(timezone.utc) + timedelta(minutes=5)}, settings.jwt_secret, algorithm="HS256"),
        jwt.encode({"sub": str(uuid.uuid4()), "rol": "player", "exp": datetime.now(timezone.utc) + timedelta(minutes=5)}, None, algorithm="none"),
        "basura",
    ],
    ids=["vencido", "otra-firma", "sin-exp", "sub-no-uuid", "rol-raro", "alg-none", "basura"],
)
def test_tokens_invalidos_se_rechazan(token):
    with pytest.raises(InvalidTokenError):
        decode_access_token(token)


def test_origenes_cors_se_leen_como_lista(monkeypatch):
    from app.core.config import Settings

    monkeypatch.setenv("ALLOWED_ORIGINS", "http://a.com, http://b.com")
    assert Settings().allowed_origins == ["http://a.com", "http://b.com"]
