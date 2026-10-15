package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Expense struct {
	ID        uuid.UUID
	UserID    int64
	Amount    decimal.Decimal
	Category  Category
	SpentAt   time.Time // date only
	Comment   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateExpenseInput struct {
	UserID   int64
	Amount   decimal.Decimal
	Category Category
	SpentAt  time.Time
	Comment  *string
}

type UpdateExpenseInput struct {
	Amount   *decimal.Decimal
	Category *Category
	SpentAt  *time.Time
	Comment  *string
	// ClearComment set to true clears comment even if Comment is nil.
	ClearComment bool
}

type ExpenseFilter struct {
	UserID   *int64
	Category *Category
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

type ExpenseSummary struct {
	TotalAmount decimal.Decimal
	Count       int
	ByCategory  map[Category]decimal.Decimal
}

type User struct {
	TelegramID  int64
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
