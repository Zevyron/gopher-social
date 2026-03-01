package main

import (
	"github.com/Zevyron/gopher-social/internal/db"
	"github.com/Zevyron/gopher-social/internal/env"
	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

const version = "0.0.1"


// @title                       GopherSocial API
// @version                     1.0
// @description                 Production-grade social networking backend built in Go.
// @termsOfService              http://swagger.io/terms/

// @contact.name                API Support
// @contact.email               support@gophersocial.dev

// @license.name                Apache 2.0
// @license.url                 http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath                    /v1

// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        Authorization
// @description                 Include the Bearer prefix: "Bearer <token>"
func main() {
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		db: dbConfig{
			addr: env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost:5432/social?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime: env.GetString("DB_MAX_IDLE_TIME", "5m"),
		},
		env: env.GetString("ENV", "development"),
	}

	// Logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	// Database
	db, err := db.New(cfg.db.addr, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)
	if err != nil {
		logger.Fatal(err)
	}

	defer db.Close()
	logger.Info("database connection pool established")
	
	appStore := store.NewPostgresStorage(db)

	v := validator.New(validator.WithRequiredStructEnabled())

	app := &application{
		config: cfg,
		store: appStore,
		validator: v,
		logger: logger,
	}

	

	mux := app.mount()
	logger.Fatal(app.run(mux))
}