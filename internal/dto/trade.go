package dto

import (
	"fmt"
	"math"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/google/uuid"
)

type CreateTradeRequest struct {
	Name         string   `json:"name" validate:"required"`
	Type         string   `json:"type" validate:"required,oneof=Intraday Swing"`
	Position     string   `json:"position" validate:"required,oneof=Long Short"`
	Quantity     float64  `json:"quantity" validate:"required,gt=0"`
	BuyingPrice  *float64 `json:"buying_price" validate:"required_if=Position Long"`
	BuyingDate   *string  `json:"buying_date" validate:"required_if=Position Long"`
	SellingPrice *float64 `json:"selling_price" validate:"required_if=Position Short"`
	SellingDate  *string  `json:"selling_date" validate:"required_if=Position Short"`
	Status       string   `json:"status" validate:"omitempty,oneof=Open Closed"`
}

type TradeResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Position     string   `json:"position"`
	Quantity     float64  `json:"quantity"`
	BuyingPrice  *float64 `json:"buying_price,omitempty"`
	BuyingDate   *string  `json:"buying_date,omitempty"`
	SellingPrice *float64 `json:"selling_price,omitempty"`
	SellingDate  *string  `json:"selling_date,omitempty"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
}

// ToDomain maps DTO to domain model.
func ToDomain(req CreateTradeRequest, userID uuid.UUID) (models.Trade, error) {
	trade := models.Trade{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      req.Name,
		Type:      models.TradeType(req.Type),
		Position:  models.PositionType(req.Position),
		Quantity:  req.Quantity,
		Status:    models.TradeStatus(req.Status),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if trade.Status == "" {
		trade.Status = models.Open
	}

	if req.BuyingPrice != nil && req.BuyingDate != nil {
		buyPriceInt := int64(math.Round(*req.BuyingPrice * 100))
		trade.BuyingPrice = &buyPriceInt
		buyingDate, err := time.Parse("2006-01-02", *req.BuyingDate)
		if err == nil {
			trade.BuyingDate = &buyingDate
		} else {
			return models.Trade{}, fmt.Errorf("invalid buying_date format, expected YYYY-MM-DD")
		}
	}

	if req.SellingPrice != nil && req.SellingDate != nil {
		sellPriceInt := int64(math.Round(*req.SellingPrice * 100))
		trade.SellingPrice = &sellPriceInt
		sellingDate, err := time.Parse("2006-01-02", *req.SellingDate)
		if err == nil {
			trade.SellingDate = &sellingDate
		} else {
			return models.Trade{}, fmt.Errorf("invalid selling_date format, expected YYYY-MM-DD")
		}
	}

	return trade, nil
}

// ToResponse maps domain model to response DTO.
func ToResponse(t *models.Trade) TradeResponse {
	var buy *float64
	var buyDate *string
	if t.BuyingPrice != nil {
		v := float64(*t.BuyingPrice) / 100.0
		buy = &v
	}
	if t.BuyingDate != nil {
		d := t.BuyingDate.Format("2006-01-02")
		buyDate = &d
	}

	var sell *float64
	var sellDate *string
	if t.SellingPrice != nil {
		v := float64(*t.SellingPrice) / 100.0
		sell = &v
	}
	if t.SellingDate != nil {
		d := t.SellingDate.Format("2006-01-02")
		sellDate = &d
	}

	return TradeResponse{
		ID:           t.ID.String(),
		Name:         t.Name,
		Type:         string(t.Type),
		Position:     string(t.Position),
		Quantity:     t.Quantity,
		BuyingPrice:  buy,
		BuyingDate:   buyDate,
		SellingPrice: sell,
		SellingDate:  sellDate,
		Status:       string(t.Status),
		CreatedAt:    t.CreatedAt.Format(time.RFC3339Nano),
	}
}

// ToResponseList maps a list of domain models to response DTOs.
func ToResponseList(trades []*models.Trade) []TradeResponse {
	resp := make([]TradeResponse, 0, len(trades))
	for _, t := range trades {
		resp = append(resp, ToResponse(t))
	}
	return resp
}
