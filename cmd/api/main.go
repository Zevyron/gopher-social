package main

import (
	"time"

	"github.com/Zevyron/gopher-social/internal/auth"
	"github.com/Zevyron/gopher-social/internal/db"
	"github.com/Zevyron/gopher-social/internal/env"
	"github.com/Zevyron/gopher-social/internal/mailer"
	"github.com/Zevyron/gopher-social/internal/ratelimiter"
	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/Zevyron/gopher-social/internal/store/cache"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
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
		auth: authConfig{
				token: tokenConfig{
						secret: env.GetString("AUTH_TOKEN_SECRET", ""),
						exp:	time.Minute * 15,
						iss:	env.GetString("AUTH_TOKEN_ISS", "gophersocial"),
				},
		},
		redisCfg: redisConfig{
			addr: 		env.GetString("REDIS_ADDR", "localhost:6379"),
			password: 	env.GetString("REDIS_PW", ""),
			db:			env.GetInt("REDIS_DB", 0),
			enabled:	env.GetBool("REDIS_ENABLED", true),
		},
		rateLimiter: ratelimiter.Config{
			RequestPerTimeFrame: 	env.GetInt("RATELIMITER_REQUESTS_COUNT", 20),
			TimeFrame: 				time.Second * 5,
			Enabled:				env.GetBool("RATELIMITER_ENABLED", true),
		},
	}

	// Logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	// Auth secret checks
	if cfg.auth.token.secret == "" {
		if cfg.env == "development" {
			cfg.auth.token.secret = "dev-only-secret-key-do-not-use-in-production"
			logger.Warn("using default insecure JWT secret in development mode")
		} else {
			logger.Fatal("AUTH_TOKEN_SECRET enviroment variable is required in non-development environments")
		}
	}

	// Database Connection Pool
	dbConn, err := db.New(cfg.db.addr, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)
	if err != nil {
		logger.Fatal(err)
	}
	defer dbConn.Close()
	logger.Info("database connection pool established")

	var rdb *redis.Client
	if cfg.redisCfg.enabled {
		rdb, err = cache.NewRedisClient(cfg.redisCfg.addr, cfg.redisCfg.password, cfg.redisCfg.db)
		if err != nil {
			logger.Fatalf("failed to connect to redis %v", err)
		}
		defer rdb.Close()
		logger.Info("redis cache connection established")
	}

	rateLimiter := ratelimiter.NewFixedWindowLimiter(
		cfg.rateLimiter.RequestPerTimeFrame,
		cfg.rateLimiter.TimeFrame,
	)

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
	
	jwtAuthenticator := auth.NewJWTAuthenticator(
		cfg.auth.token.secret,
		cfg.auth.token.iss,
		cfg.auth.token.iss,
	)

	app := &application{
		config:    cfg,
		store:     appStore,
		db:        dbConn, // Needed by OutboxWorker in app.run
		validator: v,
		logger:    logger,
		mailer:    mailClient,
		authenticator: jwtAuthenticator,
		cacheStorage: cache.NewRedisCache(rdb),
		rateLimiter: rateLimiter,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}