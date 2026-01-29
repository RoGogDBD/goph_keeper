package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/storage"
)

func NewRouter(userStore storage.UserStore, secretStore storage.SecretStore, jwtSvc *auth.JWTService) (http.Handler, error) {
	authSvc := auth.NewService(userStore, jwtSvc, 24*time.Hour)
	authHandler := NewAuthHandler(authSvc)
	secretHandler := NewSecretHandler(secretStore)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)
	router.Use(middleware.Compress(5))

	router.Route("/api", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.With(AuthMiddleware(jwtSvc)).Get("/me", authHandler.Me)
		r.With(AuthMiddleware(jwtSvc)).Route("/secrets", func(sr chi.Router) {
			sr.Post("/", secretHandler.Create)
			sr.Get("/", secretHandler.List)
			sr.Get("/{id}", secretHandler.Get)
			sr.Put("/{id}", secretHandler.Update)
			sr.Delete("/{id}", secretHandler.Delete)
		})
	})

	return router, nil
}
