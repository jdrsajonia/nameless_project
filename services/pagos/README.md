# Pagos y usuarios (FastAPI + PostgreSQL)

Servicio dueño de PostgreSQL: usuarios, saldos en tokens, apuestas y ledger.
Modelo de datos: `documentation/modelo-datos-pagos.pdf`.

## Estructura

```
app/
├── main.py          # crea la app: CORS, lifespan (espera BD + crea tablas), routers
├── core/config.py   # variables de entorno (Settings)
├── core/security.py # hash de contraseñas (bcrypt) y JWT de sesión
├── db.py            # engine, SessionLocal, get_db, wait_for_db, init_db
├── deps.py          # DbSession, CurrentUser, AdminUser para los routers
├── models.py        # tablas users, bets, ledger_entries
├── schemas/         # modelos Pydantic de entrada/salida
├── routers/         # endpoints REST (registrar cada router en routers/__init__.py)
└── services/        # lógica de negocio (registro, compra, débito, liquidación)
tests/               # pytest
```

## Cómo agregar una HU

1. Esquemas en `app/schemas/<tema>.py`.
2. Lógica en `app/services/<tema>.py` (recibe la `Session`).
3. Endpoints en `app/routers/<tema>.py` con `router = APIRouter(prefix="/<tema>", tags=["<tema>"])`.
4. Agregar el router a `ROUTERS` en `app/routers/__init__.py`.

Para exigir sesión: `def endpoint(user: CurrentUser, db: DbSession)`.
Solo admin: `def endpoint(admin: AdminUser)`.

## Contrato de sesión con el motor (Go)

Debe coincidir con `services/motor/internal/auth/contrato.go`:

| Qué | Valor |
| --- | --- |
| Algoritmo | HS256 con `JWT_SECRET` (compartido) |
| Cookie | `session`, HttpOnly |
| Claims | `sub` = UUID del usuario, `rol` = `player` \| `admin`, `exp` obligatorio |

Emitir el token: `create_access_token(user.id, user.role)` en `app/core/security.py`.

## Correr

Con todo el sistema: `docker compose up --build` desde la raíz → http://localhost:8000/docs

Pruebas (necesitan Python 3.12+ y el Postgres del compose):

```bash
docker compose up -d postgres   # el puerto 5432 no está publicado por defecto, ver nota
cd services/pagos
pip install -r requirements-dev.txt
pytest
```

Nota: el compose no publica el puerto 5432 de Postgres al host. Sin Postgres
accesible, las pruebas que tocan la BD se saltan y solo corren las de seguridad.
