package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/k1ra/uchet/internal/domain"
	"github.com/k1ra/uchet/internal/service"
)

type Handler struct {
	svc *service.ExpenseService
	loc *time.Location
}

func NewHandler(svc *service.ExpenseService) *Handler {
	return &Handler{svc: svc, loc: svc.Location()}
}

func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	// Liveness of process; DB readiness checked by composition root via optional ping.
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.ListUsers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]userDTO, 0, len(users))
	for _, u := range users {
		out = append(out, toUserDTO(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) ListExpenses(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r)
	if err != nil {
		writeError(w, err)
		return
	}
	page, err := h.svc.ListPage(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]expenseDTO, 0, len(page.Items))
	for _, e := range page.Items {
		out = append(out, toExpenseDTO(e))
	}
	writeJSON(w, http.StatusOK, expenseListDTO{
		Items:  out,
		Total:  page.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	})
}

func (h *Handler) GetExpense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: invalid id", domain.ErrValidation))
		return
	}
	e, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toExpenseDTO(*e))
}

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var req createExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: invalid json", domain.ErrValidation))
		return
	}
	amount, err := service.ParseAmount(req.Amount)
	if err != nil {
		writeError(w, err)
		return
	}
	cat, err := domain.ParseCategory(req.Category)
	if err != nil {
		writeError(w, err)
		return
	}
	spentAt, err := parseDate(req.SpentAt, h.loc)
	if err != nil {
		writeError(w, err)
		return
	}
	e, err := h.svc.Create(r.Context(), domain.CreateExpenseInput{
		UserID:   req.UserID,
		Amount:   amount,
		Category: cat,
		SpentAt:  spentAt,
		Comment:  req.Comment,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toExpenseDTO(*e))
}

func (h *Handler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: invalid id", domain.ErrValidation))
		return
	}
	var req updateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: invalid json", domain.ErrValidation))
		return
	}
	in := domain.UpdateExpenseInput{}
	if req.Amount != nil {
		amount, err := service.ParseAmount(*req.Amount)
		if err != nil {
			writeError(w, err)
			return
		}
		in.Amount = &amount
	}
	if req.Category != nil {
		cat, err := domain.ParseCategory(*req.Category)
		if err != nil {
			writeError(w, err)
			return
		}
		in.Category = &cat
	}
	if req.SpentAt != nil {
		spentAt, err := parseDate(*req.SpentAt, h.loc)
		if err != nil {
			writeError(w, err)
			return
		}
		in.SpentAt = &spentAt
	}
	if req.Comment != nil {
		if *req.Comment == "" {
			in.ClearComment = true
		} else {
			in.Comment = req.Comment
		}
	}
	e, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toExpenseDTO(*e))
}

func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: invalid id", domain.ErrValidation))
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseFilter(r *http.Request) (domain.ExpenseFilter, error) {
	q := r.URL.Query()
	f := domain.ExpenseFilter{}

	if v := q.Get("user_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return f, fmt.Errorf("%w: invalid user_id", domain.ErrValidation)
		}
		f.UserID = &id
	}
	if v := q.Get("category"); v != "" {
		cat, err := domain.ParseCategory(v)
		if err != nil {
			return f, err
		}
		f.Category = &cat
	}
	if v := q.Get("from"); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, time.UTC)
		if err != nil {
			return f, fmt.Errorf("%w: invalid from", domain.ErrValidation)
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, time.UTC)
		if err != nil {
			return f, fmt.Errorf("%w: invalid to", domain.ErrValidation)
		}
		f.To = &t
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, fmt.Errorf("%w: invalid limit", domain.ErrValidation)
		}
		f.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, fmt.Errorf("%w: invalid offset", domain.ErrValidation)
		}
		f.Offset = n
	}
	return f, nil
}

func parseDate(s string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}
	t, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: invalid date, want YYYY-MM-DD", domain.ErrValidation)
	}
	return t, nil
}
