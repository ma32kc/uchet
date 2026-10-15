package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/k1ra/uchet/internal/domain"
	"github.com/k1ra/uchet/internal/store"
	"github.com/shopspring/decimal"
)

type mockExpenses struct {
	byID   map[uuid.UUID]*domain.Expense
	create []*domain.Expense
}

func newMockExpenses() *mockExpenses {
	return &mockExpenses{byID: make(map[uuid.UUID]*domain.Expense)}
}

func (m *mockExpenses) Create(_ context.Context, e *domain.Expense) error {
	cp := *e
	m.byID[e.ID] = &cp
	m.create = append(m.create, &cp)
	return nil
}
func (m *mockExpenses) GetByID(_ context.Context, id uuid.UUID) (*domain.Expense, error) {
	e, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *e
	return &cp, nil
}
func (m *mockExpenses) List(_ context.Context, f domain.ExpenseFilter) ([]domain.Expense, error) {
	var out []domain.Expense
	for _, e := range m.byID {
		if f.UserID != nil && e.UserID != *f.UserID {
			continue
		}
		out = append(out, *e)
	}
	return out, nil
}
func (m *mockExpenses) Update(_ context.Context, e *domain.Expense) error {
	if _, ok := m.byID[e.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *e
	m.byID[e.ID] = &cp
	return nil
}
func (m *mockExpenses) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.byID, id)
	return nil
}
func (m *mockExpenses) Count(_ context.Context, f domain.ExpenseFilter) (int, error) {
	n := 0
	for _, e := range m.byID {
		if f.UserID != nil && e.UserID != *f.UserID {
			continue
		}
		n++
	}
	return n, nil
}
func (m *mockExpenses) Summarize(context.Context, domain.ExpenseFilter) (domain.ExpenseSummary, error) {
	return domain.ExpenseSummary{ByCategory: map[domain.Category]decimal.Decimal{}}, nil
}
func (m *mockExpenses) FindLatest(_ context.Context, userID int64) (*domain.Expense, error) {
	var latest *domain.Expense
	for _, e := range m.byID {
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
func (m *mockExpenses) FindLatestMatching(context.Context, int64, string) (*domain.Expense, error) {
	return nil, domain.ErrNotFound
}

type mockUsers struct {
	users map[int64]*domain.User
}

func newMockUsers() *mockUsers {
	return &mockUsers{users: make(map[int64]*domain.User)}
}

func (m *mockUsers) Upsert(_ context.Context, telegramID int64, displayName string) (*domain.User, error) {
	u := &domain.User{TelegramID: telegramID, DisplayName: displayName, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.users[telegramID] = u
	return u, nil
}
func (m *mockUsers) List(context.Context) ([]domain.User, error) { return nil, nil }
func (m *mockUsers) Get(_ context.Context, telegramID int64) (*domain.User, error) {
	u, ok := m.users[telegramID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func TestCreateForUserForcesOwner(t *testing.T) {
	exp := newMockExpenses()
	users := newMockUsers()
	if _, err := users.Upsert(context.Background(), 42, "Alice"); err != nil {
		t.Fatal(err)
	}
	svc := NewExpenseService(store.Repositories{Expenses: exp, Users: users}, time.UTC)

	got, err := svc.CreateForUser(context.Background(), 42, domain.CreateExpenseInput{
		UserID:   999, // should be ignored
		Amount:   decimal.NewFromInt(350),
		Category: domain.CategoryFood,
		SpentAt:  time.Now(),
		Comment:  strPtr("кофе"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != 42 {
		t.Fatalf("user_id = %d, want 42", got.UserID)
	}
}

func TestGetForUserForbidden(t *testing.T) {
	exp := newMockExpenses()
	users := newMockUsers()
	id := uuid.New()
	exp.byID[id] = &domain.Expense{
		ID: id, UserID: 1, Amount: decimal.NewFromInt(10),
		Category: domain.CategoryOther, SpentAt: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	svc := NewExpenseService(store.Repositories{Expenses: exp, Users: users}, time.UTC)

	_, err := svc.GetForUser(context.Background(), 2, id)
	if err != domain.ErrForbidden {
		t.Fatalf("got %v, want forbidden", err)
	}
}

func TestUpdateForUserCannotTouchOthers(t *testing.T) {
	exp := newMockExpenses()
	users := newMockUsers()
	id := uuid.New()
	exp.byID[id] = &domain.Expense{
		ID: id, UserID: 1, Amount: decimal.NewFromInt(10),
		Category: domain.CategoryOther, SpentAt: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	svc := NewExpenseService(store.Repositories{Expenses: exp, Users: users}, time.UTC)

	amount := decimal.NewFromInt(99)
	_, err := svc.UpdateForUser(context.Background(), 2, id, domain.UpdateExpenseInput{Amount: &amount})
	if err != domain.ErrForbidden {
		t.Fatalf("got %v, want forbidden", err)
	}
}

func TestValidateCreate(t *testing.T) {
	err := validateCreate(domain.CreateExpenseInput{
		UserID: 1, Amount: decimal.Zero, Category: domain.CategoryFood, SpentAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func strPtr(s string) *string { return &s }
