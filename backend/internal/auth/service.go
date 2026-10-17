package auth

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"bank-app-ioc/backend/repository"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInactive = errors.New("inactive user")
var validUsername = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,99}$`)

type Service struct {
	users             *repository.UserRepository
	sessions          *repository.SessionRepository
	idleTimeout       time.Duration
	maxLifetime       time.Duration
	passwordMinLength int
}

func NewService(users *repository.UserRepository, sessions *repository.SessionRepository, idle, lifetime time.Duration, minPassword int) *Service {
	return &Service{users: users, sessions: sessions, idleTimeout: idle, maxLifetime: lifetime, passwordMinLength: minPassword}
}

func (s *Service) Signup(ctx context.Context, req SignupRequest) (User, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.FullName = strings.TrimSpace(req.FullName)
	if !validUsername.MatchString(req.Username) || req.FullName == "" || len(req.FullName) > 150 || len(req.Password) < s.passwordMinLength {
		return User{}, errors.New("invalid signup request")
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		return User{}, err
	}
	record, err := s.users.CreateCustomer(ctx, req.Username, hash, req.FullName)
	return User{ID: record.ID, Username: record.Username, PasswordHash: record.PasswordHash, Role: record.Role, Status: record.Status}, err
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (User, string, error) {
	record, err := s.users.FindByUsername(ctx, strings.TrimSpace(req.Username))
	if err != nil {
		verifyPassword(req.Password, dummyHash)
		return User{}, "", ErrInvalidCredentials
	}
	if !verifyPassword(req.Password, record.PasswordHash) || record.Status != "ACTIVE" {
		return User{}, "", ErrInvalidCredentials
	}
	now := time.Now().UTC()
	sessionID, err := newSessionID()
	if err != nil {
		return User{}, "", err
	}
	if _, err = s.sessions.Create(ctx, record.ID, sessionHash(sessionID), now, now.Add(s.maxLifetime)); err != nil {
		return User{}, "", err
	}
	if err = s.users.UpdateLastLogin(ctx, record.ID, now); err != nil {
		return User{}, "", err
	}
	return User{ID: record.ID, Username: record.Username, PasswordHash: record.PasswordHash, Role: record.Role, Status: record.Status}, sessionID, nil
}

func (s *Service) Authenticate(ctx context.Context, sessionID string) (Session, error) {
	if sessionID == "" {
		return Session{}, ErrInvalidCredentials
	}
	hash := sessionHash(sessionID)
	record, err := s.sessions.Find(ctx, hash)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	now := time.Now().UTC()
	if now.After(record.ExpiresAt) || now.Sub(record.LastActiveAt) > s.idleTimeout {
		_ = s.sessions.Delete(ctx, hash)
		return Session{}, ErrInvalidCredentials
	}
	if err := s.sessions.Touch(ctx, hash, now); err != nil {
		return Session{}, err
	}
	return Session{ID: record.ID, UserID: record.UserID, Username: record.Username, Role: record.Role, LastActiveAt: record.LastActiveAt, ExpiresAt: record.ExpiresAt}, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.sessions.Delete(ctx, sessionHash(sessionID))
}

func PublicUserFor(u User) PublicUser {
	return PublicUser{ID: u.ID, Username: u.Username, Role: u.Role}
}
func PublicUserForSession(s Session) PublicUser {
	return PublicUser{ID: s.UserID, Username: s.Username, Role: s.Role}
}
func IsUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}
