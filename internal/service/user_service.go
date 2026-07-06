package service

import (
	"context"

	"kibou/thisisyouryear/internal/domain"
)

// UserReader is the read-only user contract the UserService needs.
type UserReader interface {
	ByID(ctx context.Context, id string) (*domain.User, error)
}

// UserService serves user profile reads (e.g. the authenticated /me endpoint).
type UserService struct {
	users UserReader
}

// NewUserService wires a UserService.
func NewUserService(users UserReader) *UserService {
	return &UserService{users: users}
}

// GetByID returns the user with the given id, or domain.ErrNotFound.
func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.users.ByID(ctx, id)
}
