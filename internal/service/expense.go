package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/k1ra/uchet/internal/domain"
	"github.com/k1ra/uchet/internal/store"
	"github.com/shopspring/decimal"
)

type ExpenseService struct {
	expenses store.ExpenseRepository
	users    store.UserRepository
	location *time.Location
}

func NewExpenseService(repos store.Repositories, loc *time.Location) *ExpenseService {
	if loc == nil {
		loc = time.UTC
	}
	return &ExpenseService{
		expenses: repos.Expenses,
		users:    repos.Users,
		location: loc,
	}
}

func (s *ExpenseService) UpsertUser(ctx context.Context, telegramID int64, displayName string) (*domain.User, error) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = fmt.Sprintf("user_%d", telegramID)
	}
	return s.users.Upsert(ctx, telegramID, name)
}

func (s *ExpenseService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.users.List(ctx)
}

func (s *ExpenseService) Create(ctx context.Context, in domain.CreateExpenseInput) (*domain.Expense, error) {
	if err := validateCreate(in); err != nil {
		return nil, err
	}
	// Ensure user exists for FK (panel may create for known telegram id).
	if _, err := s.users.Get(ctx, in.UserID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			if _, err := s.users.Upsert(ctx, in.UserID, fmt.Sprintf("user_%d", in.UserID)); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	now := time.Now().UTC()
	e := &domain.Expense{
		ID:        uuid.New(),
		UserID:    in.UserID,
		Amount:    in.Amount,
		Category:  in.Category,
		SpentAt:   store.DateOnly(in.SpentAt, s.location),
		Comment:   normalizeComment(in.Comment),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.expenses.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// CreateForUser forces owner to actorUserID (bot mode).
func (s *ExpenseService) CreateForUser(ctx context.Context, actorUserID int64, in domain.CreateExpenseInput) (*domain.Expense, error) {
	in.UserID = actorUserID
	return s.Create(ctx, in)
}

func (s *ExpenseService) Get(ctx context.Context, id uuid.UUID) (*domain.Expense, error) {
	return s.expenses.GetByID(ctx, id)
}

func (s *ExpenseService) GetForUser(ctx context.Context, actorUserID int64, id uuid.UUID) (*domain.Expense, error) {
	e, err := s.expenses.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.UserID != actorUserID {
		return nil, domain.ErrForbidden
	}
	return e, nil
}

func (s *ExpenseService) List(ctx context.Context, f domain.ExpenseFilter) ([]domain.Expense, error) {
	return s.expenses.List(ctx, f)
}

func (s *ExpenseService) Count(ctx context.Context, f domain.ExpenseFilter) (int, error) {
	return s.expenses.Count(ctx, f)
}

type PageResult struct {
	Items  []domain.Expense
	Total  int
	Limit  int
	Offset int
}

func (s *ExpenseService) ListPage(ctx context.Context, f domain.ExpenseFilter) (PageResult, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	total, err := s.expenses.Count(ctx, f)
	if err != nil {
		return PageResult{}, err
	}
	items, err := s.expenses.List(ctx, f)
	if err != nil {
		return PageResult{}, err
	}
	return PageResult{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

func (s *ExpenseService) ListForUser(ctx context.Context, actorUserID int64, f domain.ExpenseFilter) ([]domain.Expense, error) {
	f.UserID = &actorUserID
	return s.expenses.List(ctx, f)
}

func (s *ExpenseService) Update(ctx context.Context, id uuid.UUID, in domain.UpdateExpenseInput) (*domain.Expense, error) {
	e, err := s.expenses.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := applyUpdate(e, in); err != nil {
		return nil, err
	}
	e.SpentAt = store.DateOnly(e.SpentAt, s.location)
	e.UpdatedAt = time.Now().UTC()
	if err := s.expenses.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *ExpenseService) UpdateForUser(ctx context.Context, actorUserID int64, id uuid.UUID, in domain.UpdateExpenseInput) (*domain.Expense, error) {
	e, err := s.GetForUser(ctx, actorUserID, id)
	if err != nil {
		return nil, err
	}
	if err := applyUpdate(e, in); err != nil {
		return nil, err
	}
	e.SpentAt = store.DateOnly(e.SpentAt, s.location)
	e.UpdatedAt = time.Now().UTC()
	if err := s.expenses.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *ExpenseService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.expenses.Delete(ctx, id)
}

func (s *ExpenseService) DeleteForUser(ctx context.Context, actorUserID int64, id uuid.UUID) error {
	if _, err := s.GetForUser(ctx, actorUserID, id); err != nil {
		return err
	}
	return s.expenses.Delete(ctx, id)
}

func (s *ExpenseService) Summarize(ctx context.Context, f domain.ExpenseFilter) (domain.ExpenseSummary, error) {
	return s.expenses.Summarize(ctx, f)
}

func (s *ExpenseService) SummarizeForUser(ctx context.Context, actorUserID int64, f domain.ExpenseFilter) (domain.ExpenseSummary, error) {
	f.UserID = &actorUserID
	return s.expenses.Summarize(ctx, f)
}

func (s *ExpenseService) ResolveLatest(ctx context.Context, actorUserID int64, query string) (*domain.Expense, error) {
	query = strings.TrimSpace(query)
	if query == "" || strings.EqualFold(query, "last") || strings.EqualFold(query, "последняя") || strings.EqualFold(query, "последнее") {
		return s.expenses.FindLatest(ctx, actorUserID)
	}
	// Try category label/code first.
	if cat, err := domain.ParseCategory(query); err == nil {
		list, err := s.expenses.List(ctx, domain.ExpenseFilter{
			UserID:   &actorUserID,
			Category: &cat,
			Limit:    1,
		})
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, domain.ErrNotFound
		}
		return &list[0], nil
	}
	e, err := s.expenses.FindLatestMatching(ctx, actorUserID, query)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *ExpenseService) Location() *time.Location {
	return s.location
}

func validateCreate(in domain.CreateExpenseInput) error {
	if in.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", domain.ErrValidation)
	}
	if !in.Amount.IsPositive() {
		return fmt.Errorf("%w: amount must be positive", domain.ErrValidation)
	}
	if !in.Category.Valid() {
		return fmt.Errorf("%w: invalid category", domain.ErrValidation)
	}
	if in.SpentAt.IsZero() {
		return fmt.Errorf("%w: spent_at is required", domain.ErrValidation)
	}
	return nil
}

func applyUpdate(e *domain.Expense, in domain.UpdateExpenseInput) error {
	if in.Amount != nil {
		if !in.Amount.IsPositive() {
			return fmt.Errorf("%w: amount must be positive", domain.ErrValidation)
		}
		e.Amount = *in.Amount
	}
	if in.Category != nil {
		if !in.Category.Valid() {
			return fmt.Errorf("%w: invalid category", domain.ErrValidation)
		}
		e.Category = *in.Category
	}
	if in.SpentAt != nil {
		if in.SpentAt.IsZero() {
			return fmt.Errorf("%w: spent_at is invalid", domain.ErrValidation)
		}
		e.SpentAt = *in.SpentAt
	}
	if in.ClearComment {
		e.Comment = nil
	} else if in.Comment != nil {
		e.Comment = normalizeComment(in.Comment)
	}
	return nil
}

func normalizeComment(c *string) *string {
	if c == nil {
		return nil
	}
	v := strings.TrimSpace(*c)
	if v == "" {
		return nil
	}
	return &v
}

// ParseAmount parses decimal amount from string.
func ParseAmount(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%w: invalid amount", domain.ErrValidation)
	}
	if !d.IsPositive() {
		return decimal.Zero, fmt.Errorf("%w: amount must be positive", domain.ErrValidation)
	}
	return d, nil
}
