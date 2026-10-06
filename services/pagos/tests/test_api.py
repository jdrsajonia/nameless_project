from fastapi import APIRouter

from app.core.security import create_access_token
from app.deps import AdminUser, CurrentUser
from tests.conftest import requires_db

# Router de prueba para ejercitar las dependencias de autenticacion.
_probe = APIRouter()


@_probe.get("/_test/me")
def _me(user: CurrentUser):
    return {"id": str(user.id), "role": user.role}


@_probe.get("/_test/admin")
def _admin(user: AdminUser):
    return {"ok": True}


@requires_db
def test_health_y_tablas_creadas(client):
    r = client.get("/health")
    assert r.status_code == 200
    assert r.json() == {"service": "pagos", "status": "ok", "postgres": True}

    from sqlalchemy import inspect

    from app.db import engine

    assert {"users", "bets", "ledger_entries"} <= set(inspect(engine).get_table_names())


@requires_db
def test_cors_permite_al_front_con_cookies(client):
    r = client.options(
        "/health",
        headers={"Origin": "http://localhost:5173", "Access-Control-Request-Method": "GET"},
    )
    assert r.headers["access-control-allow-origin"] == "http://localhost:5173"
    assert r.headers["access-control-allow-credentials"] == "true"

    r = client.options(
        "/health",
        headers={"Origin": "http://malicioso.com", "Access-Control-Request-Method": "GET"},
    )
    assert "access-control-allow-origin" not in r.headers


@requires_db
def test_dependencias_de_sesion(client, make_user):
    client.app.include_router(_probe)
    player = make_user("player")
    admin = make_user("admin")

    assert client.get("/_test/me").status_code == 401
    client.cookies.set("session", "token-falso")
    assert client.get("/_test/me").status_code == 401

    client.cookies.set("session", create_access_token(player.id, player.role))
    r = client.get("/_test/me")
    assert r.status_code == 200 and r.json() == {"id": str(player.id), "role": "player"}
    assert client.get("/_test/admin").status_code == 403

    client.cookies.set("session", create_access_token(admin.id, admin.role))
    assert client.get("/_test/admin").status_code == 200
