package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/timofeevav/gophkeeper/internal/server/middleware"
	"github.com/timofeevav/gophkeeper/internal/server/model"
	"github.com/timofeevav/gophkeeper/internal/server/service"
)

// SecretHandler обрабатывает запросы к секретам.
type SecretHandler struct {
	svc *service.SecretService
}

// NewSecretHandler создаёт SecretHandler.
func NewSecretHandler(svc *service.SecretService) *SecretHandler {
	return &SecretHandler{svc: svc}
}

type createSecretRequest struct {
	Type model.SecretType `json:"type"`
	Name string           `json:"name"`
	Data string           `json:"data"` // base64-encoded encrypted payload
	Meta string           `json:"meta"`
}

// Create обрабатывает POST /api/v1/secrets.
func (h *SecretHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	var req createSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Type == "" {
		writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}

	data, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(req.Data), "\n", ""))
	if err != nil {
		writeError(w, http.StatusBadRequest, "data must be base64-encoded")
		return
	}

	id, err := h.svc.CreateSecret(r.Context(), &service.CreateInput{
		UserID: userID,
		Type:   req.Type,
		Name:   req.Name,
		Data:   data,
		Meta:   req.Meta,
	})
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         id,
		"version":    1,
		"created_at": time.Now().UTC(),
	})
}

// List обрабатывает GET /api/v1/secrets.
func (h *SecretHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	secretType := model.SecretType(r.URL.Query().Get("type"))
	var since *time.Time
	if s := r.URL.Query().Get("since"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid since format, use RFC3339")
			return
		}
		since = &t
	}

	secrets, err := h.svc.ListSecrets(r.Context(), userID, secretType, since)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	type secretListItem struct {
		ID        uuid.UUID        `json:"id"`
		Type      model.SecretType `json:"type"`
		Name      string           `json:"name"`
		Meta      string           `json:"meta"`
		Version   int64            `json:"version"`
		CreatedAt time.Time        `json:"created_at"`
		UpdatedAt time.Time        `json:"updated_at"`
		DeletedAt *time.Time       `json:"deleted_at"`
	}

	items := make([]secretListItem, 0, len(secrets))
	for _, s := range secrets {
		items = append(items, secretListItem{
			ID:        s.ID,
			Type:      s.Type,
			Name:      s.Name,
			Meta:      s.Meta,
			Version:   s.Version,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
			DeletedAt: s.DeletedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"secrets": items,
		"total":   len(items),
	})
}

// Get обрабатывает GET /api/v1/secrets/{id}.
func (h *SecretHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid secret id")
		return
	}

	secret, err := h.svc.GetSecret(r.Context(), id, userID)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "secret not found")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         secret.ID,
		"type":       secret.Type,
		"name":       secret.Name,
		"data":       base64.StdEncoding.EncodeToString(secret.Data),
		"meta":       secret.Meta,
		"version":    secret.Version,
		"created_at": secret.CreatedAt,
		"updated_at": secret.UpdatedAt,
	})
}

type updateSecretRequest struct {
	Name    string `json:"name"`
	Data    string `json:"data"`
	Meta    string `json:"meta"`
	Version int64  `json:"version"`
}

// Update обрабатывает PUT /api/v1/secrets/{id}.
func (h *SecretHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid secret id")
		return
	}

	var req updateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	data, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(req.Data), "\n", ""))
	if err != nil {
		writeError(w, http.StatusBadRequest, "data must be base64-encoded")
		return
	}

	err = h.svc.UpdateSecret(r.Context(), &service.UpdateInput{
		ID:      id,
		UserID:  userID,
		Name:    req.Name,
		Data:    data,
		Meta:    req.Meta,
		Version: req.Version,
	})
	if errors.Is(err, service.ErrVersionConflict) {
		writeError(w, http.StatusConflict, "version conflict")
		return
	}
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "secret not found")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         id,
		"version":    req.Version + 1,
		"updated_at": time.Now().UTC(),
	})
}

// Delete обрабатывает DELETE /api/v1/secrets/{id}.
func (h *SecretHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid secret id")
		return
	}

	err = h.svc.DeleteSecret(r.Context(), id, userID)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "secret not found")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
