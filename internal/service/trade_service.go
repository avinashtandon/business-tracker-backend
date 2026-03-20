package service

import (
	"context"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/google/uuid"
)

type CreateTradeInput struct {
	Name         string   `json:"name" validate:"required"`
	Type         string   `json:"type" validate:"required"`
	Position     string   `json:"position" validate:"required"`
	Quantity     float64  `json:"quantity" validate:"required,gt=0"`
	BuyingPrice  float64  `json:"buying_price" validate:"required"`
	BuyingDate   string   `json:"buying_date" validate:"required"`
	SellingPrice *float64 `json:"selling_price"`
	SellingDate  *string  `json:"selling_date"`
	Status       string   `json:"status"`
}

type TradeService interface {
	CreateTrade(ctx context.Context, userID uuid.UUID, input CreateTradeInput) (*models.Trade, error)
	GetTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID) (*models.Trade, error)
	ListTrades(ctx context.Context, userID uuid.UUID) ([]*models.Trade, error)
	UpdateTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID, input CreateTradeInput) error
	DeleteTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID) error
}

type tradeService struct {
	repo repository.TradeRepository
}

func NewTradeService(repo repository.TradeRepository) TradeService {
	return &tradeService{repo: repo}
}

func (s *tradeService) CreateTrade(ctx context.Context, userID uuid.UUID, input CreateTradeInput) (*models.Trade, error) {
	status := "Open"
	if input.Status != "" {
		status = input.Status
	}

	trade := &models.Trade{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         input.Name,
		Type:         input.Type,
		Position:     input.Position,
		Quantity:     input.Quantity,
		BuyingPrice:  input.BuyingPrice,
		BuyingDate:   input.BuyingDate,
		SellingPrice: input.SellingPrice,
		SellingDate:  input.SellingDate,
		Status:       status,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.repo.CreateTrade(ctx, trade); err != nil {
		return nil, err
	}
	return trade, nil
}

func (s *tradeService) GetTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID) (*models.Trade, error) {
	return s.repo.GetTradeByID(ctx, tradeID, userID)
}

func (s *tradeService) ListTrades(ctx context.Context, userID uuid.UUID) ([]*models.Trade, error) {
	return s.repo.ListTradesByUser(ctx, userID)
}

func (s *tradeService) UpdateTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID, input CreateTradeInput) error {
	trade, err := s.repo.GetTradeByID(ctx, tradeID, userID)
	if err != nil {
		return err
	}

	trade.Name = input.Name
	trade.Type = input.Type
	trade.Position = input.Position
	trade.Quantity = input.Quantity
	trade.BuyingPrice = input.BuyingPrice
	trade.BuyingDate = input.BuyingDate
	trade.SellingPrice = input.SellingPrice
	trade.SellingDate = input.SellingDate

	if input.Status != "" {
		trade.Status = input.Status
	}
	trade.UpdatedAt = time.Now().UTC()

	return s.repo.UpdateTrade(ctx, trade)
}

func (s *tradeService) DeleteTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID) error {
	return s.repo.DeleteTrade(ctx, tradeID, userID)
}
