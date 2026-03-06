package models

import (
	"time"

	"github.com/google/uuid"
)

// CryptoHolding represents a user's holding in a particular coin.
type CryptoHolding struct {
	ID          uuid.UUID        `db:"id" json:"id"`
	UserID      uuid.UUID        `db:"user_id" json:"-"`
	Name        string           `db:"name" json:"name"`
	Symbol      string           `db:"symbol" json:"symbol"`
	CoingeckoID string           `db:"coingecko_id" json:"coingecko_id"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
	Purchases   []CryptoPurchase `db:"-" json:"purchases,omitempty"`
}

// CryptoPurchase represents a single buy event in a holding.
type CryptoPurchase struct {
	ID             uuid.UUID `db:"id" json:"id"`
	HoldingID      uuid.UUID `db:"holding_id" json:"-"`
	Quantity       float64   `db:"quantity" json:"quantity"`
	BuyPrice       float64   `db:"buy_price" json:"buy_price"`
	InvestedAmount float64   `db:"invested_amount" json:"invested_amount"`
	Date           time.Time `db:"date" json:"date"`
	Exchange       string    `db:"exchange" json:"exchange"`
	Note           string    `db:"note" json:"note"`
	CreatedAt      time.Time `db:"created_at" json:"-"`
}
