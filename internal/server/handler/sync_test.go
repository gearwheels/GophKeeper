package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timofeevav/gophkeeper/internal/server/handler"
	"github.com/timofeevav/gophkeeper/internal/server/model"
	"github.com/timofeevav/gophkeeper/internal/server/service"
)

func TestSyncHandler_BadJSON(t *testing.T) {
	svc := service.NewSyncService(newSecretRepo())
	h := handler.NewSyncHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("bad"))
	req = injectUserID(req, uuid.New())
	w := httptest.NewRecorder()
	h.Sync(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSyncHandler_Success_NoChanges(t *testing.T) {
	svc := service.NewSyncService(newSecretRepo())
	h := handler.NewSyncHandler(svc)
	userID := uuid.New()

	body, _ := json.Marshal(map[string]interface{}{
		"last_sync_at":    time.Now().Add(-time.Hour),
		"client_versions": []interface{}{},
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	h.Sync(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["sync_at"])
}

func TestSyncHandler_Success_WithUpdates(t *testing.T) {
	repo := newSecretRepo()
	svc := service.NewSyncService(repo)
	h := handler.NewSyncHandler(svc)
	userID := uuid.New()

	// Создадим секрет в репозитории
	repo.Create(context.Background(), &model.Secret{ //nolint:errcheck
		UserID: userID,
		Type:   model.SecretTypeText,
		Name:   "MyNote",
		Data:   []byte("data"),
	})

	body, _ := json.Marshal(map[string]interface{}{
		"last_sync_at":    time.Now().Add(-time.Hour),
		"client_versions": []interface{}{},
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	h.Sync(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	updated := resp["updated"].([]interface{})
	assert.Len(t, updated, 1)
}

func TestSyncHandler_Success_ServerAhead(t *testing.T) {
	repo := newSecretRepo()
	svc := service.NewSyncService(repo)
	h := handler.NewSyncHandler(svc)
	userID := uuid.New()

	secretID, _ := repo.Create(context.Background(), &model.Secret{
		UserID: userID,
		Type:   model.SecretTypeText,
		Name:   "Note",
		Data:   []byte("data"),
	})

	// server version (1) > client version (0) → updated, no conflict
	body, _ := json.Marshal(map[string]interface{}{
		"last_sync_at": time.Now().Add(-time.Hour),
		"client_versions": []interface{}{
			map[string]interface{}{"id": secretID.String(), "version": 0},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	h.Sync(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	updated := resp["updated"].([]interface{})
	assert.Len(t, updated, 1)
	conflicts := resp["conflicts"].([]interface{})
	assert.Empty(t, conflicts)
}
