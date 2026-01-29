package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"goph_keeper/internal/auth"
	"goph_keeper/internal/storage"
)

type AuthHandler struct {
	auth *auth.Service
}

func NewAuthHandler(authSvc *auth.Service) *AuthHandler {
	return &AuthHandler{auth: authSvc}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	_, err := h.auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			writeError(w, http.StatusConflict, "user already exists")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	const maxBodySize = 1 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	defer func() {
		_ = r.Body.Close()
	}()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("unexpected data after json body")
	}
	return nil
}
