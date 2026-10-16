import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import {
  ActionIcon,
  Alert,
  Avatar,
  Badge,
  Button,
  Container,
  Group,
  Image,
  Modal,
  Pagination,
  Paper,
  Select,
  Stack,
  Table,
  Text,
  Title,
  Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import {
  CATEGORIES,
  createExpense,
  deleteExpense,
  fetchExpenses,
  fetchUsers,
  updateExpense,
  type Expense,
  type User,
} from './api';

type Filters = {
  user_id: string;
  category: string;
  from: string;
  to: string;
};

type FormState = {
  user_id: string;
  amount: string;
  category: string;
  spent_at: string;
  comment: string;
};

const PAGE_SIZE = 10;

const emptyFilters = (): Filters => ({
  user_id: '',
  category: '',
  from: '',
  to: '',
});

const emptyForm = (): FormState => ({
  user_id: '',
  amount: '',
  category: 'food',
  spent_at: new Date().toISOString().slice(0, 10),
  comment: '',
});

function formatMoney(value: string | number) {
  const n = Number(value);
  if (Number.isNaN(n)) return '0.00';
  return n.toLocaleString('ru-RU', { minimumFractionDigits: 2 });
}

function Field({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      {children}
    </label>
  );
}

export default function App() {
  const [users, setUsers] = useState<User[]>([]);
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [opened, { open, close }] = useDisclosure(false);
  const [editing, setEditing] = useState<Expense | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);

  const userName = useCallback(
    (id: number) => {
      const u = users.find((x) => x.telegram_id === id);
      return u?.display_name || String(id);
    },
    [users],
  );

  const apiFilters = useMemo(
    () => ({
      user_id: filters.user_id || undefined,
      category: filters.category || undefined,
      from: filters.from || undefined,
      to: filters.to || undefined,
    }),
    [filters],
  );

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [u, pageData] = await Promise.all([
        fetchUsers(),
        fetchExpenses(apiFilters, page, PAGE_SIZE),
      ]);
      setUsers(u);
      setExpenses(pageData.items ?? []);
      setTotal(pageData.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка загрузки');
    } finally {
      setLoading(false);
    }
  }, [apiFilters, page]);

  useEffect(() => {
    void load();
  }, [load]);

  // Reset to first page when filters change (without fighting load).
  useEffect(() => {
    setPage(1);
  }, [filters.user_id, filters.category, filters.from, filters.to]);

  useEffect(() => {
    const maxPage = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1);
    if (page > maxPage) setPage(maxPage);
  }, [total, page]);

  const pageTotal = useMemo(
    () => expenses.reduce((acc, e) => acc + Number(e.amount), 0),
    [expenses],
  );
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1);

  function openCreate() {
    setEditing(null);
    const next = emptyForm();
    if (users[0]) next.user_id = String(users[0].telegram_id);
    setForm(next);
    open();
  }

  function openEdit(e: Expense) {
    setEditing(e);
    setForm({
      user_id: String(e.user_id),
      amount: String(e.amount ?? ''),
      category: e.category || 'food',
      spent_at: e.spent_at || '',
      comment: e.comment ?? '',
    });
    open();
  }

  async function onSubmit() {
    if (!form.spent_at || !form.amount.trim()) {
      notifications.show({ color: 'red', message: 'Заполните сумму и дату' });
      return;
    }
    if (!editing && !form.user_id) {
      notifications.show({ color: 'red', message: 'Укажите участника' });
      return;
    }

    setSaving(true);
    setError(null);
    try {
      const amount = form.amount.trim().replace(',', '.');
      if (editing) {
        await updateExpense(editing.id, {
          amount,
          category: form.category,
          spent_at: form.spent_at,
          comment: form.comment,
        });
      } else {
        await createExpense({
          user_id: Number(form.user_id),
          amount,
          category: form.category,
          spent_at: form.spent_at,
          comment: form.comment || undefined,
        });
      }
      close();
      notifications.show({
        color: 'teal',
        message: editing ? 'Трата обновлена' : 'Трата добавлена',
      });
      await load();
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Ошибка сохранения';
      setError(msg);
      notifications.show({ color: 'red', message: msg });
    } finally {
      setSaving(false);
    }
  }

  async function onDelete(e: Expense) {
    if (!confirm(`Удалить трату ${formatMoney(e.amount)} ₽ (${e.category_label})?`)) return;
    try {
      await deleteExpense(e.id);
      notifications.show({ color: 'teal', message: 'Трата удалена' });
      await load();
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Ошибка удаления';
      setError(msg);
      notifications.show({ color: 'red', message: msg });
    }
  }

  const userOptions = users.map((u) => ({
    value: String(u.telegram_id),
    label: u.display_name || String(u.telegram_id),
  }));

  const categoryOptions = CATEGORIES.map((c) => ({
    value: c.value,
    label: c.label,
  }));

  const rows = expenses.map((e) => (
    <Table.Tr key={e.id}>
      <Table.Td>{e.spent_at}</Table.Td>
      <Table.Td>{userName(e.user_id)}</Table.Td>
      <Table.Td>
        <Badge variant="light" color="teal" radius="sm">
          {e.category_label}
        </Badge>
      </Table.Td>
      <Table.Td style={{ fontVariantNumeric: 'tabular-nums', fontWeight: 600 }}>
        {formatMoney(e.amount)} ₽
      </Table.Td>
      <Table.Td c="dimmed">{e.comment || '—'}</Table.Td>
      <Table.Td>
        <Group gap={6} justify="flex-end" wrap="nowrap">
          <Tooltip label="Изменить">
            <ActionIcon variant="subtle" color="gray" onClick={() => openEdit(e)} aria-label="Изменить">
              ✎
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Удалить">
            <ActionIcon variant="subtle" color="red" onClick={() => void onDelete(e)} aria-label="Удалить">
              ⌫
            </ActionIcon>
          </Tooltip>
        </Group>
      </Table.Td>
    </Table.Tr>
  ));

  return (
    <div className="app-shell">
      <Container size="lg" py="xl">
        <Stack gap="lg">
          <Group justify="space-between" align="center" wrap="nowrap">
            <Group gap="md" wrap="nowrap">
              <Avatar
                src="/mascot.jpg"
                alt="Маскот"
                size={72}
                radius="xl"
                className="mascot-avatar"
              />
              <div>
                <Title order={1}>Семейные расходы</Title>
                <Text c="dimmed" size="sm" mt={4}>
                  Панель главы семьи
                </Text>
              </div>
            </Group>
            <Button color="teal" onClick={openCreate}>
              Добавить трату
            </Button>
          </Group>

          <Paper withBorder radius="lg" p="md" className="panel">
            <div className="filters-grid">
              <Select
                label="Член семьи"
                placeholder="Все"
                clearable
                data={userOptions}
                value={filters.user_id || null}
                onChange={(v) => setFilters((f) => ({ ...f, user_id: v ?? '' }))}
              />
              <Select
                label="Категория"
                placeholder="Все"
                clearable
                data={categoryOptions}
                value={filters.category || null}
                onChange={(v) => setFilters((f) => ({ ...f, category: v ?? '' }))}
              />
              <Field label="С">
                <input
                  className="native-input"
                  type="date"
                  value={filters.from}
                  onChange={(e) => setFilters((f) => ({ ...f, from: e.target.value }))}
                />
              </Field>
              <Field label="По">
                <input
                  className="native-input"
                  type="date"
                  value={filters.to}
                  onChange={(e) => setFilters((f) => ({ ...f, to: e.target.value }))}
                />
              </Field>
            </div>
          </Paper>

          {error && (
            <Alert color="red" variant="light" title="Ошибка" withCloseButton onClose={() => setError(null)}>
              {error}
            </Alert>
          )}

          <Group gap="lg">
            <Text size="sm" c="dimmed">
              Всего:{' '}
              <Text span fw={600}>
                {total}
              </Text>
            </Text>
            <Text size="sm" c="dimmed">
              На странице:{' '}
              <Text span fw={700} c="teal">
                {formatMoney(pageTotal)} ₽
              </Text>
            </Text>
          </Group>

          <Paper withBorder radius="lg" className="panel" style={{ overflow: 'hidden' }}>
            {loading ? (
              <Text p="xl" c="dimmed">
                Загрузка…
              </Text>
            ) : expenses.length === 0 ? (
              <Stack p="xl" align="center" gap="md" py={48}>
                <Image
                  src="/mascot.jpg"
                  alt="Маскот"
                  w={160}
                  h={160}
                  fit="contain"
                  className="mascot-empty"
                />
                <Stack gap={4} align="center">
                  <Text fw={600} size="lg">
                    Трат пока нет
                  </Text>
                  <Text c="dimmed" size="sm" ta="center" maw={360}>
                    Добавьте первую запись или напишите боту в Telegram: «кофе 350».
                  </Text>
                </Stack>
                <Button color="teal" variant="light" onClick={openCreate}>
                  Добавить трату
                </Button>
              </Stack>
            ) : (
              <>
                <Table.ScrollContainer minWidth={720}>
                  <Table horizontalSpacing="md" verticalSpacing="sm" highlightOnHover>
                    <Table.Thead>
                      <Table.Tr>
                        <Table.Th>Дата</Table.Th>
                        <Table.Th>Участник</Table.Th>
                        <Table.Th>Категория</Table.Th>
                        <Table.Th>Сумма</Table.Th>
                        <Table.Th>Комментарий</Table.Th>
                        <Table.Th />
                      </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>{rows}</Table.Tbody>
                  </Table>
                </Table.ScrollContainer>
                {total > PAGE_SIZE && (
                  <Group justify="center" p="md">
                    <Pagination
                      total={totalPages}
                      value={page}
                      onChange={setPage}
                      color="teal"
                      radius="md"
                    />
                  </Group>
                )}
              </>
            )}
          </Paper>
        </Stack>
      </Container>

      <Modal
        opened={opened}
        onClose={close}
        title={editing ? 'Редактировать трату' : 'Новая трата'}
        centered
        radius="lg"
      >
        <Stack gap="sm">
          {!editing &&
            (users.length > 0 ? (
              <Select
                label="Участник"
                required
                data={userOptions}
                value={form.user_id || null}
                onChange={(v) => setForm((f) => ({ ...f, user_id: v ?? '' }))}
              />
            ) : (
              <Field label="Telegram ID участника">
                <input
                  className="native-input"
                  required
                  inputMode="numeric"
                  placeholder="например 123456789"
                  value={form.user_id}
                  onChange={(e) => setForm((f) => ({ ...f, user_id: e.target.value }))}
                />
              </Field>
            ))}

          <Field label="Сумма">
            <input
              className="native-input"
              required
              inputMode="decimal"
              placeholder="350"
              value={form.amount}
              onChange={(e) => setForm((f) => ({ ...f, amount: e.target.value }))}
            />
          </Field>

          <Select
            label="Категория"
            data={categoryOptions}
            value={form.category}
            onChange={(v) => setForm((f) => ({ ...f, category: v || 'food' }))}
          />

          <Field label="Дата">
            <input
              className="native-input"
              type="date"
              required
              value={form.spent_at}
              onChange={(e) => setForm((f) => ({ ...f, spent_at: e.target.value }))}
            />
          </Field>

          <Field label="Комментарий">
            <input
              className="native-input"
              value={form.comment}
              onChange={(e) => setForm((f) => ({ ...f, comment: e.target.value }))}
            />
          </Field>

          <Group justify="flex-end" mt="sm">
            <Button variant="default" onClick={close}>
              Отмена
            </Button>
            <Button color="teal" loading={saving} onClick={() => void onSubmit()}>
              Сохранить
            </Button>
          </Group>
        </Stack>
      </Modal>
    </div>
  );
}
