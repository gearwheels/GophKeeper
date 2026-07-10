package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/timofeevav/gophkeeper/internal/server/model"
	"github.com/timofeevav/gophkeeper/internal/server/repository/postgres"
	"github.com/timofeevav/gophkeeper/internal/server/service"
)

// mockUserRepo — мок репозитория пользователей.
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) Create(ctx context.Context, login, passwordHash string) (uuid.UUID, error) {
	args := m.Called(ctx, login, passwordHash)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockUserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

const testJWTSecret = "test-secret-32-bytes-long-here!!"

func TestAuthService_Register_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewAuthService(repo, testJWTSecret)

	userID := uuid.New()
	repo.On("GetByLogin", mock.Anything, "newuser").Return(nil, nil)
	repo.On("Create", mock.Anything, "newuser", mock.AnythingOfType("string")).Return(userID, nil)

	result, err := svc.Register(context.Background(), "newuser", "password123")
	require.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
	assert.NotEmpty(t, result.Token)
	repo.AssertExpectations(t)
}

func TestAuthService_Register_LoginTaken(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewAuthService(repo, testJWTSecret)

	existing := &model.User{ID: uuid.New(), Login: "existinguser"}
	repo.On("GetByLogin", mock.Anything, "existinguser").Return(existing, nil)

	_, err := svc.Register(context.Background(), "existinguser", "password")
	assert.ErrorIs(t, err, service.ErrLoginTaken)
	repo.AssertExpectations(t)
}

func TestAuthService_Register_RaceLoginTaken(t *testing.T) {
	// Гонка: GetByLogin прошёл (логин свободен), но параллельная регистрация
	// успела первой — Create возвращает нарушение уникальности.
	repo := new(mockUserRepo)
	svc := service.NewAuthService(repo, testJWTSecret)

	repo.On("GetByLogin", mock.Anything, "raceuser").Return(nil, nil)
	repo.On("Create", mock.Anything, "raceuser", mock.AnythingOfType("string")).
		Return(uuid.Nil, postgres.ErrDuplicate)

	_, err := svc.Register(context.Background(), "raceuser", "password")
	assert.ErrorIs(t, err, service.ErrLoginTaken)
	repo.AssertExpectations(t)
}

func TestAuthService_Register_RepoError(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewAuthService(repo, testJWTSecret)

	repo.On("GetByLogin", mock.Anything, "user").Return(nil, errors.New("db error"))

	_, err := svc.Register(context.Background(), "user", "password")
	assert.Error(t, err)
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewAuthService(repo, testJWTSecret)

	hash := captureHashFromRegister(t, "mypassword")
	user := &model.User{ID: uuid.New(), Login: "alice", Password: hash}
	repo.On("GetByLogin", mock.Anything, "alice").Return(user, nil)

	result, err := svc.Login(context.Background(), "alice", "mypassword")
	require.NoError(t, err)
	assert.Equal(t, user.ID, result.UserID)
	assert.NotEmpty(t, result.Token)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewAuthService(repo, testJWTSecret)

	hash := captureHashFromRegister(t, "correctpassword")
	user := &model.User{ID: uuid.New(), Login: "alice", Password: hash}
	repo.On("GetByLogin", mock.Anything, "alice").Return(user, nil)

	_, err := svc.Login(context.Background(), "alice", "wrongpassword")
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewAuthService(repo, testJWTSecret)

	repo.On("GetByLogin", mock.Anything, "nobody").Return(nil, nil)

	_, err := svc.Login(context.Background(), "nobody", "password")
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

// captureHashFromRegister регистрирует временного пользователя и возвращает bcrypt-хеш пароля.
func captureHashFromRegister(t *testing.T, password string) string {
	t.Helper()
	repo := new(mockUserRepo)
	svc := service.NewAuthService(repo, testJWTSecret)

	var capturedHash string
	repo.On("GetByLogin", mock.Anything, "_hashtest").Return(nil, nil)
	repo.On("Create", mock.Anything, "_hashtest", mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) { capturedHash = args.String(2) }).
		Return(uuid.New(), nil)

	_, err := svc.Register(context.Background(), "_hashtest", password)
	require.NoError(t, err)
	return capturedHash
}
