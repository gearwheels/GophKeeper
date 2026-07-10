package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/timofeevav/gophkeeper/internal/server/model"
	"github.com/timofeevav/gophkeeper/internal/server/repository/postgres"
)

// ErrNotFound возвращается когда секрет не найден.
var ErrNotFound = errors.New("secret not found")

// ErrForbidden возвращается при попытке доступа к чужому секрету.
var ErrForbidden = errors.New("forbidden")

// ErrVersionConflict возвращается при конфликте версий при обновлении.
var ErrVersionConflict = errors.New("version conflict")

// SecretService реализует бизнес-логику работы с секретами.
type SecretService struct {
	secrets SecretRepository
}

// NewSecretService создаёт SecretService.
func NewSecretService(secrets SecretRepository) *SecretService {
	return &SecretService{secrets: secrets}
}

// CreateInput содержит данные для создания нового секрета.
type CreateInput struct {
	UserID uuid.UUID
	Type   model.SecretType
	Name   string
	Data   []byte
	Meta   string
}

// CreateSecret создаёт новый секрет и возвращает его ID.
func (s *SecretService) CreateSecret(ctx context.Context, in *CreateInput) (uuid.UUID, error) {
	secret := &model.Secret{
		UserID: in.UserID,
		Type:   in.Type,
		Name:   in.Name,
		Data:   in.Data,
		Meta:   in.Meta,
	}
	id, err := s.secrets.Create(ctx, secret)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create secret: %w", err)
	}
	return id, nil
}

// GetSecret возвращает секрет по ID.
func (s *SecretService) GetSecret(ctx context.Context, id, userID uuid.UUID) (*model.Secret, error) {
	secret, err := s.secrets.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}
	if secret == nil {
		return nil, ErrNotFound
	}
	return secret, nil
}

// ListSecrets возвращает список секретов пользователя без поля data.
func (s *SecretService) ListSecrets(ctx context.Context, userID uuid.UUID, secretType model.SecretType, since *time.Time) ([]*model.Secret, error) {
	secrets, err := s.secrets.List(ctx, userID, secretType, since)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	return secrets, nil
}

// UpdateInput содержит данные для обновления секрета.
type UpdateInput struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Name    string
	Data    []byte
	Meta    string
	Version int64
}

// UpdateSecret обновляет секрет с optimistic locking.
func (s *SecretService) UpdateSecret(ctx context.Context, in *UpdateInput) error {
	secret := &model.Secret{
		ID:      in.ID,
		UserID:  in.UserID,
		Name:    in.Name,
		Data:    in.Data,
		Meta:    in.Meta,
		Version: in.Version,
	}
	err := s.secrets.Update(ctx, secret)
	if errors.Is(err, postgres.ErrConflict) {
		return ErrVersionConflict
	}
	if err != nil {
		return fmt.Errorf("update secret: %w", err)
	}
	return nil
}

// DeleteSecret мягко удаляет секрет.
func (s *SecretService) DeleteSecret(ctx context.Context, id, userID uuid.UUID) error {
	err := s.secrets.Delete(ctx, id, userID)
	if errors.Is(err, postgres.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	return nil
}
