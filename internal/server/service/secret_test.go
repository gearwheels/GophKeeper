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
	"github.com/timofeevav/gophkeeper/internal/server/repository/postgres"
	"github.com/timofeevav/gophkeeper/internal/server/service"
)

// mockSecretRepo — мок репозитория секретов.
type mockSecretRepo struct {
	mock.Mock
}

func (m *mockSecretRepo) Create(ctx context.Context, s *model.Secret) (uuid.UUID, error) {
	args := m.Called(ctx, s)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockSecretRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.Secret, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Secret), args.Error(1)
}

func (m *mockSecretRepo) List(ctx context.Context, userID uuid.UUID, t model.SecretType, since *time.Time) ([]*model.Secret, error) {
	args := m.Called(ctx, userID, t, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Secret), args.Error(1)
}

func (m *mockSecretRepo) Update(ctx context.Context, s *model.Secret) error {
	return m.Called(ctx, s).Error(0)
}

func (m *mockSecretRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return m.Called(ctx, id, userID).Error(0)
}

func (m *mockSecretRepo) GetVersions(ctx context.Context, userID uuid.UUID, since time.Time) ([]*model.Secret, error) {
	args := m.Called(ctx, userID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Secret), args.Error(1)
}

func TestSecretService_CreateSecret(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSecretService(repo)

	userID := uuid.New()
	newID := uuid.New()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Secret")).Return(newID, nil)

	id, err := svc.CreateSecret(context.Background(), &service.CreateInput{
		UserID: userID,
		Type:   model.SecretTypeLoginPassword,
		Name:   "GitHub",
		Data:   []byte("encrypted"),
		Meta:   "dev account",
	})

	require.NoError(t, err)
	assert.Equal(t, newID, id)
}

func TestSecretService_GetSecret_Found(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSecretService(repo)

	userID := uuid.New()
	secretID := uuid.New()
	expected := &model.Secret{ID: secretID, UserID: userID, Name: "Test"}

	repo.On("GetByID", mock.Anything, secretID, userID).Return(expected, nil)

	secret, err := svc.GetSecret(context.Background(), secretID, userID)
	require.NoError(t, err)
	assert.Equal(t, expected, secret)
}

func TestSecretService_GetSecret_NotFound(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSecretService(repo)

	userID := uuid.New()
	secretID := uuid.New()
	repo.On("GetByID", mock.Anything, secretID, userID).Return(nil, nil)

	_, err := svc.GetSecret(context.Background(), secretID, userID)
	assert.ErrorIs(t, err, service.ErrNotFound)
}

func TestSecretService_UpdateSecret_Conflict(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSecretService(repo)

	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Secret")).Return(postgres.ErrConflict)

	err := svc.UpdateSecret(context.Background(), &service.UpdateInput{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Version: 1,
	})
	assert.ErrorIs(t, err, service.ErrVersionConflict)
}

func TestSecretService_UpdateSecret_Success(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSecretService(repo)

	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Secret")).Return(nil)

	err := svc.UpdateSecret(context.Background(), &service.UpdateInput{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Name:    "Updated",
		Version: 2,
	})
	assert.NoError(t, err)
}

func TestSecretService_DeleteSecret_Success(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSecretService(repo)

	id := uuid.New()
	userID := uuid.New()
	repo.On("Delete", mock.Anything, id, userID).Return(nil)

	err := svc.DeleteSecret(context.Background(), id, userID)
	assert.NoError(t, err)
}

func TestSecretService_DeleteSecret_NotFound(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSecretService(repo)

	id := uuid.New()
	userID := uuid.New()
	repo.On("Delete", mock.Anything, id, userID).Return(postgres.ErrNotFound)

	err := svc.DeleteSecret(context.Background(), id, userID)
	assert.ErrorIs(t, err, service.ErrNotFound)
}

func TestSecretService_ListSecrets(t *testing.T) {
	repo := new(mockSecretRepo)
	svc := service.NewSecretService(repo)

	userID := uuid.New()
	secrets := []*model.Secret{
		{ID: uuid.New(), Name: "secret1"},
		{ID: uuid.New(), Name: "secret2"},
	}
	repo.On("List", mock.Anything, userID, model.SecretType(""), (*time.Time)(nil)).Return(secrets, nil)

	result, err := svc.ListSecrets(context.Background(), userID, "", nil)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}
