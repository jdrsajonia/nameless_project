"""Esquemas compartidos por varios routers."""
from pydantic import BaseModel


class HealthOut(BaseModel):
    service: str
    status: str
    postgres: bool


class MessageOut(BaseModel):
    service: str
    message: str
