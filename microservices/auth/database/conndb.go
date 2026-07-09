package database

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5"
)

func ConnDB() (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))

	if err != nil {
		return nil, err
	}

	defer conn.Close(context.Background())

	return conn, nil
}
