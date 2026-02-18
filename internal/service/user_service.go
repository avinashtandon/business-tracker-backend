package service

import (
	"context"
	"fmt"

	"github.com/linktag/auth-backend/internal/models"
	"github.com/linktag/auth-backend/internal/repository"
)

// UserService defines the business logic for user management (admin operations).
type UserService interface {
	ListUsers(ctx context.Context) ([]*models.PublicUser, error)
}

type userService struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new UserService.
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

// ListUsers returns all users as public profiles (admin only).
func (s *userService) ListUsers(ctx context.Context) ([]*models.PublicUser, error) {
	users, err := s.userRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	result := make([]*models.PublicUser, 0, len(users))
	for _, u := range users {
		roles, err := s.userRepo.GetRoles(ctx, u.ID)
		if err != nil {
			return nil, fmt.Errorf("getting roles for user %s: %w", u.ID, err)
		}
		u.Roles = roles
		result = append(result, u.ToPublic())
	}
	return result, nil
}
