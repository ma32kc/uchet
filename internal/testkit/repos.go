package testkit

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/k1ra/uchet/internal/domain"
	"github.com/k1ra/uchet/internal/store"
	"github.com/shopspring/decimal"
)

type Mem struct {
	mu       sync.Mutex
	Expenses map[uuid.UUID]*domain.Expense
	Users    map[int64]*domain.User
}

func NewMem() *Mem {
	return &Mem{
		Expenses: make(map[uuid.UUID]*domain.Expense),
		Users:    make(map[int64]*domain.User),
	}
}

func (m *Mem) Repositories() store.Repositories {
	return store.Repositories{
		Expenses: &memExpenses{m: m},
		Users:    &memUsers{m: m},
	}
}

type memUsers struct{ m *Mem }

func (r *memUsers) Upsert(_ context.Context, telegramID int64, displayName string) (*domain.User, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	now := time.Now().UTC()
	u, ok := r.m.Users[telegramID]
	if !ok {
		u = &domain.User{TelegramID: telegramID, CreatedAt: now}
		r.m.Users[telegramID] = u
	}
	u.DisplayName = displayName
	u.UpdatedAt = now
	cp := *u
	return &cp, nil
}

func (r *memUsers) Get(_ context.Context, telegramID int64) (*domain.User, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	u, ok := r.m.Users[telegramID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *memUsers) List(_ context.Context) ([]domain.User, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	out := make([]domain.User, 0, len(r.m.Users))
	for _, u := range r.m.Users {
		out = append(out, *u)
	}
	return out, nil
}

type memExpenses struct{ m *Mem }

func (r *memExpenses) Create(_ context.Context, e *domain.Expense) error {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	cp := *e
	r.m.Expenses[e.ID] = &cp
	return nil
}

func (r *memExpenses) GetByID(_ context.Context, id uuid.UUID) (*domain.Expense, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	e, ok := r.m.Expenses[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (r *memExpenses) matched(f domain.ExpenseFilter) []domain.Expense {
	var out []domain.Expense
	for _, e := range r.m.Expenses {
		if f.UserID != nil && e.UserID != *f.UserID {
			continue
		}
		if f.Category != nil && e.Category != *f.Category {
			continue
		}
		if f.From != nil && e.SpentAt.Before(*f.From) {
			continue
		}
		if f.To != nil && e.SpentAt.After(*f.To) {
			continue
		}
		out = append(out, *e)
	}
	return out
}

func (r *memExpenses) List(_ context.Context, f domain.ExpenseFilter) ([]domain.Expense, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	out := r.matched(f)
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(out) {
		return []domain.Expense{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (r *memExpenses) Count(_ context.Context, f domain.ExpenseFilter) (int, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	return len(r.matched(f)), nil
}

func (r *memExpenses) Update(_ context.Context, e *domain.Expense) error {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	if _, ok := r.m.Expenses[e.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *e
	r.m.Expenses[e.ID] = &cp
	return nil
}

func (r *memExpenses) Delete(_ context.Context, id uuid.UUID) error {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	if _, ok := r.m.Expenses[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.m.Expenses, id)
	return nil
}

func (r *memExpenses) Summarize(ctx context.Context, f domain.ExpenseFilter) (domain.ExpenseSummary, error) {
	items, err := r.List(ctx, f)
	if err != nil {
		return domain.ExpenseSummary{}, err
	}
	sum := domain.ExpenseSummary{ByCategory: map[domain.Category]decimal.Decimal{}}
	for _, e := range items {
		sum.Count++
		sum.TotalAmount = sum.TotalAmount.Add(e.Amount)
		sum.ByCategory[e.Category] = sum.ByCategory[e.Category].Add(e.Amount)
	}
	return sum, nil
}

func (r *memExpenses) FindLatest(_ context.Context, userID int64) (*domain.Expense, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	var latest *domain.Expense
	for _, e := range r.m.Expenses {
		if e.UserID != userID {
			continue
		}
		if latest == nil || e.CreatedAt.After(latest.CreatedAt) {
			cp := *e
			latest = &cp
		}
	}
	if latest == nil {
		return nil, domain.ErrNotFound
	}
	return latest, nil
}

func (r *memExpenses) FindLatestMatching(_ context.Context, userID int64, query string) (*domain.Expense, error) {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	q := strings.ToLower(query)
	var latest *domain.Expense
	for _, e := range r.m.Expenses {
		if e.UserID != userID {
			continue
		}
		hay := strings.ToLower(string(e.Category) + " " + e.Category.Label())
		if e.Comment != nil {
			hay += " " + strings.ToLower(*e.Comment)
		}
		if !strings.Contains(hay, q) {
			continue
		}
		if latest == nil || e.CreatedAt.After(latest.CreatedAt) {
			cp := *e
			latest = &cp
		}
	}
	if latest == nil {
		return nil, domain.ErrNotFound
	}
	return latest, nil
}
