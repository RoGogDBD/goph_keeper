package handlers

import (
	"errors"
	"net/http"
	"time"

	"goph_keeper/internal/models"
	"goph_keeper/internal/service"
)

// SyncHandler handles sync endpoints.
type SyncHandler struct {
	service service.SyncService
}

// NewSyncHandler creates a SyncHandler.
func NewSyncHandler(syncSvc service.SyncService) *SyncHandler {
	return &SyncHandler{service: syncSvc}
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

type (
	syncPullResponse struct {
		Items []syncItem `json:"items"`
	}

	syncPushRequest struct {
		Items []syncItem `json:"items"`
	}

	syncPushResponse struct {
		Applied int `json:"applied"`
	}
)

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

	items, err := h.service.Pull(r.Context(), claims.UserID, since)
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

// Push applies client changes to the server.
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

	items := make([]models.Secret, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, models.Secret{
			ID:        item.ID,
			Type:      item.Type,
			Payload:   item.Payload,
			Meta:      item.Meta,
			Deleted:   item.Deleted,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}

	applied, err := h.service.Push(r.Context(), claims.UserID, items)
	if err != nil {
		if errors.Is(err, service.ErrSyncInvalidID) || errors.Is(err, service.ErrSyncInvalidType) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "sync push failed")
		return
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
