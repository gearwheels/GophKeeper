package api

import (
	"fmt"
	"net/http"
	"time"
)

// ClientVersion содержит ID и версию секрета на клиенте.
type ClientVersion struct {
	ID      string `json:"id"`
	Version int64  `json:"version"`
}

// SyncConflict описывает конфликт версий.
type SyncConflict struct {
	ID            string `json:"id"`
	ServerVersion int64  `json:"server_version"`
	ClientVersion int64  `json:"client_version"`
}

// SyncRequest содержит параметры запроса синхронизации.
type SyncRequest struct {
	LastSyncAt     time.Time       `json:"last_sync_at"`
	ClientVersions []ClientVersion `json:"client_versions"`
}

// SyncResponse содержит результат синхронизации.
type SyncResponse struct {
	SyncAt    time.Time      `json:"sync_at"`
	Updated   []SecretFull   `json:"updated"`
	Conflicts []SyncConflict `json:"conflicts"`
}

// Sync выполняет синхронизацию с сервером.
func (c *Client) Sync(req SyncRequest) (*SyncResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/sync", req)
	if err != nil {
		return nil, fmt.Errorf("sync: %w", err)
	}

	var result SyncResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("sync response: %w", err)
	}
	return &result, nil
}
