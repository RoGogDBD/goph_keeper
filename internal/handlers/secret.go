package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"goph_keeper/internal/models"
	"goph_keeper/internal/repository"
)

type SecretHandler struct {
	store repository.SecretStore
}

func NewSecretHandler(store repository.SecretStore) *SecretHandler {
	return &SecretHandler{store: store}
}

type secretRequest struct {
	Type    string            `json:"type"`
	Payload []byte            `json:"payload"`
	Meta    map[string]string `json:"meta"`
}

type secretResponse struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Payload   []byte            `json:"payload"`
	Meta      map[string]string `json:"meta"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

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

	id, err := newID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate id")
		return
	}

	now := time.Now().UTC()
	secret := models.Secret{
		ID:        id,
		OwnerID:   claims.UserID,
		Type:      secretType,
		Payload:   req.Payload,
		Meta:      req.Meta,
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := h.store.Create(r.Context(), secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}

	writeJSON(w, http.StatusCreated, toSecretResponse(created))
}

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

	secrets, err := h.store.ListByOwner(r.Context(), claims.UserID)
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

	secret, err := h.store.GetByID(r.Context(), claims.UserID, id)
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

	secret := models.Secret{
		ID:        id,
		OwnerID:   claims.UserID,
		Type:      secretType,
		Payload:   req.Payload,
		Meta:      req.Meta,
		UpdatedAt: time.Now().UTC(),
	}

	updated, err := h.store.Update(r.Context(), secret)
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

	if err := h.store.Delete(r.Context(), claims.UserID, id); err != nil {
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

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
