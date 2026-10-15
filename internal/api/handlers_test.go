package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/k1ra/uchet/internal/service"
	"github.com/k1ra/uchet/internal/testkit"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	mem := testkit.NewMem()
	svc := service.NewExpenseService(mem.Repositories(), time.UTC)
	return NewRouter(NewHandler(svc), nil)
}

func TestHealthz(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestCreateListFilterExpenses(t *testing.T) {
	r := newTestRouter(t)

	body := map[string]any{
		"user_id":  42,
		"amount":   "350.50",
		"category": "еда",
		"spent_at": "2026-10-15",
		"comment":  "кофе",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d body %s", rec.Code, rec.Body.String())
	}

	var created expenseDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.UserID != 42 || created.Category != "food" || created.Amount != "350.50" {
		t.Fatalf("unexpected create response: %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/expenses?user_id=42&category=food", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d", rec.Code)
	}
	var list expenseListDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Items) != 1 {
		t.Fatalf("want 1 expense, got total=%d items=%d", list.Total, len(list.Items))
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("users status %d", rec.Code)
	}
	var users []userDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].TelegramID != 42 {
		t.Fatalf("users: %+v", users)
	}
}

func TestCreateValidationError(t *testing.T) {
	r := newTestRouter(t)
	raw, _ := json.Marshal(map[string]any{
		"user_id": 1, "amount": "-10", "category": "food", "spent_at": "2026-10-15",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateAndDeleteExpense(t *testing.T) {
	r := newTestRouter(t)
	raw, _ := json.Marshal(map[string]any{
		"user_id": 7, "amount": "100", "category": "transport", "spent_at": "2026-10-01", "comment": "такси",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var created expenseDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	patch, _ := json.Marshal(map[string]any{"amount": "120"})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/expenses/"+created.ID, bytes.NewReader(patch))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/expenses/"+created.ID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/expenses/"+created.ID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}
