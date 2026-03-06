package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type LoanRepository interface {
	CreateLoan(ctx context.Context, loan *models.Loan) error
	GetLoanByID(ctx context.Context, loanID uuid.UUID, userID uuid.UUID) (*models.Loan, error)
	ListLoansByUser(ctx context.Context, userID uuid.UUID) ([]*models.Loan, error)
	UpdateLoan(ctx context.Context, loan *models.Loan) error
	DeleteLoan(ctx context.Context, loanID uuid.UUID, userID uuid.UUID) error

	CreateTransaction(ctx context.Context, tx *models.Transaction) error
	DeleteTransaction(ctx context.Context, txID uuid.UUID, loanID uuid.UUID) error
}

type loanRepo struct {
	db *sqlx.DB
}

func NewLoanRepository(db *sqlx.DB) LoanRepository {
	return &loanRepo{db: db}
}

// Structs for scanning DB rows with BINARY(16) IDs
type loanRow struct {
	ID              []byte    `db:"id"`
	UserID          []byte    `db:"user_id"`
	PersonName      string    `db:"person_name"`
	Purpose         string    `db:"purpose"`
	PrincipalAmount float64   `db:"principal_amount"`
	InterestAmount  float64   `db:"interest_amount"`
	Duration        string    `db:"duration"`
	DueDate         time.Time `db:"due_date"`
	PaymentMode     string    `db:"payment_mode"`
	Status          string    `db:"status"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

func (r loanRow) toModel() (*models.Loan, error) {
	id, err := uuid.FromBytes(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing loan id: %w", err)
	}
	userID, err := uuid.FromBytes(r.UserID)
	if err != nil {
		return nil, fmt.Errorf("parsing user id: %w", err)
	}
	return &models.Loan{
		ID:              id,
		UserID:          userID,
		PersonName:      r.PersonName,
		Purpose:         r.Purpose,
		PrincipalAmount: r.PrincipalAmount,
		InterestAmount:  r.InterestAmount,
		Duration:        r.Duration,
		DueDate:         r.DueDate,
		PaymentMode:     r.PaymentMode,
		Status:          r.Status,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		Transactions:    []models.Transaction{},
	}, nil
}

type txRow struct {
	ID        []byte    `db:"id"`
	LoanID    []byte    `db:"loan_id"`
	Date      time.Time `db:"date"`
	Amount    float64   `db:"amount"`
	Mode      string    `db:"mode"`
	Note      string    `db:"note"`
	CreatedAt time.Time `db:"created_at"`
}

func (r txRow) toModel() (*models.Transaction, error) {
	id, err := uuid.FromBytes(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing tx id: %w", err)
	}
	loanID, err := uuid.FromBytes(r.LoanID)
	if err != nil {
		return nil, fmt.Errorf("parsing loan id: %w", err)
	}
	return &models.Transaction{
		ID:        id,
		LoanID:    loanID,
		Date:      r.Date,
		Amount:    r.Amount,
		Mode:      r.Mode,
		Note:      r.Note,
		CreatedAt: r.CreatedAt,
	}, nil
}

func (r *loanRepo) CreateLoan(ctx context.Context, loan *models.Loan) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO loans (id, user_id, person_name, purpose, principal_amount, interest_amount, duration, due_date, payment_mode, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		loan.ID[:], loan.UserID[:], loan.PersonName, loan.Purpose, loan.PrincipalAmount, loan.InterestAmount, loan.Duration, loan.DueDate, loan.PaymentMode, loan.Status, loan.CreatedAt, loan.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting loan: %w", err)
	}
	return nil
}

func (r *loanRepo) GetLoanByID(ctx context.Context, loanID uuid.UUID, userID uuid.UUID) (*models.Loan, error) {
	var row loanRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM loans WHERE id = ? AND user_id = ?`, loanID[:], userID[:])
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting loan: %w", err)
	}

	loan, err := row.toModel()
	if err != nil {
		return nil, err
	}

	// Fetch transactions
	var txRows []txRow
	err = r.db.SelectContext(ctx, &txRows, `SELECT * FROM transactions WHERE loan_id = ? ORDER BY date DESC, created_at DESC`, loanID[:])
	if err != nil {
		return nil, fmt.Errorf("getting transactions: %w", err)
	}

	for _, tr := range txRows {
		tx, err := tr.toModel()
		if err != nil {
			return nil, err
		}
		loan.Transactions = append(loan.Transactions, *tx)
	}

	return loan, nil
}

func (r *loanRepo) ListLoansByUser(ctx context.Context, userID uuid.UUID) ([]*models.Loan, error) {
	var rows []loanRow
	err := r.db.SelectContext(ctx, &rows, `SELECT * FROM loans WHERE user_id = ? ORDER BY created_at DESC`, userID[:])
	if err != nil {
		return nil, fmt.Errorf("listing loans: %w", err)
	}

	if len(rows) == 0 {
		return []*models.Loan{}, nil
	}

	loans := make([]*models.Loan, 0, len(rows))
	loanIDs := make([][]byte, 0, len(rows))
	loanMap := make(map[uuid.UUID]*models.Loan)

	for _, row := range rows {
		loan, err := row.toModel()
		if err != nil {
			return nil, err
		}
		loans = append(loans, loan)
		loanIDs = append(loanIDs, loan.ID[:])
		loanMap[loan.ID] = loan
	}

	// Fetch all transactions for these loans
	query, args, err := sqlx.In(`SELECT * FROM transactions WHERE loan_id IN (?) ORDER BY date DESC, created_at DESC`, loanIDs)
	if err != nil {
		return nil, fmt.Errorf("preparing transactions query: %w", err)
	}
	query = r.db.Rebind(query)

	var txRows []txRow
	err = r.db.SelectContext(ctx, &txRows, query, args...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("fetching transactions: %w", err)
	}

	for _, tr := range txRows {
		tx, err := tr.toModel()
		if err != nil {
			continue
		}
		if l, ok := loanMap[tx.LoanID]; ok {
			l.Transactions = append(l.Transactions, *tx)
		}
	}

	return loans, nil
}

func (r *loanRepo) UpdateLoan(ctx context.Context, loan *models.Loan) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE loans SET person_name = ?, purpose = ?, principal_amount = ?, interest_amount = ?, duration = ?, due_date = ?, payment_mode = ?, status = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`,
		loan.PersonName, loan.Purpose, loan.PrincipalAmount, loan.InterestAmount, loan.Duration, loan.DueDate, loan.PaymentMode, loan.Status, loan.UpdatedAt,
		loan.ID[:], loan.UserID[:],
	)
	if err != nil {
		return fmt.Errorf("updating loan: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *loanRepo) DeleteLoan(ctx context.Context, loanID uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM loans WHERE id = ? AND user_id = ?`, loanID[:], userID[:])
	if err != nil {
		return fmt.Errorf("deleting loan: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *loanRepo) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO transactions (id, loan_id, date, amount, mode, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		tx.ID[:], tx.LoanID[:], tx.Date, tx.Amount, tx.Mode, tx.Note, tx.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting transaction: %w", err)
	}
	return nil
}

func (r *loanRepo) DeleteTransaction(ctx context.Context, txID uuid.UUID, loanID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = ? AND loan_id = ?`, txID[:], loanID[:])
	if err != nil {
		return fmt.Errorf("deleting transaction: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
