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

type CryptoRepository interface {
	CreateHolding(ctx context.Context, h *models.CryptoHolding) error
	ListHoldings(ctx context.Context, userID uuid.UUID) ([]*models.CryptoHolding, error)
	GetHoldingByID(ctx context.Context, holdingID uuid.UUID, userID uuid.UUID) (*models.CryptoHolding, error)
	DeleteHolding(ctx context.Context, holdingID uuid.UUID, userID uuid.UUID) error

	CreatePurchase(ctx context.Context, p *models.CryptoPurchase) error
	DeletePurchase(ctx context.Context, purchaseID uuid.UUID, holdingID uuid.UUID) error
}

type cryptoRepo struct {
	db *sqlx.DB
}

func NewCryptoRepository(db *sqlx.DB) CryptoRepository {
	return &cryptoRepo{db: db}
}

// --- scan rows ---

type holdingRow struct {
	ID          []byte    `db:"id"`
	UserID      []byte    `db:"user_id"`
	Name        string    `db:"name"`
	Symbol      string    `db:"symbol"`
	CoingeckoID string    `db:"coingecko_id"`
	CreatedAt   time.Time `db:"created_at"`
}

func (r holdingRow) toModel() (*models.CryptoHolding, error) {
	id, err := uuid.FromBytes(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing holding id: %w", err)
	}
	userID, err := uuid.FromBytes(r.UserID)
	if err != nil {
		return nil, fmt.Errorf("parsing user id: %w", err)
	}
	return &models.CryptoHolding{
		ID:          id,
		UserID:      userID,
		Name:        r.Name,
		Symbol:      r.Symbol,
		CoingeckoID: r.CoingeckoID,
		CreatedAt:   r.CreatedAt,
		Purchases:   []models.CryptoPurchase{},
	}, nil
}

type purchaseRow struct {
	ID             []byte    `db:"id"`
	HoldingID      []byte    `db:"holding_id"`
	Quantity       float64   `db:"quantity"`
	BuyPrice       float64   `db:"buy_price"`
	InvestedAmount float64   `db:"invested_amount"`
	Date           time.Time `db:"date"`
	Exchange       string    `db:"exchange"`
	Note           string    `db:"note"`
	CreatedAt      time.Time `db:"created_at"`
}

func (r purchaseRow) toModel() (*models.CryptoPurchase, error) {
	id, err := uuid.FromBytes(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing purchase id: %w", err)
	}
	holdingID, err := uuid.FromBytes(r.HoldingID)
	if err != nil {
		return nil, fmt.Errorf("parsing holding id: %w", err)
	}
	return &models.CryptoPurchase{
		ID:             id,
		HoldingID:      holdingID,
		Quantity:       r.Quantity,
		BuyPrice:       r.BuyPrice,
		InvestedAmount: r.InvestedAmount,
		Date:           r.Date,
		Exchange:       r.Exchange,
		Note:           r.Note,
		CreatedAt:      r.CreatedAt,
	}, nil
}

// --- impl ---

func (r *cryptoRepo) CreateHolding(ctx context.Context, h *models.CryptoHolding) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO crypto_holdings (id, user_id, name, symbol, coingecko_id, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		h.ID[:], h.UserID[:], h.Name, h.Symbol, h.CoingeckoID, h.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting holding: %w", err)
	}
	return nil
}

func (r *cryptoRepo) GetHoldingByID(ctx context.Context, holdingID uuid.UUID, userID uuid.UUID) (*models.CryptoHolding, error) {
	var row holdingRow
	err := r.db.GetContext(ctx, &row,
		`SELECT * FROM crypto_holdings WHERE id = ? AND user_id = ?`, holdingID[:], userID[:])
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting holding: %w", err)
	}

	holding, err := row.toModel()
	if err != nil {
		return nil, err
	}

	var pRows []purchaseRow
	if err := r.db.SelectContext(ctx, &pRows,
		`SELECT * FROM crypto_purchases WHERE holding_id = ? ORDER BY date DESC, created_at DESC`, holdingID[:]); err != nil {
		return nil, fmt.Errorf("getting purchases: %w", err)
	}
	for _, pr := range pRows {
		p, err := pr.toModel()
		if err != nil {
			return nil, err
		}
		holding.Purchases = append(holding.Purchases, *p)
	}

	return holding, nil
}

func (r *cryptoRepo) ListHoldings(ctx context.Context, userID uuid.UUID) ([]*models.CryptoHolding, error) {
	var rows []holdingRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM crypto_holdings WHERE user_id = ? ORDER BY created_at DESC`, userID[:]); err != nil {
		return nil, fmt.Errorf("listing holdings: %w", err)
	}

	if len(rows) == 0 {
		return []*models.CryptoHolding{}, nil
	}

	holdings := make([]*models.CryptoHolding, 0, len(rows))
	holdingIDs := make([][]byte, 0, len(rows))
	holdingMap := make(map[uuid.UUID]*models.CryptoHolding)

	for _, row := range rows {
		h, err := row.toModel()
		if err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
		holdingIDs = append(holdingIDs, h.ID[:])
		holdingMap[h.ID] = h
	}

	query, args, err := sqlx.In(
		`SELECT * FROM crypto_purchases WHERE holding_id IN (?) ORDER BY date DESC, created_at DESC`, holdingIDs)
	if err != nil {
		return nil, fmt.Errorf("preparing purchases query: %w", err)
	}
	query = r.db.Rebind(query)

	var pRows []purchaseRow
	if err := r.db.SelectContext(ctx, &pRows, query, args...); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("fetching purchases: %w", err)
	}

	for _, pr := range pRows {
		p, err := pr.toModel()
		if err != nil {
			continue
		}
		if h, ok := holdingMap[p.HoldingID]; ok {
			h.Purchases = append(h.Purchases, *p)
		}
	}

	return holdings, nil
}

func (r *cryptoRepo) DeleteHolding(ctx context.Context, holdingID uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM crypto_holdings WHERE id = ? AND user_id = ?`, holdingID[:], userID[:])
	if err != nil {
		return fmt.Errorf("deleting holding: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *cryptoRepo) CreatePurchase(ctx context.Context, p *models.CryptoPurchase) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO crypto_purchases (id, holding_id, quantity, buy_price, invested_amount, date, exchange, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID[:], p.HoldingID[:], p.Quantity, p.BuyPrice, p.InvestedAmount, p.Date, p.Exchange, p.Note, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting purchase: %w", err)
	}
	return nil
}

func (r *cryptoRepo) DeletePurchase(ctx context.Context, purchaseID uuid.UUID, holdingID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM crypto_purchases WHERE id = ? AND holding_id = ?`, purchaseID[:], holdingID[:])
	if err != nil {
		return fmt.Errorf("deleting purchase: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
