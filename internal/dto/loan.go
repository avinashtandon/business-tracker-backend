package dto

import (
	"fmt"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/google/uuid"
)

type CreateLoanRequest struct {
	PersonName      string  `json:"person_name" validate:"required"`
	Purpose         string  `json:"purpose" validate:"required"`
	PrincipalAmount float64 `json:"principal_amount" validate:"required,gt=0"`
	InterestAmount  float64 `json:"interest_amount" validate:"gte=0"`
	Duration        string  `json:"duration" validate:"required"`
	DueDate         string  `json:"due_date" validate:"required"`
	PaymentMode     string  `json:"payment_mode"`
}

type LoanResponse struct {
	ID              string                `json:"id"`
	PersonName      string                `json:"person_name"`
	Purpose         string                `json:"purpose"`
	PrincipalAmount float64               `json:"principal_amount"`
	InterestAmount  float64               `json:"interest_amount"`
	Duration        string                `json:"duration"`
	DueDate         string                `json:"due_date"`
	PaymentMode     string                `json:"payment_mode"`
	Status          string                `json:"status"`
	CreatedAt       string                `json:"created_at"`
	Transactions    []TransactionResponse `json:"transactions,omitempty"`
}

type CreateTransactionRequest struct {
	Date   string  `json:"date" validate:"required"`
	Amount float64 `json:"amount" validate:"required,gt=0"`
	Mode   string  `json:"mode" validate:"required"`
	Note   string  `json:"note"`
}

type TransactionResponse struct {
	ID     string  `json:"id"`
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Mode   string  `json:"mode"`
	Note   string  `json:"note"`
}

func ToLoanDomain(input CreateLoanRequest, userID uuid.UUID) (models.Loan, error) {
	dueDate, err := time.Parse("2006-01-02", input.DueDate)
	if err != nil {
		return models.Loan{}, fmt.Errorf("invalid due_date format, expected YYYY-MM-DD: %w", err)
	}

	return models.Loan{
		ID:              uuid.New(),
		UserID:          userID,
		PersonName:      input.PersonName,
		Purpose:         input.Purpose,
		PrincipalAmount: input.PrincipalAmount,
		InterestAmount:  input.InterestAmount,
		Duration:        input.Duration,
		DueDate:         dueDate,
		PaymentMode:     input.PaymentMode,
		Status:          "pending",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Transactions:    []models.Transaction{},
	}, nil
}

func ToTransactionDomain(input CreateTransactionRequest, loanID uuid.UUID) (models.Transaction, error) {
	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		return models.Transaction{}, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	return models.Transaction{
		ID:        uuid.New(),
		LoanID:    loanID,
		Date:      date,
		Amount:    input.Amount,
		Mode:      input.Mode,
		Note:      input.Note,
		CreatedAt: time.Now(),
	}, nil
}

func ToLoanResponse(l *models.Loan) LoanResponse {
	var txs []TransactionResponse
	for _, tx := range l.Transactions {
		txs = append(txs, ToTransactionResponse(tx))
	}

	return LoanResponse{
		ID:              l.ID.String(),
		PersonName:      l.PersonName,
		Purpose:         l.Purpose,
		PrincipalAmount: l.PrincipalAmount,
		InterestAmount:  l.InterestAmount,
		Duration:        l.Duration,
		DueDate:         l.DueDate.Format("2006-01-02"),
		PaymentMode:     l.PaymentMode,
		Status:          l.Status,
		CreatedAt:       l.CreatedAt.Format(time.RFC3339Nano),
		Transactions:    txs,
	}
}

func ToLoanResponseList(loans []*models.Loan) []LoanResponse {
	list := make([]LoanResponse, len(loans))
	for i, l := range loans {
		list[i] = ToLoanResponse(l)
	}
	return list
}

func ToTransactionResponse(t models.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:     t.ID.String(),
		Date:   t.Date.Format("2006-01-02"),
		Amount: t.Amount,
		Mode:   t.Mode,
		Note:   t.Note,
	}
}
