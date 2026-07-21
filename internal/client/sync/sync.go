// Package sync реализует синхронизацию локального хранилища с сервером.
package sync

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/timofeevav/gophkeeper/internal/client/api"
	"github.com/timofeevav/gophkeeper/internal/client/storage"
)

// Syncer выполняет синхронизацию между локальным хранилищем и сервером.
type Syncer struct {
	client  *api.Client
	storage *storage.Storage
}

// New создаёт Syncer.
func New(client *api.Client, store *storage.Storage) *Syncer {
	return &Syncer{client: client, storage: store}
}

// SyncResult содержит результат синхронизации.
type SyncResult struct {
	Updated   int
	Conflicts int
}

// Run выполняет полный цикл синхронизации.
func (s *Syncer) Run() (*SyncResult, error) {
	lastSync, err := s.storage.GetLastSyncAt()
	if err != nil {
		return nil, fmt.Errorf("get last sync: %w", err)
	}

	localSecrets, err := s.storage.ListSecrets()
	if err != nil {
		return nil, fmt.Errorf("list local secrets: %w", err)
	}

	versions := make([]api.ClientVersion, 0, len(localSecrets))
	for _, ls := range localSecrets {
		versions = append(versions, api.ClientVersion{ID: ls.ID, Version: ls.Version})
	}

	resp, err := s.client.Sync(api.SyncRequest{
		LastSyncAt:     lastSync,
		ClientVersions: versions,
	})
	if err != nil {
		return nil, fmt.Errorf("sync request: %w", err)
	}

	for _, updated := range resp.Updated {
		data, err := base64.StdEncoding.DecodeString(updated.Data)
		if err != nil {
			return nil, fmt.Errorf("decode secret data %s: %w", updated.ID, err)
		}

		local := &storage.LocalSecret{
			ID:        updated.ID,
			Type:      updated.Type,
			Name:      updated.Name,
			Data:      data,
			Meta:      updated.Meta,
			Version:   updated.Version,
			UpdatedAt: updated.UpdatedAt,
			DeletedAt: updated.DeletedAt,
		}
		if err := s.storage.UpsertSecret(local); err != nil {
			return nil, fmt.Errorf("upsert secret %s: %w", updated.ID, err)
		}
	}

	if err := s.storage.SetLastSyncAt(time.Now()); err != nil {
		return nil, fmt.Errorf("set last sync: %w", err)
	}

	return &SyncResult{
		Updated:   len(resp.Updated),
		Conflicts: len(resp.Conflicts),
	}, nil
}
