package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/repository"
)

type telegramBody struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
	Mode   string `json:"parse_mode"`
}

// SendTelegramAlert texts the configured phone via the Telegram Bot API.
func SendTelegramAlert(message string) error {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if token == "" || chatID == "" {
		return fmt.Errorf("telegram configuration missing")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	body, err := json.Marshal(telegramBody{
		ChatID: chatID,
		Text:   message,
		Mode:   "Markdown",
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("calling telegram API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}

	return nil
}

// StartOverdueNotifier starts a background worker that checks for overdue loans.
func StartOverdueNotifier(ctx context.Context, logger *slog.Logger, userRepo repository.UserRepository, loanRepo repository.LoanRepository) {
	// For testing, we run it immediately once, then every 24 hours.
	go func() {
		// Run first check after a short delay so server has time to start completely.
		time.Sleep(5 * time.Second)
		checkAndSendOverdueAlerts(ctx, logger, userRepo, loanRepo)

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("stopping telegram notifier background worker")
				return
			case <-ticker.C:
				checkAndSendOverdueAlerts(ctx, logger, userRepo, loanRepo)
			}
		}
	}()
}

func checkAndSendOverdueAlerts(ctx context.Context, logger *slog.Logger, userRepo repository.UserRepository, loanRepo repository.LoanRepository) {
	// 1. Fetch all users
	users, err := userRepo.ListAll(ctx)
	if err != nil {
		logger.Error("failed to list users for notifications", "error", err)
		return
	}

	now := time.Now()
	alertCount := 0

	for _, user := range users {
		// 2. Fetch loans for user
		loans, err := loanRepo.ListLoansByUser(ctx, user.ID)
		if err != nil {
			logger.Error("failed to list loans for user", "userID", user.ID, "error", err)
			continue
		}

		for _, loan := range loans {
			if loan.DueDate.IsZero() || loan.Status == "Received" {
				continue
			}

			// Calculate remaining amount
			var totalPaid float64
			for _, t := range loan.Transactions {
				if t.Mode == "Received" {
					totalPaid += t.Amount
				}
			}
			totalReturn := loan.PrincipalAmount + loan.InterestAmount
			remaining := totalReturn - totalPaid

			if remaining <= 0 {
				continue
			}

			// Check if overdue
			// If due date was before today (midnight)
			due := time.Date(loan.DueDate.Year(), loan.DueDate.Month(), loan.DueDate.Day(), 0, 0, 0, 0, time.Local)
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

			if due.Before(today) {
				daysOverdue := int(today.Sub(due).Hours() / 24)
				msg := fmt.Sprintf("🚨 *Overdue Alert* (Loan #%s)\n\n*%s* owes *₹%.2f*\nDue: %s (%d days ago!)\nPurpose: %s",
					loan.ID.String()[:8], loan.PersonName, remaining, loan.DueDate.Format("02 Jan 2006"), daysOverdue, loan.Purpose)

				err := SendTelegramAlert(msg)
				if err != nil {
					logger.Error("failed to send telegram alert", "error", err)
				} else {
					alertCount++
				}
			}
		}
	}

	if alertCount > 0 {
		logger.Info("sent telegram overdue alerts", "count", alertCount)
	}
}
