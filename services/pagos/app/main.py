"""Servicio Pagos y usuarios - Prototipo 1 (ArquiSoft).

Esqueleto base (HU-10). Expone /health para que docker compose verifique que
el servicio esta vivo y conectado a PostgreSQL. Las HU de negocio (registro,
login, compra de tokens, debito de apuesta, liquidacion) se construyen encima
de esta base en tareas posteriores.
"""
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from sqlalchemy import text

from app.config import settings
from app.db import engine


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Al arrancar: verificar que PostgreSQL responde.
    with engine.connect() as conn:
        conn.execute(text("SELECT 1"))
    yield


app = FastAPI(title="Pagos y usuarios", version="0.1.0", lifespan=lifespan)

# El front corre en otro origen (puerto distinto), por eso habilitamos CORS.
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # para el prototipo; restringir en entregas posteriores
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
def health():
    """Healthcheck: el servicio esta arriba y PostgreSQL responde."""
    try:
        with engine.connect() as conn:
            conn.execute(text("SELECT 1"))
        db_ok = True
    except Exception:
        db_ok = False
    return {"service": "pagos", "status": "ok", "postgres": db_ok}


@app.get("/")
def root():
    return {"service": "pagos", "message": "Pagos y usuarios - Prototipo 1"}
