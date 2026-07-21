package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/timofeevav/gophkeeper/internal/server/model"
	"github.com/timofeevav/gophkeeper/internal/server/service"
)

func TestSyncService_Sync_NoChanges(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSyncService(repo)

	userID := uuid.New()
	lastSync := time.Now().Add(-time.Hour)

	repo.On("GetVersions", mock.Anything, userID, mock.AnythingOfType("time.Time")).
		Return([]*model.Secret{}, nil)

	result, err := svc.Sync(context.Background(), &service.SyncInput{
		UserID:         userID,
		LastSyncAt:     lastSync,
		ClientVersions: nil,
	})

	require.NoError(t, err)
	assert.Empty(t, result.Updated)
	assert.Empty(t, result.Conflicts)
	assert.False(t, result.SyncAt.IsZero())
}

func TestSyncService_Sync_NewSecrets(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSyncService(repo)

	userID := uuid.New()
	lastSync := time.Now().Add(-time.Hour)
	now := time.Now()

	serverSecrets := []*model.Secret{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Type:      model.SecretTypeLoginPassword,
			Name:      "GitHub",
			Data:      []byte("encrypted"),
			Version:   1,
			UpdatedAt: now,
		},
	}

	repo.On("GetVersions", mock.Anything, userID, mock.AnythingOfType("time.Time")).
		Return(serverSecrets, nil)

	result, err := svc.Sync(context.Background(), &service.SyncInput{
		UserID:         userID,
		LastSyncAt:     lastSync,
		ClientVersions: []model.ClientVersion{}, // клиент не знает о секрете
	})

	require.NoError(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Empty(t, result.Conflicts)
	assert.Equal(t, "GitHub", result.Updated[0].Name)
}

func TestSyncService_Sync_ServerAhead(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSyncService(repo)

	userID := uuid.New()
	secretID := uuid.New()
	lastSync := time.Now().Add(-time.Hour)
	now := time.Now()

	serverSecrets := []*model.Secret{
		{
			ID:        secretID,
			UserID:    userID,
			Type:      model.SecretTypeText,
			Name:      "Note",
			Data:      []byte("server-data"),
			Version:   5,
			UpdatedAt: now,
		},
	}

	repo.On("GetVersions", mock.Anything, userID, mock.AnythingOfType("time.Time")).
		Return(serverSecrets, nil)

	// server version (5) > client version (3) → normal update, no conflict
	result, err := svc.Sync(context.Background(), &service.SyncInput{
		UserID:     userID,
		LastSyncAt: lastSync,
		ClientVersions: []model.ClientVersion{
			{ID: secretID, Version: 3},
		},
	})

	require.NoError(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Empty(t, result.Conflicts)
	assert.Equal(t, secretID, result.Updated[0].ID)
}

func TestSyncService_Sync_VersionConflict(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSyncService(repo)

	userID := uuid.New()
	secretID := uuid.New()
	lastSync := time.Now().Add(-time.Hour)
	now := time.Now()

	serverSecrets := []*model.Secret{
		{
			ID:        secretID,
			UserID:    userID,
			Type:      model.SecretTypeText,
			Name:      "Note",
			Data:      []byte("server-data"),
			Version:   3,
			UpdatedAt: now,
		},
	}

	repo.On("GetVersions", mock.Anything, userID, mock.AnythingOfType("time.Time")).
		Return(serverSecrets, nil)

	// client version (99) > server version (3) → conflict
	result, err := svc.Sync(context.Background(), &service.SyncInput{
		UserID:     userID,
		LastSyncAt: lastSync,
		ClientVersions: []model.ClientVersion{
			{ID: secretID, Version: 99},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result.Updated)
	assert.Len(t, result.Conflicts, 1)
	assert.Equal(t, secretID, result.Conflicts[0].ID)
	assert.Equal(t, int64(3), result.Conflicts[0].ServerVersion)
	assert.Equal(t, int64(99), result.Conflicts[0].ClientVersion)
}

func TestSyncService_Sync_SameVersion_NoConflict(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSyncService(repo)

	userID := uuid.New()
	secretID := uuid.New()
	lastSync := time.Now().Add(-time.Hour)
	now := time.Now()

	serverSecrets := []*model.Secret{
		{
			ID:        secretID,
			UserID:    userID,
			Name:      "Note",
			Data:      []byte("data"),
			Version:   3,
			UpdatedAt: now,
		},
	}

	repo.On("GetVersions", mock.Anything, userID, mock.AnythingOfType("time.Time")).
		Return(serverSecrets, nil)

	// same version → already in sync, skip
	result, err := svc.Sync(context.Background(), &service.SyncInput{
		UserID:     userID,
		LastSyncAt: lastSync,
		ClientVersions: []model.ClientVersion{
			{ID: secretID, Version: 3},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result.Updated)
	assert.Empty(t, result.Conflicts)
}

func TestSyncService_Sync_DeletedSecret(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSyncService(repo)

	userID := uuid.New()
	lastSync := time.Now().Add(-time.Hour)
	now := time.Now()
	deletedAt := now

	serverSecrets := []*model.Secret{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Deleted",
			Data:      []byte("data"),
			Version:   2,
			UpdatedAt: now,
			DeletedAt: &deletedAt,
		},
	}

	repo.On("GetVersions", mock.Anything, userID, mock.AnythingOfType("time.Time")).
		Return(serverSecrets, nil)

	result, err := svc.Sync(context.Background(), &service.SyncInput{
		UserID:     userID,
		LastSyncAt: lastSync,
	})

	require.NoError(t, err)
	assert.Len(t, result.Updated, 1)
	assert.NotNil(t, result.Updated[0].DeletedAt)
}
