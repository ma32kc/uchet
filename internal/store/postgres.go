package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k1ra/uchet/internal/domain"
	"github.com/shopspring/decimal"
)

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func NewRepositories(pool *pgxpool.Pool) Repositories {
	return Repositories{
		Expenses: &expenseRepo{pool: pool},
		Users:    &userRepo{pool: pool},
	}
}

type userRepo struct{ pool *pgxpool.Pool }

func (r *userRepo) Upsert(ctx context.Context, telegramID int64, displayName string) (*domain.User, error) {
	const q = `
INSERT INTO users (telegram_id, display_name)
VALUES ($1, $2)
ON CONFLICT (telegram_id) DO UPDATE
SET display_name = EXCLUDED.display_name,
    updated_at = NOW()
RETURNING telegram_id, display_name, created_at, updated_at`

	var u domain.User
	err := r.pool.QueryRow(ctx, q, telegramID, displayName).Scan(
		&u.TelegramID, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return &u, nil
}

func (r *userRepo) Get(ctx context.Context, telegramID int64) (*domain.User, error) {
	const q = `SELECT telegram_id, display_name, created_at, updated_at FROM users WHERE telegram_id = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, q, telegramID).Scan(
		&u.TelegramID, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (r *userRepo) List(ctx context.Context) ([]domain.User, error) {
	const q = `SELECT telegram_id, display_name, created_at, updated_at FROM users ORDER BY display_name, telegram_id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.TelegramID, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

type expenseRepo struct{ pool *pgxpool.Pool }

func (r *expenseRepo) Create(ctx context.Context, e *domain.Expense) error {
	const q = `
INSERT INTO expenses (id, user_id, amount, category, spent_at, comment, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, q,
		e.ID, e.UserID, e.Amount, string(e.Category), e.SpentAt, e.Comment, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create expense: %w", err)
	}
	return nil
}

func (r *expenseRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Expense, error) {
	const q = `
SELECT id, user_id, amount, category, spent_at, comment, created_at, updated_at
FROM expenses WHERE id = $1`
	e, err := scanExpense(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get expense: %w", err)
	}
	return e, nil
}

func expenseWhere(f domain.ExpenseFilter) (string, []any) {
	q := " WHERE 1=1"
	args := make([]any, 0, 4)
	argN := 1
	if f.UserID != nil {
		q += fmt.Sprintf(" AND user_id = $%d", argN)
		args = append(args, *f.UserID)
		argN++
	}
	if f.Category != nil {
		q += fmt.Sprintf(" AND category = $%d", argN)
		args = append(args, string(*f.Category))
		argN++
	}
	if f.From != nil {
		q += fmt.Sprintf(" AND spent_at >= $%d", argN)
		args = append(args, f.From.Format("2006-01-02"))
		argN++
	}
	if f.To != nil {
		q += fmt.Sprintf(" AND spent_at <= $%d", argN)
		args = append(args, f.To.Format("2006-01-02"))
		argN++
	}
	_ = argN
	return q, args
}

func (r *expenseRepo) List(ctx context.Context, f domain.ExpenseFilter) ([]domain.Expense, error) {
	where, args := expenseWhere(f)
	q := `
SELECT id, user_id, amount, category, spent_at, comment, created_at, updated_at
FROM expenses` + where + ` ORDER BY spent_at DESC, created_at DESC`

	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	argN := len(args) + 1
	q += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, limit, f.Offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	var out []domain.Expense
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *expenseRepo) Count(ctx context.Context, f domain.ExpenseFilter) (int, error) {
	where, args := expenseWhere(f)
	q := `SELECT COUNT(*) FROM expenses` + where
	var n int
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count expenses: %w", err)
	}
	return n, nil
}

func (r *expenseRepo) Update(ctx context.Context, e *domain.Expense) error {
	const q = `
UPDATE expenses
SET amount = $2, category = $3, spent_at = $4, comment = $5, updated_at = $6
WHERE id = $1`

	ct, err := r.pool.Exec(ctx, q,
		e.ID, e.Amount, string(e.Category), e.SpentAt, e.Comment, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update expense: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *expenseRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM expenses WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *expenseRepo) Summarize(ctx context.Context, f domain.ExpenseFilter) (domain.ExpenseSummary, error) {
	q := `SELECT category, COALESCE(SUM(amount), 0), COUNT(*) FROM expenses WHERE 1=1`
	args := make([]any, 0, 4)
	argN := 1

	if f.UserID != nil {
		q += fmt.Sprintf(" AND user_id = $%d", argN)
		args = append(args, *f.UserID)
		argN++
	}
	if f.Category != nil {
		q += fmt.Sprintf(" AND category = $%d", argN)
		args = append(args, string(*f.Category))
		argN++
	}
	if f.From != nil {
		q += fmt.Sprintf(" AND spent_at >= $%d", argN)
		args = append(args, f.From.Format("2006-01-02"))
		argN++
	}
	if f.To != nil {
		q += fmt.Sprintf(" AND spent_at <= $%d", argN)
		args = append(args, f.To.Format("2006-01-02"))
		argN++
	}
	q += " GROUP BY category"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return domain.ExpenseSummary{}, fmt.Errorf("summarize: %w", err)
	}
	defer rows.Close()

	summary := domain.ExpenseSummary{
		ByCategory: make(map[domain.Category]decimal.Decimal),
	}
	for rows.Next() {
		var cat string
		var sum decimal.Decimal
		var count int
		if err := rows.Scan(&cat, &sum, &count); err != nil {
			return domain.ExpenseSummary{}, err
		}
		c := domain.Category(cat)
		summary.ByCategory[c] = sum
		summary.TotalAmount = summary.TotalAmount.Add(sum)
		summary.Count += count
	}
	return summary, rows.Err()
}

func (r *expenseRepo) FindLatest(ctx context.Context, userID int64) (*domain.Expense, error) {
	const q = `
SELECT id, user_id, amount, category, spent_at, comment, created_at, updated_at
FROM expenses WHERE user_id = $1
ORDER BY created_at DESC LIMIT 1`
	e, err := scanExpense(r.pool.QueryRow(ctx, q, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find latest: %w", err)
	}
	return e, nil
}

func (r *expenseRepo) FindLatestMatching(ctx context.Context, userID int64, query string) (*domain.Expense, error) {
	const q = `
SELECT id, user_id, amount, category, spent_at, comment, created_at, updated_at
FROM expenses
WHERE user_id = $1
  AND (
    COALESCE(comment, '') ILIKE '%' || $2 || '%'
    OR category ILIKE '%' || $2 || '%'
  )
ORDER BY created_at DESC
LIMIT 1`
	e, err := scanExpense(r.pool.QueryRow(ctx, q, userID, query))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find matching: %w", err)
	}
	return e, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanExpense(row scannable) (*domain.Expense, error) {
	var e domain.Expense
	var category string
	var amount decimal.Decimal
	err := row.Scan(
		&e.ID, &e.UserID, &amount, &category, &e.SpentAt, &e.Comment, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	e.Amount = amount
	e.Category = domain.Category(category)
	return &e, nil
}
