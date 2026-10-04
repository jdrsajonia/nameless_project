# ---- build ----
FROM golang:1.25-alpine AS build

WORKDIR /src

# Resolver dependencias. go.sum se genera aqui si no existe.
COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /motor .

# ---- runtime ----
FROM alpine:3.20

# wget (incluido en busybox de alpine) se usa en el healthcheck del compose.
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /motor /app/motor
USER app

EXPOSE 8080
ENTRYPOINT ["/app/motor"]
