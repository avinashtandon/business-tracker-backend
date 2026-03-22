package models

import (
	"time"

	"github.com/google/uuid"
)

// Loan represents a loan given to a person.
type Loan struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	PersonName      string
	Purpose         string
	PrincipalAmount float64
	InterestAmount  float64
	Duration        string
	DueDate         time.Time
	PaymentMode     string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Transactions    []Transaction
}

// Transaction represents a payment or transaction against a loan.
type Transaction struct {
	ID        uuid.UUID
	LoanID    uuid.UUID
	Date      time.Time
	Amount    float64
	Mode      string
	Note      string
	CreatedAt time.Time
}
