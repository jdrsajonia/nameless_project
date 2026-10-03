"""Conexion a PostgreSQL con SQLAlchemy."""
from sqlalchemy import create_engine
from sqlalchemy.orm import DeclarativeBase, sessionmaker

from app.config import settings

engine = create_engine(settings.database_url, pool_pre_ping=True)
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)


class Base(DeclarativeBase):
    """Base declarativa para los modelos ORM."""


def get_db():
    """Dependencia de FastAPI que entrega una sesion de BD por request."""
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
