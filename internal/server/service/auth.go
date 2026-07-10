// Package service содержит бизнес-логику сервера.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/timofeevav/gophkeeper/internal/server/crypto"
	"github.com/timofeevav/gophkeeper/internal/server/repository/postgres"
)

// ErrLoginTaken возвращается при попытке зарегистрировать занятый логин.
var ErrLoginTaken = errors.New("login already taken")

// ErrInvalidCredentials возвращается при неверном логине или пароле.
var ErrInvalidCredentials = errors.New("invalid credentials")

// TokenTTL — время жизни JWT-токена.
const TokenTTL = 24 * time.Hour

// AuthService реализует аутентификацию и авторизацию пользователей.
type AuthService struct {
	users     UserRepository
	jwtSecret string
}

// NewAuthService создаёт AuthService.
func NewAuthService(users UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

// RegisterResult содержит результат успешной регистрации.
type RegisterResult struct {
	Token  string
	UserID uuid.UUID
}

// Register регистрирует нового пользователя и возвращает JWT-токен.
func (s *AuthService) Register(ctx context.Context, login, password string) (*RegisterResult, error) {
	existing, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("check login: %w", err)
	}
	if existing != nil {
		return nil, ErrLoginTaken
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	userID, err := s.users.Create(ctx, login, hash)
	if errors.Is(err, postgres.ErrDuplicate) {
		// Параллельная регистрация с тем же логином прошла проверку GetByLogin
		// раньше нас — уникальный индекс в БД разрешает гонку.
		return nil, ErrLoginTaken
	}
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := crypto.GenerateToken(userID, s.jwtSecret, TokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &RegisterResult{Token: token, UserID: userID}, nil
}

// LoginResult содержит результат успешного входа.
type LoginResult struct {
	Token  string
	UserID uuid.UUID
}

// Login аутентифицирует пользователя и возвращает JWT-токен.
func (s *AuthService) Login(ctx context.Context, login, password string) (*LoginResult, error) {
	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil || !crypto.CheckPassword(password, user.Password) {
		return nil, ErrInvalidCredentials
	}

	token, err := crypto.GenerateToken(user.ID, s.jwtSecret, TokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResult{Token: token, UserID: user.ID}, nil
}
