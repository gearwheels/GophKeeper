package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timofeevav/gophkeeper/internal/server/handler"
	"github.com/timofeevav/gophkeeper/internal/server/model"
	"github.com/timofeevav/gophkeeper/internal/server/service"
)

// userRepo — мок репозитория пользователей для тестов обработчиков.
type userRepo struct {
	users map[string]*model.User
}

func newUserRepo() *userRepo {
	return &userRepo{users: make(map[string]*model.User)}
}

func (r *userRepo) Create(_ context.Context, login, hash string) (uuid.UUID, error) {
	id := uuid.New()
	r.users[login] = &model.User{ID: id, Login: login, Password: hash}
	return id, nil
}

func (r *userRepo) GetByLogin(_ context.Context, login string) (*model.User, error) {
	u, ok := r.users[login]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func TestAuthHandler_Register_BadJSON(t *testing.T) {
	h := handler.NewAuthHandler(service.NewAuthService(newUserRepo(), "secret"))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	h.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_EmptyFields(t *testing.T) {
	h := handler.NewAuthHandler(service.NewAuthService(newUserRepo(), "secret"))

	body, _ := json.Marshal(map[string]string{"login": "", "password": ""})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	repo := newUserRepo()
	h := handler.NewAuthHandler(service.NewAuthService(repo, "secret-32-bytes-long-here-ok!!"))

	body, _ := json.Marshal(map[string]string{"login": "alice", "password": "pass123"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["token"])
	assert.NotEmpty(t, resp["user_id"])
}

func TestAuthHandler_Register_Conflict(t *testing.T) {
	repo := newUserRepo()
	h := handler.NewAuthHandler(service.NewAuthService(repo, "secret-32-bytes-long-here-ok!!"))

	body, _ := json.Marshal(map[string]string{"login": "alice", "password": "pass123"})

	// первая регистрация
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Register(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// повторная регистрация
	req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	w2 := httptest.NewRecorder()
	h.Register(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	repo := newUserRepo()
	svc := service.NewAuthService(repo, "secret-32-bytes-long-here-ok!!")
	h := handler.NewAuthHandler(svc)

	// зарегистрировать
	regBody, _ := json.Marshal(map[string]string{"login": "bob", "password": "mypass"})
	regReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(regBody))
	regW := httptest.NewRecorder()
	h.Register(regW, regReq)
	require.Equal(t, http.StatusCreated, regW.Code)

	// войти
	loginBody, _ := json.Marshal(map[string]string{"login": "bob", "password": "mypass"})
	loginReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(loginBody))
	loginW := httptest.NewRecorder()
	h.Login(loginW, loginReq)

	require.Equal(t, http.StatusOK, loginW.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginW.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["token"])
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	repo := newUserRepo()
	svc := service.NewAuthService(repo, "secret-32-bytes-long-here-ok!!")
	h := handler.NewAuthHandler(svc)

	regBody, _ := json.Marshal(map[string]string{"login": "charlie", "password": "correct"})
	regReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(regBody))
	h.Register(httptest.NewRecorder(), regReq)

	loginBody, _ := json.Marshal(map[string]string{"login": "charlie", "password": "wrong"})
	loginReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(loginBody))
	w := httptest.NewRecorder()
	h.Login(w, loginReq)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Login_BadJSON(t *testing.T) {
	h := handler.NewAuthHandler(service.NewAuthService(newUserRepo(), "secret"))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("bad"))
	w := httptest.NewRecorder()
	h.Login(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
