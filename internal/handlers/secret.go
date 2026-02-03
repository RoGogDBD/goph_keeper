package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"goph_keeper/internal/models"
	"goph_keeper/internal/repository"
	"goph_keeper/internal/service"
)

// SecretHandler handles secret CRUD endpoints.
type SecretHandler struct {
	service service.SecretService
}

// NewSecretHandler creates a SecretHandler.
func NewSecretHandler(secretSvc service.SecretService) *SecretHandler {
	return &SecretHandler{service: secretSvc}
}

type (
	secretRequest struct {
		Type    string            `json:"type"`
		Payload []byte            `json:"payload"`
		Meta    map[string]string `json:"meta"`
	}

	secretResponse struct {
		ID        string            `json:"id"`
		Type      string            `json:"type"`
		Payload   []byte            `json:"payload"`
		Meta      map[string]string `json:"meta"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
	}
)

func (h *SecretHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req secretRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	secretType := strings.TrimSpace(req.Type)
	if secretType == "" || len(req.Payload) == 0 {
		writeError(w, http.StatusBadRequest, "type and payload are required")
		return
	}

	created, err := h.service.Create(r.Context(), claims.UserID, secretType, req.Payload, req.Meta)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}

	writeJSON(w, http.StatusCreated, toSecretResponse(created))
}

// List returns all secrets for the current user.
func (h *SecretHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	secrets, err := h.service.List(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}

	resp := make([]secretResponse, 0, len(secrets))
	for _, s := range secrets {
		resp = append(resp, toSecretResponse(s))
	}

	writeJSON(w, http.StatusOK, resp)
}

// Get returns a secret by ID.
func (h *SecretHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	secret, err := h.service.Get(r.Context(), claims.UserID, id)
	if err != nil {
		if errors.Is(err, repository.ErrSecretNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "get failed")
		return
	}

	writeJSON(w, http.StatusOK, toSecretResponse(secret))
}

// Update modifies a secret by ID.
func (h *SecretHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req secretRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	secretType := strings.TrimSpace(req.Type)
	if secretType == "" || len(req.Payload) == 0 {
		writeError(w, http.StatusBadRequest, "type and payload are required")
		return
	}

	updated, err := h.service.Update(r.Context(), claims.UserID, id, secretType, req.Payload, req.Meta)
	if err != nil {
		if errors.Is(err, repository.ErrSecretNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}

	writeJSON(w, http.StatusOK, toSecretResponse(updated))
}

// Delete marks a secret as deleted.
func (h *SecretHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.service.Delete(r.Context(), claims.UserID, id); err != nil {
		if errors.Is(err, repository.ErrSecretNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toSecretResponse(s models.Secret) secretResponse {
	return secretResponse{
		ID:        s.ID,
		Type:      s.Type,
		Payload:   s.Payload,
		Meta:      s.Meta,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
