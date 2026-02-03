package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/repository"
	"goph_keeper/internal/service"
)

// NewRouter builds the HTTP router for API endpoints.
func NewRouter(userStore repository.UserStore, secretStore repository.SecretStore, jwtSvc *auth.JWTService) (http.Handler, error) {
	authSvc := auth.NewService(userStore, jwtSvc, 24*time.Hour)
	secretSvc := service.NewSecretService(secretStore)
	syncSvc := service.NewSyncService(secretStore)
	authHandler := NewAuthHandler(authSvc)
	secretHandler := NewSecretHandler(secretSvc)
	syncHandler := NewSyncHandler(syncSvc)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)
	router.Use(middleware.Compress(5))

	router.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	router.Mount("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir("swagger"))))

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
		r.With(AuthMiddleware(jwtSvc)).Route("/sync", func(sr chi.Router) {
			sr.Get("/", syncHandler.Pull)
			sr.Post("/", syncHandler.Push)
		})
	})

	return router, nil
}
