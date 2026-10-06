"""Healthcheck que usa docker compose (ver docker-compose.yml)."""
from fastapi import APIRouter, Response, status

from app.db import ping_db
from app.schemas.common import HealthOut

router = APIRouter(tags=["health"])


@router.get("/health", response_model=HealthOut)
def health(response: Response) -> HealthOut:
    """200 si el servicio y PostgreSQL responden; 503 si la BD no responde."""
    db_ok = ping_db()
    if not db_ok:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE
    return HealthOut(service="pagos", status="ok" if db_ok else "degraded", postgres=db_ok)
