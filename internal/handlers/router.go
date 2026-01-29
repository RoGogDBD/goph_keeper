package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/storage"
)

func NewRouter(store storage.UserStore, jwtSvc *auth.JWTService) (http.Handler, error) {
	authSvc := auth.NewService(store, jwtSvc, 24*time.Hour)
	authHandler := NewAuthHandler(authSvc)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)

	router.Route("/api", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.With(AuthMiddleware(jwtSvc)).Get("/me", authHandler.Me)
	})

	return router, nil
}
