package main

import (
	"log"

	"github.com/Zevyron/gopher-social/internal/db"
	"github.com/Zevyron/gopher-social/internal/env"
	"github.com/Zevyron/gopher-social/internal/store"
)

func main() {
	addr := env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost:5432/social?sslmode=disable")
	
	conn, err := db.New(addr, 3, 3, "15m")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	storage := store.NewPostgresStorage(conn)
	
	db.Seed(*storage)
}