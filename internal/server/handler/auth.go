package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/timofeevav/gophkeeper/internal/server/service"
)

// AuthHandler обрабатывает запросы аутентификации.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler создаёт AuthHandler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Register обрабатывает POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Login == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	result, err := h.auth.Register(r.Context(), req.Login, req.Password)
	if errors.Is(err, service.ErrLoginTaken) {
		writeError(w, http.StatusConflict, "login already taken")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"token":   result.Token,
		"user_id": result.UserID,
	})
}

// Login обрабатывает POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Login == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	result, err := h.auth.Login(r.Context(), req.Login, req.Password)
	if errors.Is(err, service.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":   result.Token,
		"user_id": result.UserID,
	})
}
