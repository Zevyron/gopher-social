package main

import (
	"log"

	"github.com/Zevyron/gopher-social/internal/env"
	"github.com/Zevyron/gopher-social/internal/store"
)

func main() {
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
	}

	appStore := store.NewPostgresStorage(nil)

	app := &application{
		config: cfg,
		store: appStore,
	}

	

	mux := app.mount()
	log.Fatal(app.run(mux))
}