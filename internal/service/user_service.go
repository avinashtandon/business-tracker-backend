package service

import (
	"context"
	"fmt"

	"github.com/avinashtandon/business-tracker-backend/internal/dto"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
)

type UserService interface {
	ListUsers(ctx context.Context) ([]dto.UserResponse, error)
}

type userService struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new UserService.
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

// ListUsers returns all users as public profiles (admin only).
func (s *userService) ListUsers(ctx context.Context) ([]dto.UserResponse, error) {
	users, err := s.userRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	for _, u := range users {
		roles, err := s.userRepo.GetRoles(ctx, u.ID)
		if err != nil {
			return nil, fmt.Errorf("getting roles for user %s: %w", u.ID, err)
		}
		u.Roles = roles
	}
	
	return dto.ToUserResponseList(users), nil
}
