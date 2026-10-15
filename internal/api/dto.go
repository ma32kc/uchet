package api

import (
	"time"

	"github.com/k1ra/uchet/internal/domain"
)

type expenseDTO struct {
	ID            string  `json:"id"`
	UserID        int64   `json:"user_id"`
	Amount        string  `json:"amount"`
	Category      string  `json:"category"`
	CategoryLabel string  `json:"category_label"`
	SpentAt       string  `json:"spent_at"`
	Comment       *string `json:"comment"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type expenseListDTO struct {
	Items  []expenseDTO `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type userDTO struct {
	TelegramID  int64  `json:"telegram_id"`
	DisplayName string `json:"display_name"`
}

type createExpenseRequest struct {
	UserID   int64   `json:"user_id"`
	Amount   string  `json:"amount"`
	Category string  `json:"category"`
	SpentAt  string  `json:"spent_at"`
	Comment  *string `json:"comment"`
}

type updateExpenseRequest struct {
	Amount   *string `json:"amount"`
	Category *string `json:"category"`
	SpentAt  *string `json:"spent_at"`
	Comment  *string `json:"comment"`
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func toExpenseDTO(e domain.Expense) expenseDTO {
	return expenseDTO{
		ID:            e.ID.String(),
		UserID:        e.UserID,
		Amount:        e.Amount.StringFixed(2),
		Category:      string(e.Category),
		CategoryLabel: e.Category.Label(),
		SpentAt:       e.SpentAt.Format("2006-01-02"),
		Comment:       e.Comment,
		CreatedAt:     e.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     e.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toUserDTO(u domain.User) userDTO {
	return userDTO{TelegramID: u.TelegramID, DisplayName: u.DisplayName}
}
