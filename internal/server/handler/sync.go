package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/timofeevav/gophkeeper/internal/server/middleware"
	"github.com/timofeevav/gophkeeper/internal/server/model"
	"github.com/timofeevav/gophkeeper/internal/server/service"
)

// SyncHandler обрабатывает запросы синхронизации.
type SyncHandler struct {
	svc *service.SyncService
}

// NewSyncHandler создаёт SyncHandler.
func NewSyncHandler(svc *service.SyncService) *SyncHandler {
	return &SyncHandler{svc: svc}
}

type syncRequest struct {
	LastSyncAt     time.Time              `json:"last_sync_at"`
	ClientVersions []model.ClientVersion  `json:"client_versions"`
}

// Sync обрабатывает POST /api/v1/sync.
func (h *SyncHandler) Sync(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Sync(r.Context(), &service.SyncInput{
		UserID:         userID,
		LastSyncAt:     req.LastSyncAt,
		ClientVersions: req.ClientVersions,
	})
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
