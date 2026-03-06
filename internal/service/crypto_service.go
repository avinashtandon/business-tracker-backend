package service

import (
	"context"
	"fmt"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/google/uuid"
)

type CreateHoldingInput struct {
	Name        string `json:"name" validate:"required"`
	Symbol      string `json:"symbol"`
	CoingeckoID string `json:"coingecko_id"`
}

type CreatePurchaseInput struct {
	Quantity       float64 `json:"quantity" validate:"required,gt=0"`
	BuyPrice       float64 `json:"buy_price" validate:"required,gt=0"`
	InvestedAmount float64 `json:"invested_amount" validate:"required,gt=0"`
	Date           string  `json:"date" validate:"required"`
	Exchange       string  `json:"exchange"`
	Note           string  `json:"note"`
}

type CryptoService interface {
	CreateHolding(ctx context.Context, userID uuid.UUID, input CreateHoldingInput) (*models.CryptoHolding, error)
	ListHoldings(ctx context.Context, userID uuid.UUID) ([]*models.CryptoHolding, error)
	DeleteHolding(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID) error

	CreatePurchase(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID, input CreatePurchaseInput) (*models.CryptoPurchase, error)
	DeletePurchase(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID, purchaseID uuid.UUID) error
}

type cryptoSvc struct {
	repo repository.CryptoRepository
}

func NewCryptoService(repo repository.CryptoRepository) CryptoService {
	return &cryptoSvc{repo: repo}
}

func (s *cryptoSvc) CreateHolding(ctx context.Context, userID uuid.UUID, input CreateHoldingInput) (*models.CryptoHolding, error) {
	h := &models.CryptoHolding{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        input.Name,
		Symbol:      input.Symbol,
		CoingeckoID: input.CoingeckoID,
		CreatedAt:   time.Now(),
		Purchases:   []models.CryptoPurchase{},
	}

	if err := s.repo.CreateHolding(ctx, h); err != nil {
		return nil, err
	}
	return h, nil
}

func (s *cryptoSvc) ListHoldings(ctx context.Context, userID uuid.UUID) ([]*models.CryptoHolding, error) {
	return s.repo.ListHoldings(ctx, userID)
}

func (s *cryptoSvc) DeleteHolding(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID) error {
	return s.repo.DeleteHolding(ctx, holdingID, userID)
}

func (s *cryptoSvc) CreatePurchase(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID, input CreatePurchaseInput) (*models.CryptoPurchase, error) {
	// Verify the holding belongs to this user (prevents IDOR)
	if _, err := s.repo.GetHoldingByID(ctx, holdingID, userID); err != nil {
		return nil, fmt.Errorf("verifying holding ownership: %w", err)
	}

	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	p := &models.CryptoPurchase{
		ID:             uuid.New(),
		HoldingID:      holdingID,
		Quantity:       input.Quantity,
		BuyPrice:       input.BuyPrice,
		InvestedAmount: input.InvestedAmount,
		Date:           date,
		Exchange:       input.Exchange,
		Note:           input.Note,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.CreatePurchase(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *cryptoSvc) DeletePurchase(ctx context.Context, userID uuid.UUID, holdingID uuid.UUID, purchaseID uuid.UUID) error {
	// Verify ownership
	if _, err := s.repo.GetHoldingByID(ctx, holdingID, userID); err != nil {
		return fmt.Errorf("verifying holding ownership: %w", err)
	}
	return s.repo.DeletePurchase(ctx, purchaseID, holdingID)
}
