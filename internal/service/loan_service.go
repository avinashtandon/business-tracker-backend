package service

import (
	"context"
	"fmt"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/google/uuid"
)

type CreateLoanInput struct {
	PersonName      string  `json:"person_name" validate:"required"`
	Purpose         string  `json:"purpose" validate:"required"`
	PrincipalAmount float64 `json:"principal_amount" validate:"required,gt=0"`
	InterestAmount  float64 `json:"interest_amount" validate:"gte=0"`
	Duration        string  `json:"duration" validate:"required"`
	DueDate         string  `json:"due_date" validate:"required"`
	PaymentMode     string  `json:"payment_mode"`
}

type CreateTransactionInput struct {
	Date   string  `json:"date" validate:"required"`
	Amount float64 `json:"amount" validate:"required,gt=0"`
	Mode   string  `json:"mode" validate:"required"`
	Note   string  `json:"note"`
}

type LoanService interface {
	CreateLoan(ctx context.Context, userID uuid.UUID, input CreateLoanInput) (*models.Loan, error)
	GetLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID) (*models.Loan, error)
	ListLoans(ctx context.Context, userID uuid.UUID) ([]*models.Loan, error)
	UpdateLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, input CreateLoanInput) error
	DeleteLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID) error

	CreateTransaction(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, input CreateTransactionInput) (*models.Transaction, error)
	DeleteTransaction(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, txID uuid.UUID) error
}

type loanSvc struct {
	repo repository.LoanRepository
}

func NewLoanService(repo repository.LoanRepository) LoanService {
	return &loanSvc{repo: repo}
}

func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

func (s *loanSvc) CreateLoan(ctx context.Context, userID uuid.UUID, input CreateLoanInput) (*models.Loan, error) {
	dueDate, err := parseDate(input.DueDate)
	if err != nil {
		return nil, fmt.Errorf("invalid due_date format, expected YYYY-MM-DD: %w", err)
	}

	loan := &models.Loan{
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
	}

	if err := s.repo.CreateLoan(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

func (s *loanSvc) GetLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID) (*models.Loan, error) {
	return s.repo.GetLoanByID(ctx, loanID, userID)
}

func (s *loanSvc) ListLoans(ctx context.Context, userID uuid.UUID) ([]*models.Loan, error) {
	return s.repo.ListLoansByUser(ctx, userID)
}

func (s *loanSvc) UpdateLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, input CreateLoanInput) error {
	dueDate, err := parseDate(input.DueDate)
	if err != nil {
		return fmt.Errorf("invalid due_date format, expected YYYY-MM-DD: %w", err)
	}

	loan, err := s.repo.GetLoanByID(ctx, loanID, userID)
	if err != nil {
		return err
	}

	loan.PersonName = input.PersonName
	loan.Purpose = input.Purpose
	loan.PrincipalAmount = input.PrincipalAmount
	loan.InterestAmount = input.InterestAmount
	loan.Duration = input.Duration
	loan.DueDate = dueDate
	loan.PaymentMode = input.PaymentMode
	loan.UpdatedAt = time.Now()

	return s.repo.UpdateLoan(ctx, loan)
}

func (s *loanSvc) DeleteLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID) error {
	return s.repo.DeleteLoan(ctx, loanID, userID)
}

func (s *loanSvc) CreateTransaction(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, input CreateTransactionInput) (*models.Transaction, error) {
	// First, verify the loan belongs to the user
	_, err := s.repo.GetLoanByID(ctx, loanID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying loan ownership: %w", err)
	}

	txDate, err := parseDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	tx := &models.Transaction{
		ID:        uuid.New(),
		LoanID:    loanID,
		Date:      txDate,
		Amount:    input.Amount,
		Mode:      input.Mode,
		Note:      input.Note,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *loanSvc) DeleteTransaction(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, txID uuid.UUID) error {
	_, err := s.repo.GetLoanByID(ctx, loanID, userID)
	if err != nil {
		return fmt.Errorf("verifying loan ownership: %w", err)
	}

	return s.repo.DeleteTransaction(ctx, txID, loanID)
}
