"""Configuracion del servicio Pagos y usuarios, leida de variables de entorno."""
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    database_url: str = "postgresql+psycopg://arqui:arqui_pass@localhost:5432/arqui_pagos"

    jwt_secret: str = "dev-secret"
    jwt_expire_minutes: int = 120

    cop_per_token: float = 1000.0

    admin_email: str = "admin@arqui.local"
    admin_password: str = "admin1234"
    admin_name: str = "Administrador"


settings = Settings()
