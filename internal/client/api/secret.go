package api

import (
	"fmt"
	"net/http"
	"time"
)

// SecretMeta содержит метаданные секрета (без поля data).
type SecretMeta struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Meta      string     `json:"meta"`
	Version   int64      `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// SecretFull содержит полные данные секрета включая зашифрованный payload.
type SecretFull struct {
	SecretMeta
	Data string `json:"data"` // base64-encoded encrypted payload
}

// ListSecretsResponse содержит ответ на запрос списка секретов.
type ListSecretsResponse struct {
	Secrets []SecretMeta `json:"secrets"`
	Total   int          `json:"total"`
}

// CreateSecretRequest содержит данные для создания секрета.
type CreateSecretRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"` // base64-encoded encrypted payload
	Meta string `json:"meta"`
}

// CreateSecretResponse содержит ответ на создание секрета.
type CreateSecretResponse struct {
	ID        string    `json:"id"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateSecretRequest содержит данные для обновления секрета.
type UpdateSecretRequest struct {
	Name    string `json:"name"`
	Data    string `json:"data"`
	Meta    string `json:"meta"`
	Version int64  `json:"version"`
}

// ListSecrets возвращает список секретов пользователя.
func (c *Client) ListSecrets(secretType string) (*ListSecretsResponse, error) {
	path := "/api/v1/secrets"
	if secretType != "" {
		path += "?type=" + secretType
	}

	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	var result ListSecretsResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("list secrets response: %w", err)
	}
	return &result, nil
}

// GetSecret возвращает полные данные секрета по ID.
func (c *Client) GetSecret(id string) (*SecretFull, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/secrets/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}

	var result SecretFull
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("get secret response: %w", err)
	}
	return &result, nil
}

// CreateSecret создаёт новый секрет.
func (c *Client) CreateSecret(req CreateSecretRequest) (*CreateSecretResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/secrets", req)
	if err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	var result CreateSecretResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("create secret response: %w", err)
	}
	return &result, nil
}

// UpdateSecret обновляет существующий секрет.
func (c *Client) UpdateSecret(id string, req UpdateSecretRequest) error {
	resp, err := c.do(http.MethodPut, "/api/v1/secrets/"+id, req)
	if err != nil {
		return fmt.Errorf("update secret: %w", err)
	}
	return decodeResponse(resp, nil)
}

// DeleteSecret удаляет секрет.
func (c *Client) DeleteSecret(id string) error {
	resp, err := c.do(http.MethodDelete, "/api/v1/secrets/"+id, nil)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete secret: unexpected status %d", resp.StatusCode)
	}
	return nil
}
