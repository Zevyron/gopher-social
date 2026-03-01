package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/Zevyron/gopher-social/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

type application struct {
	config 	config
	store 	*store.Storage
	validator *validator.Validate
}

type config struct {
	addr 	string
	db 		dbConfig
	env 	string
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

	srv := &http.Server{
		Addr: app.config.addr,
		Handler: mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout: time.Second * 10,
		IdleTimeout: time.Minute,
	}

	log.Printf("Server has started at %s", app.config.addr)

	return srv.ListenAndServe()
}