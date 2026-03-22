package service

import (
	"context"
	"fmt"

	"github.com/avinashtandon/business-tracker-backend/internal/dto"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/google/uuid"
)

type CryptoService interface {
	CreateHolding(ctx context.Context, userID uuid.UUID, input dto.CreateHoldingRequest) (*dto.CryptoHoldingResponse, error)
	ListHoldings(ctx context.Context, userID uuid.UUID) ([]dto.CryptoHoldingResponse, error)
	DeleteHolding(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID) error

	CreatePurchase(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID, input dto.CreatePurchaseRequest) (*dto.CryptoPurchaseResponse, error)
	DeletePurchase(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID, purchaseID uuid.UUID) error
}

type cryptoSvc struct {
	repo repository.CryptoRepository
}

func NewCryptoService(repo repository.CryptoRepository) CryptoService {
	return &cryptoSvc{repo: repo}
}

func (s *cryptoSvc) CreateHolding(ctx context.Context, userID uuid.UUID, input dto.CreateHoldingRequest) (*dto.CryptoHoldingResponse, error) {
	h := dto.ToHoldingDomain(input, userID)

	if err := s.repo.CreateHolding(ctx, &h); err != nil {
		return nil, err
	}
	resp := dto.ToHoldingResponse(&h)
	return &resp, nil
}

func (s *cryptoSvc) ListHoldings(ctx context.Context, userID uuid.UUID) ([]dto.CryptoHoldingResponse, error) {
	holdings, err := s.repo.ListHoldings(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.ToHoldingResponseList(holdings), nil
}

func (s *cryptoSvc) DeleteHolding(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID) error {
	return s.repo.DeleteHolding(ctx, holdingID, userID)
}

func (s *cryptoSvc) CreatePurchase(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID, input dto.CreatePurchaseRequest) (*dto.CryptoPurchaseResponse, error) {
	// Verify the holding belongs to this user (prevents IDOR)
	if _, err := s.repo.GetHoldingByID(ctx, holdingID, userID); err != nil {
		return nil, fmt.Errorf("verifying holding ownership: %w", err)
	}

	p, err := dto.ToPurchaseDomain(input, holdingID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreatePurchase(ctx, &p); err != nil {
		return nil, err
	}
	resp := dto.ToPurchaseResponse(p)
	return &resp, nil
}

func (s *cryptoSvc) DeletePurchase(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID, purchaseID uuid.UUID) error {
	// Verify ownership
	if _, err := s.repo.GetHoldingByID(ctx, holdingID, userID); err != nil {
		return fmt.Errorf("verifying holding ownership: %w", err)
	}
	return s.repo.DeletePurchase(ctx, purchaseID, holdingID)
}
