package models

import (
	"time"

	"github.com/google/uuid"
)

// Trade represents a user's stock or crypto trade.
type Trade struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	UserID       uuid.UUID  `db:"user_id" json:"-"`
	Name         string     `db:"name" json:"name"`
	Type         string     `db:"type" json:"type"`
	Position     string     `db:"position" json:"position"`
	Quantity     float64    `db:"quantity" json:"quantity"`
	BuyingPrice  float64    `db:"buying_price" json:"buying_price"`
	BuyingDate   string     `db:"buying_date" json:"buying_date"`
	SellingPrice *float64   `db:"selling_price" json:"selling_price"`
	SellingDate  *string    `db:"selling_date" json:"selling_date"`
	Status       string     `db:"status" json:"status"`
	CreatedAt    time.Time  `db:"created_at" json:"-"`
	UpdatedAt    time.Time  `db:"updated_at" json:"-"`
}
