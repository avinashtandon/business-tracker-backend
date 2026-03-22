package dto

import (
	"time"
)

type RegisterRequest struct {
	Email     string `json:"email"      validate:"required,email,max=255"`
	Username  string `json:"username"   validate:"required,min=3,max=50"`
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name"  validate:"required,max=100"`
	Password  string `json:"password"   validate:"required,min=8,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

type AuthTokensResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int           `json:"expires_in"` // seconds
	ExpiresAt    time.Time     `json:"expires_at"`
	User         *UserResponse `json:"user,omitempty"`
}
