package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/Zevyron/gopher-social/internal/mailer"
	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/Zevyron/gopher-social/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type application struct {
	config 	config
	store 	*store.Storage
	validator *validator.Validate
	logger *zap.SugaredLogger
	db	*sql.DB
	mailer mailer.Client
}

type config struct {
	addr 		string
	frontendURL string
	db 			dbConfig
	env 		string
	mail 		mailConfig
}

type mailConfig struct {
	exp time.Duration
}

type dbConfig struct {
	addr 			string
	maxOpenConns 	int
	maxIdleConns 	int
	maxIdleTime 	string
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
    content, err := os.ReadFile("./docs/swagger.json")
    if err != nil {
        http.Error(w, "swagger.json not found", http.StatusInternalServerError)
        return
    }

    htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
        SpecContent: string(content),
        DarkMode:    true,
        CustomOptions: scalar.CustomOptions{
            PageTitle: "GopherSocial API Reference",
        },
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(htmlContent))
})

	
	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", app.healthCheckHandler)

		r.Route("/authentication", func(r chi.Router) {
			r.Post("/user", app.registerUserHandler)
			r.Put("/activate/{token}", app.activateUserHandler)
			r.Post("/token/resend", app.resendActivationTokenHandler)
		})

		r.Route("/posts", func(r chi.Router) {
			r.Post("/", app.createPostHandler)

			r.Route("/{postID}", func(r chi.Router) {
				r.Use(app.postsContextMiddleware)

				r.Get("/", app.getPostHandler)
				r.Delete("/", app.deletePostHandler)
				r.Patch("/", app.updatePostHandler)
			})
		})
		
		r.Route("/users", func(r chi.Router) {
			r.Route("/{userID}", func(r chi.Router) {
				r.Use(app.userContextMiddleware)
				
				r.Get("/", app.getUserHandler)
				r.Put("/follow", app.followUserHandler)
				r.Put("/unfollow", app.unfollowUserHandler)
			})

			r.Group(func(r chi.Router) {
				r.Get("/feed", app.getUserFeedHandler)
			})
		})
	})

	return r
}

func (app *application) run(mux http.Handler) error {
	// Root context that listens for termination signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// Start the Outbox Worker in a background goroutine
	outboxWorker := worker.NewOutboxWorker(app.db, app.mailer, app.logger, app.config.frontendURL)
	wg.Add(1)
	go outboxWorker.Start(ctx, &wg)

	// Configure HTTP Server
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  time.Minute,
	}

	serverErrors := make(chan error, 1)
	go func() {
		app.logger.Infow("server is listening", "addr", srv.Addr, "env", app.config.env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Block until SIGINT/SIGTERM or a server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		app.logger.Info("shutdown signal received; terminating background processes...")
	}

	// Gracefully drain the HTTP server - 10-second timeout
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return fmt.Errorf("failed to shutdown http server cleanly: %w", err)
	}

	// Wait for the Outbox Worker to finish its current batch
	app.logger.Info("waiting for background workers to finish...")
	wg.Wait()

	app.logger.Info("application exited cleanly")
	return nil
}