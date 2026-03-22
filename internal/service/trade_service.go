package service

import (
	"context"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/avinashtandon/business-tracker-backend/internal/dto"
	"github.com/google/uuid"
)

type TradeService interface {
	CreateTrade(ctx context.Context, userID uuid.UUID, input dto.CreateTradeRequest) (*dto.TradeResponse, error)
	GetTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID) (*dto.TradeResponse, error)
	ListTrades(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]dto.TradeResponse, bool, error)
	UpdateTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID, input dto.CreateTradeRequest) error
	DeleteTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID) error
}

type tradeService struct {
	repo repository.TradeRepository
}

func NewTradeService(repo repository.TradeRepository) TradeService {
	return &tradeService{repo: repo}
}

func (s *tradeService) CreateTrade(ctx context.Context, userID uuid.UUID, input dto.CreateTradeRequest) (*dto.TradeResponse, error) {
	trade, err := dto.ToDomain(input, userID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateTrade(ctx, &trade); err != nil {
		return nil, err
	}
	resp := dto.ToResponse(&trade)
	return &resp, nil
}

func (s *tradeService) GetTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID) (*dto.TradeResponse, error) {
	trade, err := s.repo.GetTradeByID(ctx, tradeID, userID)
	if err != nil {
		return nil, err
	}
	resp := dto.ToResponse(trade)
	return &resp, nil
}

func (s *tradeService) ListTrades(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]dto.TradeResponse, bool, error) {
	trades, hasMore, err := s.repo.ListTradesByUser(ctx, userID, cursor, limit)
	if err != nil {
		return nil, false, err
	}
	return dto.ToResponseList(trades), hasMore, nil
}

func (s *tradeService) UpdateTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID, input dto.CreateTradeRequest) error {
	trade, err := s.repo.GetTradeByID(ctx, tradeID, userID)
	if err != nil {
		return err
	}

	updatedTrade, err := dto.ToDomain(input, userID)
	if err != nil {
		return err
	}
	
	trade.Name = updatedTrade.Name
	trade.Type = updatedTrade.Type
	trade.Position = updatedTrade.Position
	trade.Quantity = updatedTrade.Quantity
	trade.BuyingPrice = updatedTrade.BuyingPrice
	trade.BuyingDate = updatedTrade.BuyingDate
	trade.SellingPrice = updatedTrade.SellingPrice
	trade.SellingDate = updatedTrade.SellingDate
	trade.Status = updatedTrade.Status

	trade.UpdatedAt = time.Now().UTC()

	return s.repo.UpdateTrade(ctx, trade)
}

func (s *tradeService) DeleteTrade(ctx context.Context, userID uuid.UUID, tradeID uuid.UUID) error {
	return s.repo.DeleteTrade(ctx, tradeID, userID)
}
