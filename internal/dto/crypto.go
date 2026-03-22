package dto

import (
	"fmt"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/google/uuid"
)

type CreateHoldingRequest struct {
	Name        string `json:"name" validate:"required"`
	Symbol      string `json:"symbol"`
	CoingeckoID string `json:"coingecko_id"`
}

type CryptoHoldingResponse struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Symbol      string                   `json:"symbol"`
	CoingeckoID string                   `json:"coingecko_id"`
	CreatedAt   string                   `json:"created_at"`
	Purchases   []CryptoPurchaseResponse `json:"purchases,omitempty"`
}

type CreatePurchaseRequest struct {
	Quantity       float64 `json:"quantity" validate:"required,gt=0"`
	BuyPrice       float64 `json:"buy_price" validate:"required,gt=0"`
	InvestedAmount float64 `json:"invested_amount" validate:"required,gt=0"`
	Date           string  `json:"date" validate:"required"`
	Exchange       string  `json:"exchange"`
	Note           string  `json:"note"`
}

type CryptoPurchaseResponse struct {
	ID             string  `json:"id"`
	Quantity       float64 `json:"quantity"`
	BuyPrice       float64 `json:"buy_price"`
	InvestedAmount float64 `json:"invested_amount"`
	Date           string  `json:"date"`
	Exchange       string  `json:"exchange"`
	Note           string  `json:"note"`
}

func ToHoldingDomain(input CreateHoldingRequest, userID uuid.UUID) models.CryptoHolding {
	return models.CryptoHolding{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        input.Name,
		Symbol:      input.Symbol,
		CoingeckoID: input.CoingeckoID,
		CreatedAt:   time.Now(),
		Purchases:   []models.CryptoPurchase{},
	}
}

func ToPurchaseDomain(input CreatePurchaseRequest, holdingID uuid.UUID) (models.CryptoPurchase, error) {
	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		return models.CryptoPurchase{}, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	return models.CryptoPurchase{
		ID:             uuid.New(),
		HoldingID:      holdingID,
		Quantity:       input.Quantity,
		BuyPrice:       input.BuyPrice,
		InvestedAmount: input.InvestedAmount,
		Date:           date,
		Exchange:       input.Exchange,
		Note:           input.Note,
		CreatedAt:      time.Now(),
	}, nil
}

func ToHoldingResponse(h *models.CryptoHolding) CryptoHoldingResponse {
	var purchases []CryptoPurchaseResponse
	for _, p := range h.Purchases {
		purchases = append(purchases, ToPurchaseResponse(p))
	}

	return CryptoHoldingResponse{
		ID:          h.ID.String(),
		Name:        h.Name,
		Symbol:      h.Symbol,
		CoingeckoID: h.CoingeckoID,
		CreatedAt:   h.CreatedAt.Format(time.RFC3339Nano),
		Purchases:   purchases,
	}
}

func ToHoldingResponseList(holdings []*models.CryptoHolding) []CryptoHoldingResponse {
	list := make([]CryptoHoldingResponse, len(holdings))
	for i, h := range holdings {
		list[i] = ToHoldingResponse(h)
	}
	return list
}

func ToPurchaseResponse(p models.CryptoPurchase) CryptoPurchaseResponse {
	return CryptoPurchaseResponse{
		ID:             p.ID.String(),
		Quantity:       p.Quantity,
		BuyPrice:       p.BuyPrice,
		InvestedAmount: p.InvestedAmount,
		Date:           p.Date.Format("2006-01-02"),
		Exchange:       p.Exchange,
		Note:           p.Note,
	}
}
