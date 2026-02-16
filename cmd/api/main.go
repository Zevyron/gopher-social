package main

import (
	"log"

	"github.com/Zevyron/gopher-social/internal/db"
	"github.com/Zevyron/gopher-social/internal/env"
	"github.com/Zevyron/gopher-social/internal/store"
)

func main() {
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		db: dbConfig{
			addr: env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost:5432/social?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime: env.GetString("DB_MAX_IDLE_TIME", "5m"),
		},
	}

	db, err := db.New(cfg.db.addr, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)
	if err != nil {
		log.Panic(err)
	}

	defer db.Close()
	log.Printf("database connection pool established")
	
	appStore := store.NewPostgresStorage(db)

	app := &application{
		config: cfg,
		store: appStore,
	}

	

	mux := app.mount()
	log.Fatal(app.run(mux))
}