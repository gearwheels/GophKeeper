package api

import (
	"fmt"
	"net/http"
)

// AuthResponse содержит ответ сервера на запрос аутентификации.
type AuthResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

// Register регистрирует нового пользователя.
func (c *Client) Register(login, password string) (*AuthResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"login":    login,
		"password": password,
	})
	if err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	var result AuthResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("register response: %w", err)
	}
	return &result, nil
}

// Login аутентифицирует пользователя.
func (c *Client) Login(login, password string) (*AuthResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"login":    login,
		"password": password,
	})
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	var result AuthResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("login response: %w", err)
	}
	return &result, nil
}
