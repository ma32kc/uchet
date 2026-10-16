package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/k1ra/uchet/internal/domain"
	"github.com/k1ra/uchet/internal/service"
	"github.com/shopspring/decimal"
)

func toolSpecs() []toolSpec {
	return []toolSpec{
		{
			Type: "function",
			Function: toolSpecFunc{
				Name:        "add_expense",
				Description: "Add a new expense for the current user",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"amount":  map[string]any{"type": "number", "description": "Expense amount"},
						"category": map[string]any{
							"type": "string",
							"enum": []string{"food", "transport", "housing", "entertainment", "health", "shopping", "other"},
						},
						"spent_at": map[string]any{"type": "string", "description": "YYYY-MM-DD; omit for today"},
						"comment":  map[string]any{"type": "string"},
					},
					"required": []string{"amount", "category"},
				},
			},
		},
		{
			Type: "function",
			Function: toolSpecFunc{
				Name:        "list_expenses",
				Description: "List current user expenses with optional filters",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"from":     map[string]any{"type": "string", "description": "YYYY-MM-DD"},
						"to":       map[string]any{"type": "string", "description": "YYYY-MM-DD"},
						"category": map[string]any{"type": "string"},
						"limit":    map[string]any{"type": "integer"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: toolSpecFunc{
				Name:        "summarize_expenses",
				Description: "Summarize totals for current user",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"from":     map[string]any{"type": "string"},
						"to":       map[string]any{"type": "string"},
						"category": map[string]any{"type": "string"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: toolSpecFunc{
				Name:        "update_expense",
				Description: "Update an expense by id, or resolve by query (e.g. last, taxi)",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":       map[string]any{"type": "string"},
						"query":    map[string]any{"type": "string", "description": "last / category / comment fragment"},
						"amount":   map[string]any{"type": "number"},
						"category": map[string]any{"type": "string"},
						"spent_at": map[string]any{"type": "string"},
						"comment":  map[string]any{"type": "string"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: toolSpecFunc{
				Name:        "delete_expense",
				Description: "Delete an expense by id or query",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":    map[string]any{"type": "string"},
						"query": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

type toolExecutor struct {
	svc    *service.ExpenseService
	userID int64
	loc    *time.Location
}

func (t *toolExecutor) execute(ctx context.Context, name, argsJSON string) (string, error) {
	var args map[string]any
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid tool args: %w", err)
		}
	}
	switch name {
	case "add_expense":
		return t.add(ctx, args)
	case "list_expenses":
		return t.list(ctx, args)
	case "summarize_expenses":
		return t.summarize(ctx, args)
	case "update_expense":
		return t.update(ctx, args)
	case "delete_expense":
		return t.delete(ctx, args)
	default:
		return "", fmt.Errorf("unknown tool %s", name)
	}
}

func (t *toolExecutor) add(ctx context.Context, args map[string]any) (string, error) {
	amount, err := numberArg(args, "amount")
	if err != nil {
		return "", err
	}
	cat, err := categoryArg(args, "category")
	if err != nil {
		return "", err
	}
	spentAt := time.Now().In(t.loc)
	if v, ok := args["spent_at"].(string); ok && strings.TrimSpace(v) != "" {
		spentAt, err = time.ParseInLocation("2006-01-02", v, t.loc)
		if err != nil {
			return "", fmt.Errorf("%w: spent_at", domain.ErrValidation)
		}
	}
	var comment *string
	if v, ok := args["comment"].(string); ok {
		comment = &v
	}
	e, err := t.svc.CreateForUser(ctx, t.userID, domain.CreateExpenseInput{
		Amount:   amount,
		Category: cat,
		SpentAt:  spentAt,
		Comment:  comment,
	})
	if err != nil {
		return "", err
	}
	return "Добавлено: " + formatExpense(*e), nil
}

func (t *toolExecutor) list(ctx context.Context, args map[string]any) (string, error) {
	f, err := filterFromArgs(args, t.loc)
	if err != nil {
		return "", err
	}
	if f.Limit == 0 {
		f.Limit = 20
	}
	items, err := t.svc.ListForUser(ctx, t.userID, f)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "Трат не найдено.", nil
	}
	var b strings.Builder
	for _, e := range items {
		b.WriteString(formatExpense(e))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String()), nil
}

func (t *toolExecutor) summarize(ctx context.Context, args map[string]any) (string, error) {
	f, err := filterFromArgs(args, t.loc)
	if err != nil {
		return "", err
	}
	sum, err := t.svc.SummarizeForUser(ctx, t.userID, f)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Всего: %s (%d шт.)\n", sum.TotalAmount.StringFixed(2), sum.Count)
	for _, cat := range domain.AllCategories {
		if v, ok := sum.ByCategory[cat]; ok {
			fmt.Fprintf(&b, "- %s: %s\n", cat.Label(), v.StringFixed(2))
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func (t *toolExecutor) update(ctx context.Context, args map[string]any) (string, error) {
	e, err := t.resolve(ctx, args)
	if err != nil {
		return "", err
	}
	in := domain.UpdateExpenseInput{}
	if _, ok := args["amount"]; ok {
		amount, err := numberArg(args, "amount")
		if err != nil {
			return "", err
		}
		in.Amount = &amount
	}
	if _, ok := args["category"]; ok {
		cat, err := categoryArg(args, "category")
		if err != nil {
			return "", err
		}
		in.Category = &cat
	}
	if v, ok := args["spent_at"].(string); ok && v != "" {
		spentAt, err := time.ParseInLocation("2006-01-02", v, t.loc)
		if err != nil {
			return "", fmt.Errorf("%w: spent_at", domain.ErrValidation)
		}
		in.SpentAt = &spentAt
	}
	if v, ok := args["comment"].(string); ok {
		in.Comment = &v
	}
	updated, err := t.svc.UpdateForUser(ctx, t.userID, e.ID, in)
	if err != nil {
		return "", err
	}
	return "Обновлено: " + formatExpense(*updated), nil
}

func (t *toolExecutor) delete(ctx context.Context, args map[string]any) (string, error) {
	e, err := t.resolve(ctx, args)
	if err != nil {
		return "", err
	}
	if err := t.svc.DeleteForUser(ctx, t.userID, e.ID); err != nil {
		return "", err
	}
	return "Удалено: " + formatExpense(*e), nil
}

func (t *toolExecutor) resolve(ctx context.Context, args map[string]any) (*domain.Expense, error) {
	if idStr, ok := args["id"].(string); ok && strings.TrimSpace(idStr) != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid id", domain.ErrValidation)
		}
		return t.svc.GetForUser(ctx, t.userID, id)
	}
	query, _ := args["query"].(string)
	return t.svc.ResolveLatest(ctx, t.userID, query)
}

func filterFromArgs(args map[string]any, loc *time.Location) (domain.ExpenseFilter, error) {
	f := domain.ExpenseFilter{}
	if v, ok := args["from"].(string); ok && v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, loc)
		if err != nil {
			return f, fmt.Errorf("%w: from", domain.ErrValidation)
		}
		f.From = &t
	}
	if v, ok := args["to"].(string); ok && v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, loc)
		if err != nil {
			return f, fmt.Errorf("%w: to", domain.ErrValidation)
		}
		f.To = &t
	}
	if v, ok := args["category"].(string); ok && v != "" {
		cat, err := domain.ParseCategory(v)
		if err != nil {
			return f, err
		}
		f.Category = &cat
	}
	if v, ok := args["limit"].(float64); ok {
		f.Limit = int(v)
	}
	return f, nil
}

func numberArg(args map[string]any, key string) (decimal.Decimal, error) {
	v, ok := args[key]
	if !ok {
		return decimal.Zero, fmt.Errorf("%w: %s required", domain.ErrValidation, key)
	}
	switch n := v.(type) {
	case float64:
		return decimal.NewFromFloat(n), nil
	case string:
		return service.ParseAmount(n)
	case json.Number:
		return service.ParseAmount(n.String())
	default:
		return decimal.Zero, fmt.Errorf("%w: invalid %s", domain.ErrValidation, key)
	}
}

func categoryArg(args map[string]any, key string) (domain.Category, error) {
	v, ok := args[key].(string)
	if !ok || strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("%w: %s required", domain.ErrValidation, key)
	}
	return domain.ParseCategory(v)
}

func formatExpense(e domain.Expense) string {
	comment := ""
	if e.Comment != nil && *e.Comment != "" {
		comment = ", " + *e.Comment
	}
	return fmt.Sprintf("%s — %s ₽, %s%s",
		e.SpentAt.Format("2006-01-02"),
		e.Amount.StringFixed(2),
		e.Category.Label(),
		comment,
	)
}
