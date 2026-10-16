export type Category =
  | 'food'
  | 'transport'
  | 'housing'
  | 'entertainment'
  | 'health'
  | 'shopping'
  | 'other';

export const CATEGORIES: { value: Category; label: string }[] = [
  { value: 'food', label: 'еда' },
  { value: 'transport', label: 'транспорт' },
  { value: 'housing', label: 'жильё' },
  { value: 'entertainment', label: 'развлечения' },
  { value: 'health', label: 'здоровье' },
  { value: 'shopping', label: 'покупки' },
  { value: 'other', label: 'другое' },
];

export type User = {
  telegram_id: number;
  display_name: string;
};

export type Expense = {
  id: string;
  user_id: number;
  amount: string;
  category: Category;
  category_label: string;
  spent_at: string;
  comment: string | null;
  created_at: string;
  updated_at: string;
};

export type ExpenseFilters = {
  user_id?: string;
  category?: string;
  from?: string;
  to?: string;
};

export type ExpensePage = {
  items: Expense[];
  total: number;
  limit: number;
  offset: number;
};

export type ExpenseInput = {
  user_id: number;
  amount: string;
  category: string;
  spent_at: string;
  comment?: string;
};

type ApiError = {
  error: { code: string; message: string };
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  });
  if (res.status === 204) {
    return undefined as T;
  }
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const err = data as ApiError;
    throw new Error(err.error?.message || `HTTP ${res.status}`);
  }
  return data as T;
}

export function fetchUsers() {
  return request<User[]>('/api/v1/users');
}

export function fetchExpenses(
  filters: ExpenseFilters,
  page: number,
  pageSize: number,
) {
  const q = new URLSearchParams();
  if (filters.user_id) q.set('user_id', filters.user_id);
  if (filters.category) q.set('category', filters.category);
  if (filters.from) q.set('from', filters.from);
  if (filters.to) q.set('to', filters.to);
  q.set('limit', String(pageSize));
  q.set('offset', String((page - 1) * pageSize));
  return request<ExpensePage>(`/api/v1/expenses?${q.toString()}`);
}

export function createExpense(body: ExpenseInput) {
  return request<Expense>('/api/v1/expenses', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function updateExpense(id: string, body: Partial<ExpenseInput>) {
  return request<Expense>(`/api/v1/expenses/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
}

export function deleteExpense(id: string) {
  return request<void>(`/api/v1/expenses/${id}`, { method: 'DELETE' });
}
