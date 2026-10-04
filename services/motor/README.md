# Motor de apuestas (Go)

Recibe los datos de carrera, maneja los mercados y las apuestas, y empuja todo lo que pasa en vivo al front por WebSocket. Es el único dueño de MongoDB: para mover saldo o puntos le pide a Pagos por REST.

## Estructura

```
main.go              # solo arma las piezas y arranca el servidor
internal/config      # variables de entorno (las obligatorias no tienen valor por defecto)
internal/mongodb     # conexión a MongoDB
internal/httpx       # JSON y CORS compartidos
internal/health      # GET /health (503 si MongoDB no responde)
internal/ws          # hub de WebSocket: GET /ws, Difundir y EnviarA
```

Para agregar una HU, crea un paquete en `internal/` con una función `Register(mux, ...)` y llámala desde `main.go`. Así nadie edita el paquete de otro:

```go
func Register(mux *http.ServeMux, db *mongo.Database, hub *ws.Hub) {
	mux.HandleFunc("POST /mercados", ...)
}
```

## Variables de entorno

| Variable | Obligatoria | Ejemplo |
| --- | --- | --- |
| `MONGO_URI` | sí | `mongodb://arqui:arqui_pass@mongo:27017/apuestas_db?authSource=admin` |
| `MONGO_DB` | sí | `apuestas_db` |
| `JWT_SECRET` | sí | el mismo que usa Pagos |
| `PAGOS_BASE_URL` | sí | `http://pagos:8000` |
| `ALLOWED_ORIGINS` | sí | `http://localhost:5173,http://127.0.0.1:5173` |
| `HTTP_PORT` | no (8080) | `8080` |
| `BETTING_WINDOW_SECONDS` | no (30) | `30` |
| `OBSERVATION_WINDOW_SECONDS` | no (60) | `60` |

Si falta una obligatoria, el motor no arranca y el log dice cuál falta.

## Mensajes del WebSocket

El front se conecta a `ws://localhost:8080/ws`. Solo se aceptan conexiones desde un origen de `ALLOWED_ORIGINS`.

Todo mensaje que envía el motor tiene la misma forma:

```json
{ "type": "mercado_abierto", "payload": { ... } }
```

El front decide qué hacer según `type`. Los tipos están en `internal/ws/mensajes.go`; si agregas uno, ponlo ahí y en esta tabla.

| `type` | A quién | HU | Cuándo |
| --- | --- | --- | --- |
| `posiciones` | todos | HU-05 | cambia la posición de un piloto |
| `mercado_abierto` | todos | HU-11 | el administrador abre un mercado |
| `mercado_cerrado` | todos | HU-11 | termina la ventana de apuestas (empieza la observación) |
| `mercado_liquidado` | todos | HU-07 | el mercado se resolvió |
| `resultado_apuesta` | solo al usuario | HU-08 | se liquidó la apuesta de ese usuario |

Desde Go:

```go
hub.Difundir(ws.Mensaje{Type: ws.TipoMercadoAbierto, Payload: mercado})
hub.EnviarA(userID, ws.Mensaje{Type: ws.TipoResultadoApuesta, Payload: resultado})
```

El `payload` de cada tipo lo define quien implementa la HU; documéntalo aquí cuando quede listo. Propuesta para `resultado_apuesta` (HU-08):

```json
{
  "type": "resultado_apuesta",
  "payload": {
    "mercado_id": "…",
    "apuesta_id": "…",
    "resultado": "ganada",
    "puntos_obtenidos": 60,
    "puntaje_total": 120,
    "saldo_tokens": 400
  }
}
```

Por ahora el front no envía mensajes por el WebSocket: las acciones (apostar, abrir mercado) van por REST. Si la conexión se cae, el front debe reconectarse solo (HU-05).

## Correr y probar

```bash
docker compose up --build motor      # desde la raíz del repo
cd services/motor && gofmt -l . && go vet ./... && go test ./...
```
