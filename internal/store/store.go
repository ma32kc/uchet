package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/k1ra/uchet/internal/domain"
)

type ExpenseRepository interface {
	Create(ctx context.Context, e *domain.Expense) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Expense, error)
	List(ctx context.Context, f domain.ExpenseFilter) ([]domain.Expense, error)
	Count(ctx context.Context, f domain.ExpenseFilter) (int, error)
	Update(ctx context.Context, e *domain.Expense) error
	Delete(ctx context.Context, id uuid.UUID) error
	Summarize(ctx context.Context, f domain.ExpenseFilter) (domain.ExpenseSummary, error)
	FindLatest(ctx context.Context, userID int64) (*domain.Expense, error)
	FindLatestMatching(ctx context.Context, userID int64, query string) (*domain.Expense, error)
}

type UserRepository interface {
	Upsert(ctx context.Context, telegramID int64, displayName string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Get(ctx context.Context, telegramID int64) (*domain.User, error)
}

type Repositories struct {
	Expenses ExpenseRepository
	Users    UserRepository
}

// DateOnly normalizes t to midnight UTC date components for DATE columns.
func DateOnly(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}
