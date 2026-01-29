package handlers

import (
	"net/http"
	"time"

	"goph_keeper/internal/models"
	"goph_keeper/internal/repository"
)

type SyncHandler struct {
	store repository.SecretStore
}

func NewSyncHandler(store repository.SecretStore) *SyncHandler {
	return &SyncHandler{store: store}
}

type syncItem struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Payload   []byte            `json:"payload"`
	Meta      map[string]string `json:"meta"`
	Deleted   bool              `json:"deleted"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type syncPullResponse struct {
	Items []syncItem `json:"items"`
}

type syncPushRequest struct {
	Items []syncItem `json:"items"`
}

type syncPushResponse struct {
	Applied int `json:"applied"`
}

func (h *SyncHandler) Pull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	since := time.Time{}
	if v := r.URL.Query().Get("since"); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid since")
			return
		}
		since = parsed
	}

	items, err := h.store.ListUpdatedSince(r.Context(), claims.UserID, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sync pull failed")
		return
	}

	resp := syncPullResponse{Items: make([]syncItem, 0, len(items))}
	for _, s := range items {
		resp.Items = append(resp.Items, toSyncItem(s))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *SyncHandler) Push(w http.ResponseWriter, r *http.Request) {
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

	var req syncPushRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	applied := 0
	for _, item := range req.Items {
		if item.ID == "" {
			writeError(w, http.StatusBadRequest, "id is required")
			return
		}
		if !item.Deleted && item.Type == "" {
			writeError(w, http.StatusBadRequest, "type is required")
			return
		}
		if item.CreatedAt.IsZero() {
			item.CreatedAt = time.Now().UTC()
		}
		if item.UpdatedAt.IsZero() {
			item.UpdatedAt = time.Now().UTC()
		}

		secret := models.Secret{
			ID:        item.ID,
			OwnerID:   claims.UserID,
			Type:      item.Type,
			Payload:   item.Payload,
			Meta:      item.Meta,
			Deleted:   item.Deleted,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}

		if _, err := h.store.Upsert(r.Context(), secret); err != nil {
			writeError(w, http.StatusInternalServerError, "sync push failed")
			return
		}
		applied++
	}

	writeJSON(w, http.StatusOK, syncPushResponse{Applied: applied})
}

func toSyncItem(s models.Secret) syncItem {
	return syncItem{
		ID:        s.ID,
		Type:      s.Type,
		Payload:   s.Payload,
		Meta:      s.Meta,
		Deleted:   s.Deleted,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
