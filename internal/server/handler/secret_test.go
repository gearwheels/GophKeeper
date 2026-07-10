package handler_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timofeevav/gophkeeper/internal/server/handler"
	"github.com/timofeevav/gophkeeper/internal/server/middleware"
	"github.com/timofeevav/gophkeeper/internal/server/model"
	"github.com/timofeevav/gophkeeper/internal/server/repository/postgres"
	"github.com/timofeevav/gophkeeper/internal/server/service"
)

// secretRepo — in-memory реализация репозитория секретов для тестов.
type secretRepo struct {
	secrets map[uuid.UUID]*model.Secret
}

func newSecretRepo() *secretRepo {
	return &secretRepo{secrets: make(map[uuid.UUID]*model.Secret)}
}

func (r *secretRepo) Create(_ context.Context, s *model.Secret) (uuid.UUID, error) {
	id := uuid.New()
	s.ID = id
	s.Version = 1
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	cp := *s
	r.secrets[id] = &cp
	return id, nil
}

func (r *secretRepo) GetByID(_ context.Context, id, userID uuid.UUID) (*model.Secret, error) {
	s, ok := r.secrets[id]
	if !ok || s.UserID != userID || s.DeletedAt != nil {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (r *secretRepo) List(_ context.Context, userID uuid.UUID, t model.SecretType, _ *time.Time) ([]*model.Secret, error) {
	var result []*model.Secret
	for _, s := range r.secrets {
		if s.UserID != userID || s.DeletedAt != nil {
			continue
		}
		if t != "" && s.Type != t {
			continue
		}
		cp := *s
		result = append(result, &cp)
	}
	return result, nil
}

func (r *secretRepo) Update(_ context.Context, s *model.Secret) error {
	existing, ok := r.secrets[s.ID]
	if !ok || existing.UserID != s.UserID || existing.DeletedAt != nil {
		return postgres.ErrNotFound
	}
	if existing.Version != s.Version {
		return postgres.ErrConflict
	}
	existing.Name = s.Name
	existing.Data = s.Data
	existing.Meta = s.Meta
	existing.Version++
	existing.UpdatedAt = time.Now()
	return nil
}

func (r *secretRepo) Delete(_ context.Context, id, userID uuid.UUID) error {
	s, ok := r.secrets[id]
	if !ok || s.UserID != userID || s.DeletedAt != nil {
		return postgres.ErrNotFound
	}
	now := time.Now()
	s.DeletedAt = &now
	return nil
}

func (r *secretRepo) GetVersions(_ context.Context, userID uuid.UUID, since time.Time) ([]*model.Secret, error) {
	var result []*model.Secret
	for _, s := range r.secrets {
		if s.UserID == userID && s.UpdatedAt.After(since) {
			cp := *s
			result = append(result, &cp)
		}
	}
	return result, nil
}

const handlerTestSecret = "test-secret-32-bytes-long-here!!"

// injectUserID добавляет userID в контекст запроса через реальный Auth middleware.
func injectUserID(req *http.Request, userID uuid.UUID) *http.Request {
	token, _ := generateTestToken(userID, handlerTestSecret)
	req.Header.Set("Authorization", "Bearer "+token)

	var captured *http.Request
	mw := middleware.Auth(handlerTestSecret)
	h := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r
	}))
	h.ServeHTTP(httptest.NewRecorder(), req)
	if captured != nil {
		return captured
	}
	return req
}

func TestSecretHandler_Create_BadJSON(t *testing.T) {
	svc := service.NewSecretService(newSecretRepo())
	h := handler.NewSecretHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("bad"))
	req = injectUserID(req, uuid.New())
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecretHandler_Create_MissingFields(t *testing.T) {
	svc := service.NewSecretService(newSecretRepo())
	h := handler.NewSecretHandler(svc)

	body, _ := json.Marshal(map[string]string{"type": "", "name": ""})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = injectUserID(req, uuid.New())
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecretHandler_Create_Success(t *testing.T) {
	svc := service.NewSecretService(newSecretRepo())
	h := handler.NewSecretHandler(svc)
	userID := uuid.New()

	data := base64.StdEncoding.EncodeToString([]byte("encrypted-payload"))
	body, _ := json.Marshal(map[string]string{
		"type": "login_password",
		"name": "GitHub",
		"data": data,
		"meta": "dev",
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	h.Create(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["id"])
}

func TestSecretHandler_List_Empty(t *testing.T) {
	svc := service.NewSecretService(newSecretRepo())
	h := handler.NewSecretHandler(svc)
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total"])
}

func TestSecretHandler_Get_NotFound(t *testing.T) {
	svc := service.NewSecretService(newSecretRepo())
	h := handler.NewSecretHandler(svc)
	userID := uuid.New()

	r := chi.NewRouter()
	r.Get("/{id}", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSecretHandler_Get_InvalidID(t *testing.T) {
	svc := service.NewSecretService(newSecretRepo())
	h := handler.NewSecretHandler(svc)
	userID := uuid.New()

	r := chi.NewRouter()
	r.Get("/{id}", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid", nil)
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecretHandler_Delete_Success(t *testing.T) {
	repo := newSecretRepo()
	svc := service.NewSecretService(repo)
	h := handler.NewSecretHandler(svc)
	userID := uuid.New()

	secretID, _ := repo.Create(context.Background(), &model.Secret{
		UserID: userID,
		Type:   model.SecretTypeText,
		Name:   "to-delete",
		Data:   []byte("data"),
	})

	r := chi.NewRouter()
	r.Delete("/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/"+secretID.String(), nil)
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSecretHandler_Delete_NotFound(t *testing.T) {
	svc := service.NewSecretService(newSecretRepo())
	h := handler.NewSecretHandler(svc)
	userID := uuid.New()

	r := chi.NewRouter()
	r.Delete("/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/"+uuid.New().String(), nil)
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSecretHandler_Update_Conflict(t *testing.T) {
	repo := newSecretRepo()
	svc := service.NewSecretService(repo)
	h := handler.NewSecretHandler(svc)
	userID := uuid.New()

	secretID, _ := repo.Create(context.Background(), &model.Secret{
		UserID: userID,
		Type:   model.SecretTypeText,
		Name:   "Note",
		Data:   []byte("data"),
	})

	r := chi.NewRouter()
	r.Put("/{id}", h.Update)

	data := base64.StdEncoding.EncodeToString([]byte("new-data"))
	body, _ := json.Marshal(map[string]interface{}{
		"name":    "Updated",
		"data":    data,
		"meta":    "",
		"version": 999,
	})

	req := httptest.NewRequest(http.MethodPut, "/"+secretID.String(), bytes.NewReader(body))
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestSecretHandler_Update_Success(t *testing.T) {
	repo := newSecretRepo()
	svc := service.NewSecretService(repo)
	h := handler.NewSecretHandler(svc)
	userID := uuid.New()

	secretID, _ := repo.Create(context.Background(), &model.Secret{
		UserID: userID,
		Type:   model.SecretTypeText,
		Name:   "Note",
		Data:   []byte("data"),
	})

	r := chi.NewRouter()
	r.Put("/{id}", h.Update)

	data := base64.StdEncoding.EncodeToString([]byte("new-data"))
	body, _ := json.Marshal(map[string]interface{}{
		"name":    "Updated Note",
		"data":    data,
		"meta":    "updated meta",
		"version": 1,
	})

	req := httptest.NewRequest(http.MethodPut, "/"+secretID.String(), bytes.NewReader(body))
	req = injectUserID(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
