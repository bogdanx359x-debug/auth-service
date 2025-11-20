package user

import (
	"context"
	"errors"
	"time"

	"authservice/internal/util/jwtutil"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo   *Repository
	signer *jwtutil.Signer
}

func NewService(secret string, ttl time.Duration, repo *Repository) *Service {
	return &Service{
		repo:   repo,
		signer: jwtutil.NewSigner(secret, ttl),
	}
}

func (s *Service) Register(ctx context.Context, username, password string) (User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, "", err
	}
	u, err := s.repo.Create(ctx, username, string(hash))
	if err != nil {
		return User{}, "", err
	}
	token, err := s.signer.Generate(u.ID, u.Username)
	if err != nil {
		return User{}, "", err
	}
	return u, token, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (User, string, error) {
	u, storedHash, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return User{}, "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		return User{}, "", errors.New("invalid credentials")
	}
	token, err := s.signer.Generate(u.ID, u.Username)
	if err != nil {
		return User{}, "", err
	}
	return u, token, nil
}

func (s *Service) VerifyToken(token string) (User, error) {
	claims, err := s.signer.Verify(token)
	if err != nil {
		return User{}, err
	}
	return User{
		ID:       claims.UserID,
		Username: claims.Username,
	}, nil
}
