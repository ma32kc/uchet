package agent

import (
	"context"
	"testing"
	"time"

	"github.com/k1ra/uchet/internal/service"
	"github.com/k1ra/uchet/internal/testkit"
)

func TestHandleStartWithoutLLM(t *testing.T) {
	mem := testkit.NewMem()
	svc := service.NewExpenseService(mem.Repositories(), time.UTC)
	a := New(nil, svc, time.UTC, 3)

	reply, err := a.Handle(context.Background(), 1, 1, "/start")
	if err != nil {
		t.Fatal(err)
	}
	if reply == "" {
		t.Fatal("empty reply")
	}
}

func TestHandleEmptyText(t *testing.T) {
	mem := testkit.NewMem()
	svc := service.NewExpenseService(mem.Repositories(), time.UTC)
	a := New(nil, svc, time.UTC, 3)

	reply, err := a.Handle(context.Background(), 1, 1, "   ")
	if err != nil {
		t.Fatal(err)
	}
	if reply == "" {
		t.Fatal("expected hint")
	}
}

func TestStartOfWeekMonday(t *testing.T) {
	// 2026-10-16 is Friday
	fri := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	got := startOfWeek(fri)
	if got.Weekday() != time.Monday {
		t.Fatalf("want Monday, got %s", got.Weekday())
	}
	if got.Day() != 12 {
		t.Fatalf("want 12, got %d", got.Day())
	}
}
