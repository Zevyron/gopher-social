package main

import (
	"time"

	"github.com/Zevyron/gopher-social/internal/db"
	"github.com/Zevyron/gopher-social/internal/env"
	"github.com/Zevyron/gopher-social/internal/mailer"
	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

const version = "0.0.1"

func main() {
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		frontendURL: env.GetString("FRONTEND_URL", "http://localhost:3000"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost:5432/social?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "5m"),
		},
		env: env.GetString("ENV", "development"),
		mail: mailConfig{
			exp: time.Hour * 24 * 3, // 3 days
		},
	}

	// Logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	// Database Connection Pool
	dbConn, err := db.New(cfg.db.addr, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)
	if err != nil {
		logger.Fatal(err)
	}
	defer dbConn.Close()
	logger.Info("database connection pool established")

	// Mailer Client Setup
	mailCfg := mailer.Config{
		Host:     env.GetString("SMTP_HOST", "sandbox.smtp.mailtrap.io"),
		Port:     env.GetInt("SMTP_PORT", 2525),
		Username: env.GetString("SMTP_USER", ""),
		Password: env.GetString("SMTP_PASS", ""),
		Sender:   env.GetString("SMTP_SENDER", "GopherSocial <no-reply@gophersocial.com>"),
	}

	var mailClient mailer.Client
	if cfg.env == "development" && mailCfg.Username == "" {
		logger.Info("using mock mailer")
		mailClient = mailer.NewMockMailer()
	} else {
		logger.Info("using smtp mailer")
		mailClient = mailer.NewSMTPMailer(mailCfg)
	}

	appStore := store.NewPostgresStorage(dbConn)
	v := validator.New(validator.WithRequiredStructEnabled())

	app := &application{
		config:    cfg,
		store:     appStore,
		db:        dbConn, // Needed by OutboxWorker in app.run
		validator: v,
		logger:    logger,
		mailer:    mailClient,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}