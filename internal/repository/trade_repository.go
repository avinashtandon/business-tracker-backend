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

type TradeRepository interface {
	CreateTrade(ctx context.Context, trade *models.Trade) error
	GetTradeByID(ctx context.Context, tradeID uuid.UUID, userID uuid.UUID) (*models.Trade, error)
	ListTradesByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]*models.Trade, bool, error)
	UpdateTrade(ctx context.Context, trade *models.Trade) error
	DeleteTrade(ctx context.Context, tradeID uuid.UUID, userID uuid.UUID) error
}

type tradeRepo struct {
	db *sqlx.DB
}

func NewTradeRepository(db *sqlx.DB) TradeRepository {
	return &tradeRepo{db: db}
}

type tradeRow struct {
	ID           []byte        `db:"id"`
	UserID       []byte        `db:"user_id"`
	Name         string        `db:"name"`
	Type         string        `db:"type"`
	Position     string        `db:"position"`
	Quantity     float64       `db:"quantity"`
	BuyingPrice  *int64        `db:"buying_price"`
	BuyingDate   *time.Time    `db:"buying_date"`
	SellingPrice *int64        `db:"selling_price"`
	SellingDate  *time.Time    `db:"selling_date"`
	Status       string        `db:"status"`
	CreatedAt    time.Time     `db:"created_at"`
	UpdatedAt    time.Time     `db:"updated_at"`
}

func (r tradeRow) toModel() (*models.Trade, error) {
	id, err := uuid.FromBytes(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing trade id: %w", err)
	}
	userID, err := uuid.FromBytes(r.UserID)
	if err != nil {
		return nil, fmt.Errorf("parsing user id: %w", err)
	}

	trade := &models.Trade{
		ID:           id,
		UserID:       userID,
		Name:         r.Name,
		Type:         models.TradeType(r.Type),
		Position:     models.PositionType(r.Position),
		Quantity:     r.Quantity,
		BuyingPrice:  r.BuyingPrice,
		BuyingDate:   r.BuyingDate,
		SellingPrice: r.SellingPrice,
		SellingDate:  r.SellingDate,
		Status:       models.TradeStatus(r.Status),
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}

	return trade, nil
}

func (r *tradeRepo) CreateTrade(ctx context.Context, trade *models.Trade) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO trades (id, user_id, name, type, position, quantity, buying_price, buying_date, selling_price, selling_date, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		trade.ID[:], trade.UserID[:], trade.Name, string(trade.Type), string(trade.Position), trade.Quantity, trade.BuyingPrice, trade.BuyingDate, trade.SellingPrice, trade.SellingDate, string(trade.Status), trade.CreatedAt, trade.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting trade: %w", err)
	}
	return nil
}

func (r *tradeRepo) GetTradeByID(ctx context.Context, tradeID uuid.UUID, userID uuid.UUID) (*models.Trade, error) {
	var row tradeRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM trades WHERE id = ? AND user_id = ?`, tradeID[:], userID[:])
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting trade: %w", err)
	}

	return row.toModel()
}

func (r *tradeRepo) ListTradesByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]*models.Trade, bool, error) {
	var rows []tradeRow
	var err error

	// We fetch one extra row (limit + 1) to determine if there are more records (hasMore).
	fetchLimit := limit + 1

	if cursor == nil {
		err = r.db.SelectContext(ctx, &rows, `SELECT * FROM trades WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`, userID[:], fetchLimit)
	} else {
		err = r.db.SelectContext(ctx, &rows, `SELECT * FROM trades WHERE user_id = ? AND created_at < ? ORDER BY created_at DESC LIMIT ?`, userID[:], *cursor, fetchLimit)
	}

	if err != nil {
		return nil, false, fmt.Errorf("listing trades: %w", err)
	}

	hasMore := false
	if len(rows) > limit {
		hasMore = true
		rows = rows[:limit] // Trim to actual requested limit
	}

	if len(rows) == 0 {
		return []*models.Trade{}, false, nil
	}

	trades := make([]*models.Trade, 0, len(rows))
	for _, row := range rows {
		trade, err := row.toModel()
		if err != nil {
			return nil, false, err
		}
		trades = append(trades, trade)
	}

	return trades, hasMore, nil
}

func (r *tradeRepo) UpdateTrade(ctx context.Context, trade *models.Trade) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE trades SET name = ?, type = ?, position = ?, quantity = ?, buying_price = ?, buying_date = ?, selling_price = ?, selling_date = ?, status = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`,
		trade.Name, string(trade.Type), string(trade.Position), trade.Quantity, trade.BuyingPrice, trade.BuyingDate, trade.SellingPrice, trade.SellingDate, string(trade.Status), trade.UpdatedAt,
		trade.ID[:], trade.UserID[:],
	)
	if err != nil {
		return fmt.Errorf("updating trade: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *tradeRepo) DeleteTrade(ctx context.Context, tradeID uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM trades WHERE id = ? AND user_id = ?`, tradeID[:], userID[:])
	if err != nil {
		return fmt.Errorf("deleting trade: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

