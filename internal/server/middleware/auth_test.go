package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/timofeevav/gophkeeper/internal/server/crypto"
	"github.com/timofeevav/gophkeeper/internal/server/middleware"
)

const testSecret = "test-jwt-secret-32-bytes-long!!"

func okHandler(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	w.Header().Set("X-User-ID", userID.String())
	w.WriteHeader(http.StatusOK)
}

func TestAuth_NoHeader(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_WrongSecret(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	token, err := crypto.GenerateToken(uuid.New(), "different-secret-32-bytes-long!!", time.Hour)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_ValidToken(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	userID := uuid.New()
	token, err := crypto.GenerateToken(userID, testSecret, time.Hour)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, userID.String(), w.Header().Get("X-User-ID"))
}

func TestAuth_MissingBearerPrefix(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	token, _ := crypto.GenerateToken(uuid.New(), testSecret, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", token) // без "Bearer "
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogger_PassesThrough(t *testing.T) {
	h := middleware.Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserIDFromCtx_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	id := middleware.UserIDFromCtx(req.Context())
	assert.Equal(t, uuid.Nil, id)
}
