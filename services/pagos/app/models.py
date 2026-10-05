"""Modelos ORM de Pagos y usuarios. Ver documentation/modelo-datos-pagos.pdf."""
import uuid
from datetime import datetime
from decimal import Decimal

from sqlalchemy import BigInteger, CheckConstraint, DateTime, ForeignKey, Numeric, String, Uuid, func
from sqlalchemy.orm import Mapped, mapped_column

from app.db import Base

ROLE_PLAYER = "player"
ROLE_ADMIN = "admin"

BET_PENDING = "pending"
BET_WON = "won"
BET_LOST = "lost"

LEDGER_PURCHASE = "purchase"
LEDGER_BET_DEBIT = "bet_debit"
LEDGER_BET_SETTLEMENT = "bet_settlement"
LEDGER_COMPLETED = "completed"


class User(Base):
    __tablename__ = "users"
    __table_args__ = (
        CheckConstraint(f"role IN ('{ROLE_PLAYER}', '{ROLE_ADMIN}')", name="ck_users_role"),
        CheckConstraint("balance >= 0", name="ck_users_balance_non_negative"),
    )

    id: Mapped[uuid.UUID] = mapped_column(Uuid, primary_key=True, default=uuid.uuid4, server_default=func.gen_random_uuid())
    email: Mapped[str] = mapped_column(String(255), unique=True, nullable=False)
    name: Mapped[str] = mapped_column(String(100), nullable=False)
    password_hash: Mapped[str] = mapped_column(String(255), nullable=False)
    role: Mapped[str] = mapped_column(String(20), nullable=False, default=ROLE_PLAYER, server_default=ROLE_PLAYER)
    balance: Mapped[int] = mapped_column(BigInteger, nullable=False, default=0, server_default="0")
    points: Mapped[Decimal] = mapped_column(Numeric(12, 1), nullable=False, default=Decimal("0"), server_default="0")
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False, server_default=func.now())


class Bet(Base):
    __tablename__ = "bets"
    __table_args__ = (
        CheckConstraint("amount > 0", name="ck_bets_amount_positive"),
        CheckConstraint(f"status IN ('{BET_PENDING}', '{BET_WON}', '{BET_LOST}')", name="ck_bets_status"),
    )

    id: Mapped[uuid.UUID] = mapped_column(Uuid, primary_key=True)  # lo genera el motor
    user_id: Mapped[uuid.UUID] = mapped_column(ForeignKey("users.id"), nullable=False, index=True)
    market_id: Mapped[str] = mapped_column(String(64), nullable=False, index=True)  # sin FK: vive en MongoDB
    option: Mapped[str] = mapped_column(String(100), nullable=False)
    amount: Mapped[int] = mapped_column(BigInteger, nullable=False)
    status: Mapped[str] = mapped_column(String(20), nullable=False, default=BET_PENDING, server_default=BET_PENDING)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False, server_default=func.now())


class LedgerEntry(Base):
    __tablename__ = "ledger_entries"
    __table_args__ = (
        CheckConstraint(
            f"type IN ('{LEDGER_PURCHASE}', '{LEDGER_BET_DEBIT}', '{LEDGER_BET_SETTLEMENT}')",
            name="ck_ledger_entries_type",
        ),
    )

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    user_id: Mapped[uuid.UUID] = mapped_column(ForeignKey("users.id"), nullable=False, index=True)
    bet_id: Mapped[uuid.UUID | None] = mapped_column(ForeignKey("bets.id"), index=True)
    type: Mapped[str] = mapped_column(String(20), nullable=False)
    tokens_delta: Mapped[int] = mapped_column(BigInteger, nullable=False, default=0, server_default="0")
    points_delta: Mapped[Decimal] = mapped_column(Numeric(12, 1), nullable=False, default=Decimal("0"), server_default="0")
    status: Mapped[str] = mapped_column(String(20), nullable=False, default=LEDGER_COMPLETED, server_default=LEDGER_COMPLETED)
    idempotency_key: Mapped[str] = mapped_column(String(255), unique=True, nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False, server_default=func.now())
