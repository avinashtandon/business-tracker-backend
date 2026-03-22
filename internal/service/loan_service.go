package service

import (
	"context"
	"fmt"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/dto"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/google/uuid"
)

type LoanService interface {
	CreateLoan(ctx context.Context, userID uuid.UUID, input dto.CreateLoanRequest) (*dto.LoanResponse, error)
	GetLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID) (*dto.LoanResponse, error)
	ListLoans(ctx context.Context, userID uuid.UUID) ([]dto.LoanResponse, error)
	UpdateLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, input dto.CreateLoanRequest) error
	DeleteLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID) error

	CreateTransaction(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, input dto.CreateTransactionRequest) (*dto.TransactionResponse, error)
	DeleteTransaction(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, txID uuid.UUID) error
}

type loanSvc struct {
	repo repository.LoanRepository
}

func NewLoanService(repo repository.LoanRepository) LoanService {
	return &loanSvc{repo: repo}
}

func (s *loanSvc) CreateLoan(ctx context.Context, userID uuid.UUID, input dto.CreateLoanRequest) (*dto.LoanResponse, error) {
	loan, err := dto.ToLoanDomain(input, userID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateLoan(ctx, &loan); err != nil {
		return nil, err
	}
	resp := dto.ToLoanResponse(&loan)
	return &resp, nil
}

func (s *loanSvc) GetLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID) (*dto.LoanResponse, error) {
	loan, err := s.repo.GetLoanByID(ctx, loanID, userID)
	if err != nil {
		return nil, err
	}
	resp := dto.ToLoanResponse(loan)
	return &resp, nil
}

func (s *loanSvc) ListLoans(ctx context.Context, userID uuid.UUID) ([]dto.LoanResponse, error) {
	loans, err := s.repo.ListLoansByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.ToLoanResponseList(loans), nil
}

func (s *loanSvc) UpdateLoan(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, input dto.CreateLoanRequest) error {
	dueDate, err := time.Parse("2006-01-02", input.DueDate)
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

func (s *loanSvc) CreateTransaction(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, input dto.CreateTransactionRequest) (*dto.TransactionResponse, error) {
	// First, verify the loan belongs to the user
	_, err := s.repo.GetLoanByID(ctx, loanID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying loan ownership: %w", err)
	}

	tx, err := dto.ToTransactionDomain(input, loanID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateTransaction(ctx, &tx); err != nil {
		return nil, err
	}
	resp := dto.ToTransactionResponse(tx)
	return &resp, nil
}

func (s *loanSvc) DeleteTransaction(ctx context.Context, userID uuid.UUID, loanID uuid.UUID, txID uuid.UUID) error {
	_, err := s.repo.GetLoanByID(ctx, loanID, userID)
	if err != nil {
		return fmt.Errorf("verifying loan ownership: %w", err)
	}

	return s.repo.DeleteTransaction(ctx, txID, loanID)
}
