package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/fernandovmedina/netflix-clone/microservices/auth/database"
)

func main() {
	var db *pgx.Conn
	var err error

	if err = godotenv.Load(".env.local"); err != nil {
		log.Println(err.Error())
		return
	}

	if db, err = database.ConnDB(); err != nil {
		log.Println(err.Error())
		return
	}

	var version string
	if err = db.QueryRow(context.Background(), "SELECT version()").Scan(&version); err != nil {
		log.Println(err.Error())
		return
	}

	fmt.Printf("Supabase Version: %s\n", version)
}
