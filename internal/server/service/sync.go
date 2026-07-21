package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/timofeevav/gophkeeper/internal/server/model"
)

// SyncService реализует синхронизацию данных между клиентами.
type SyncService struct {
	secrets SecretRepository
}

// NewSyncService создаёт SyncService.
func NewSyncService(secrets SecretRepository) *SyncService {
	return &SyncService{secrets: secrets}
}

// SyncInput содержит параметры запроса синхронизации.
type SyncInput struct {
	UserID         uuid.UUID
	LastSyncAt     time.Time
	ClientVersions []model.ClientVersion
}

// SecretDTO содержит данные секрета для передачи клиенту.
type SecretDTO struct {
	ID        uuid.UUID        `json:"id"`
	Type      model.SecretType `json:"type"`
	Name      string           `json:"name"`
	Data      string           `json:"data"`
	Meta      string           `json:"meta"`
	Version   int64            `json:"version"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt *time.Time       `json:"deleted_at"`
}

// SyncResult содержит результат синхронизации.
type SyncResult struct {
	SyncAt    time.Time              `json:"sync_at"`
	Updated   []SecretDTO            `json:"updated"`
	Conflicts []model.SyncConflict   `json:"conflicts"`
}

// Sync выполняет синхронизацию и возвращает изменения с момента last_sync_at.
func (s *SyncService) Sync(ctx context.Context, in *SyncInput) (*SyncResult, error) {
	serverSecrets, err := s.secrets.GetVersions(ctx, in.UserID, in.LastSyncAt)
	if err != nil {
		return nil, fmt.Errorf("get versions: %w", err)
	}

	clientMap := make(map[uuid.UUID]int64, len(in.ClientVersions))
	for _, cv := range in.ClientVersions {
		clientMap[cv.ID] = cv.Version
	}

	result := &SyncResult{SyncAt: time.Now()}

	for _, s := range serverSecrets {
		clientVer, known := clientMap[s.ID]
		if known && clientVer == s.Version {
			continue
		}
		if known && clientVer > s.Version {
			result.Conflicts = append(result.Conflicts, model.SyncConflict{
				ID:            s.ID,
				ServerVersion: s.Version,
				ClientVersion: clientVer,
			})
			continue
		}
		result.Updated = append(result.Updated, SecretDTO{
			ID:        s.ID,
			Type:      s.Type,
			Name:      s.Name,
			Data:      base64.StdEncoding.EncodeToString(s.Data),
			Meta:      s.Meta,
			Version:   s.Version,
			UpdatedAt: s.UpdatedAt,
			DeletedAt: s.DeletedAt,
		})
	}

	if result.Updated == nil {
		result.Updated = []SecretDTO{}
	}
	if result.Conflicts == nil {
		result.Conflicts = []model.SyncConflict{}
	}

	return result, nil
}
