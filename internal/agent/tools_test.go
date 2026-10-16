package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/k1ra/uchet/internal/service"
	"github.com/k1ra/uchet/internal/testkit"
)

func newToolExec(userID int64) (*toolExecutor, *testkit.Mem) {
	mem := testkit.NewMem()
	svc := service.NewExpenseService(mem.Repositories(), time.UTC)
	return &toolExecutor{svc: svc, userID: userID, loc: time.UTC}, mem
}

func TestToolAddListSummarizeScopedToUser(t *testing.T) {
	_, mem := newToolExec(1)
	svc := service.NewExpenseService(mem.Repositories(), time.UTC)
	execA := &toolExecutor{svc: svc, userID: 1, loc: time.UTC}
	execB := &toolExecutor{svc: svc, userID: 2, loc: time.UTC}

	ctx := context.Background()
	if _, err := execA.execute(ctx, "add_expense", `{"amount":350,"category":"food","comment":"кофе"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := execB.execute(ctx, "add_expense", `{"amount":1200,"category":"transport","comment":"такси"}`); err != nil {
		t.Fatal(err)
	}

	out, err := execA.execute(ctx, "list_expenses", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "кофе") || strings.Contains(out, "такси") {
		t.Fatalf("user1 list leaked: %s", out)
	}

	sum, err := execA.execute(ctx, "summarize_expenses", `{"category":"food"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sum, "350.00") {
		t.Fatalf("summary: %s", sum)
	}
}

func TestToolUpdateAndDeleteByQuery(t *testing.T) {
	exec, _ := newToolExec(10)
	ctx := context.Background()
	if _, err := exec.execute(ctx, "add_expense", `{"amount":350,"category":"food","comment":"кофе"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.execute(ctx, "add_expense", `{"amount":1200,"category":"transport","comment":"такси"}`); err != nil {
		t.Fatal(err)
	}

	upd, err := exec.execute(ctx, "update_expense", `{"query":"последнее","amount":450}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(upd, "450.00") {
		t.Fatalf("update: %s", upd)
	}

	del, err := exec.execute(ctx, "delete_expense", `{"query":"такси"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(del, "Удалено") {
		t.Fatalf("delete: %s", del)
	}

	list, err := exec.execute(ctx, "list_expenses", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(list, "такси") {
		t.Fatalf("taxi still present: %s", list)
	}
}

func TestToolRejectsUnknownCategory(t *testing.T) {
	exec, _ := newToolExec(1)
	_, err := exec.execute(context.Background(), "add_expense", `{"amount":10,"category":"crypto"}`)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
