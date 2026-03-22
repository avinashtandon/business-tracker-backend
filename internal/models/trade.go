package models

import (
	"time"

	"github.com/google/uuid"
)

// TradeType represents the type of asset traded.
type TradeType string

const (
	Intraday TradeType = "Intraday"
	Swing    TradeType = "Swing"
)

// PositionType represents if the trade is long or short.
type PositionType string

const (
	Long  PositionType = "Long"
	Short PositionType = "Short"
)

// TradeStatus represents if the trade is open or closed.
type TradeStatus string

const (
	Open   TradeStatus = "Open"
	Closed TradeStatus = "Closed"
)

// Trade represents a user's stock or crypto trade in the domain.
type Trade struct {
	ID           uuid.UUID    `json:"id"`
	UserID       uuid.UUID    `json:"-"`
	Name         string       `json:"name"`
	Type         TradeType    `json:"type"`
	Position     PositionType `json:"position"`
	Quantity     float64      `json:"quantity"`
	BuyingPrice  *int64       `json:"buying_price"`
	BuyingDate   *time.Time   `json:"buying_date"`
	SellingPrice *int64       `json:"selling_price"`
	SellingDate  *time.Time   `json:"selling_date"`
	Status       TradeStatus  `json:"status"`
	CreatedAt    time.Time    `json:"-"`
	UpdatedAt    time.Time    `json:"-"`
}

// TradeModel represents the database row for a trade.
type TradeModel struct {
	ID           uuid.UUID  `db:"id"`
	UserID       uuid.UUID  `db:"user_id"`
	Name         string     `db:"name"`
	Type         string     `db:"type"`
	Position     string     `db:"position"`
	Quantity     float64    `db:"quantity"`
	BuyingPrice  *int64     `db:"buying_price"`
	BuyingDate   *time.Time `db:"buying_date"`
	SellingPrice *int64     `db:"selling_price"`
	SellingDate  *time.Time `db:"selling_date"`
	Status       string     `db:"status"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// CalculateProfit calculates the profit for a trade in paise.
func CalculateProfit(t Trade) int64 {
	if t.SellingPrice == nil || t.BuyingPrice == nil {
		return 0
	}

	if t.Position == Long {
		return (*t.SellingPrice - *t.BuyingPrice) * int64(t.Quantity)
	}

	// SHORT
	return (*t.BuyingPrice - *t.SellingPrice) * int64(t.Quantity)
}
