package models

import (
	"time"

	"github.com/google/uuid"
)

// CryptoHolding represents a user's holding in a particular coin.
type CryptoHolding struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Symbol      string
	CoingeckoID string
	CreatedAt   time.Time
	Purchases   []CryptoPurchase
}

// CryptoPurchase represents a single buy event in a holding.
type CryptoPurchase struct {
	ID             uuid.UUID
	HoldingID      uuid.UUID
	Quantity       float64
	BuyPrice       float64
	InvestedAmount float64
	Date           time.Time
	Exchange       string
	Note           string
	CreatedAt      time.Time
}
