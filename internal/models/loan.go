package models

import (
	"time"

	"github.com/google/uuid"
)

// Loan represents a loan given to a person.
type Loan struct {
	ID              uuid.UUID     `db:"id" json:"id"`
	UserID          uuid.UUID     `db:"user_id" json:"-"`
	PersonName      string        `db:"person_name" json:"person_name"`
	Purpose         string        `db:"purpose" json:"purpose"`
	PrincipalAmount float64       `db:"principal_amount" json:"principal_amount"`
	InterestAmount  float64       `db:"interest_amount" json:"interest_amount"`
	Duration        string        `db:"duration" json:"duration"`
	DueDate         time.Time     `db:"due_date" json:"due_date"`
	PaymentMode     string        `db:"payment_mode" json:"payment_mode"`
	Status          string        `db:"status" json:"status"`
	CreatedAt       time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time     `db:"updated_at" json:"-"`
	Transactions    []Transaction `db:"-" json:"transactions,omitempty"`
}

// Transaction represents a payment or transaction against a loan.
type Transaction struct {
	ID        uuid.UUID `db:"id" json:"id"`
	LoanID    uuid.UUID `db:"loan_id" json:"-"`
	Date      time.Time `db:"date" json:"date"`
	Amount    float64   `db:"amount" json:"amount"`
	Mode      string    `db:"mode" json:"mode"`
	Note      string    `db:"note" json:"note"`
	CreatedAt time.Time `db:"created_at" json:"-"`
}
