package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/k1ra/uchet/internal/domain"
	"github.com/k1ra/uchet/internal/testkit"
	"github.com/shopspring/decimal"
)

func TestResolveLatestByLastAndComment(t *testing.T) {
	mem := testkit.NewMem()
	svc := NewExpenseService(mem.Repositories(), time.UTC)
	ctx := context.Background()

	_, _ = svc.UpsertUser(ctx, 5, "Ann")
	older := &domain.Expense{
		ID: uuid.New(), UserID: 5, Amount: decimal.NewFromInt(100),
		Category: domain.CategoryFood, SpentAt: time.Now(), Comment: strPtr("кофе"),
		CreatedAt: time.Now().Add(-time.Minute), UpdatedAt: time.Now(),
	}
	newer := &domain.Expense{
		ID: uuid.New(), UserID: 5, Amount: decimal.NewFromInt(200),
		Category: domain.CategoryTransport, SpentAt: time.Now(), Comment: strPtr("такси"),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	_ = mem.Repositories().Expenses.Create(ctx, older)
	_ = mem.Repositories().Expenses.Create(ctx, newer)

	last, err := svc.ResolveLatest(ctx, 5, "последнее")
	if err != nil {
		t.Fatal(err)
	}
	if last.ID != newer.ID {
		t.Fatalf("want newer, got %s", last.ID)
	}

	taxi, err := svc.ResolveLatest(ctx, 5, "такси")
	if err != nil {
		t.Fatal(err)
	}
	if taxi.ID != newer.ID {
		t.Fatalf("want taxi expense")
	}

	food, err := svc.ResolveLatest(ctx, 5, "еда")
	if err != nil {
		t.Fatal(err)
	}
	if food.Category != domain.CategoryFood {
		t.Fatalf("want food, got %s", food.Category)
	}
}

func TestParseAmount(t *testing.T) {
	d, err := ParseAmount("350,5")
	if err != nil {
		t.Fatal(err)
	}
	if !d.Equal(decimal.RequireFromString("350.5")) {
		t.Fatalf("got %s", d)
	}
	if _, err := ParseAmount("-1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteForUserForbidden(t *testing.T) {
	mem := testkit.NewMem()
	svc := NewExpenseService(mem.Repositories(), time.UTC)
	ctx := context.Background()
	_, _ = svc.UpsertUser(ctx, 1, "A")

	e, err := svc.CreateForUser(ctx, 1, domain.CreateExpenseInput{
		Amount: decimal.NewFromInt(10), Category: domain.CategoryOther, SpentAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteForUser(ctx, 2, e.ID); err != domain.ErrForbidden {
		t.Fatalf("got %v", err)
	}
}
