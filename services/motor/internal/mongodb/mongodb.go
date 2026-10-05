// Package mongodb abre y cierra la conexion del motor con su base de datos.
//
// El motor es el unico dueño de MongoDB (eventos de carrera, mercados y
// contexto). Cada modulo recibe *mongo.Database y usa sus colecciones; ojo
// con los validadores de db/mongo/init: monto y puntos_otorgados son int32.
package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Conexion struct {
	Cliente *mongo.Client
	DB      *mongo.Database
}

// Conectar abre la conexion y verifica con un ping que MongoDB responde.
func Conectar(ctx context.Context, uri, nombreDB string) (*Conexion, error) {
	cliente, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar a MongoDB: %w", err)
	}
	if err := cliente.Ping(ctx, readpref.Primary()); err != nil {
		_ = cliente.Disconnect(context.Background())
		return nil, fmt.Errorf("MongoDB no responde al ping: %w", err)
	}
	return &Conexion{Cliente: cliente, DB: cliente.Database(nombreDB)}, nil
}

// Ping se usa en el healthcheck.
func (c *Conexion) Ping(ctx context.Context) error {
	return c.Cliente.Ping(ctx, readpref.Primary())
}

func (c *Conexion) Cerrar(ctx context.Context) error {
	return c.Cliente.Disconnect(ctx)
}
