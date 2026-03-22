package dto

import (
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
)

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Status    string    `json:"status"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToUserResponse(u *models.User) UserResponse {
	roles := u.Roles
	if roles == nil {
		roles = []string{}
	}
	return UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Status:    string(u.Status),
		Roles:     roles,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func ToUserResponseList(users []*models.User) []UserResponse {
	list := make([]UserResponse, len(users))
	for i, u := range users {
		list[i] = ToUserResponse(u)
	}
	return list
}
