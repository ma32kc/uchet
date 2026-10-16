package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/k1ra/uchet/internal/domain"
	"github.com/k1ra/uchet/internal/service"
)

type Agent struct {
	llm      *LLMClient
	svc      *service.ExpenseService
	memory   *Memory
	loc      *time.Location
	maxIters int
}

func New(llm *LLMClient, svc *service.ExpenseService, loc *time.Location, maxIters int) *Agent {
	if maxIters <= 0 {
		maxIters = 5
	}
	return &Agent{
		llm:      llm,
		svc:      svc,
		memory:   NewMemory(24, 2*time.Hour),
		loc:      loc,
		maxIters: maxIters,
	}
}

func (a *Agent) Handle(ctx context.Context, chatID, userID int64, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "Напиши трату обычным текстом, например: «кофе 350».", nil
	}
	if text == "/start" || strings.HasPrefix(text, "/start ") {
		return "Привет! Я помогу вести учёт расходов. Пиши как обычно: «кофе 350», «такси 1200 вчера», «сколько потратил на еду в этом месяце».", nil
	}

	system := a.systemPrompt()
	history := a.memory.Get(chatID)
	messages := make([]chatMessage, 0, len(history)+4)
	messages = append(messages, chatMessage{Role: "system", Content: system})
	messages = append(messages, history...)
	userMsg := chatMessage{Role: "user", Content: text}
	messages = append(messages, userMsg)

	exec := &toolExecutor{svc: a.svc, userID: userID, loc: a.loc}
	tools := toolSpecs()

	var final string
	for i := 0; i < a.maxIters; i++ {
		msg, err := a.llm.Chat(ctx, messages, tools)
		if err != nil {
			return "", err
		}
		if len(msg.ToolCalls) == 0 {
			final = strings.TrimSpace(msg.Content)
			if final == "" {
				final = "Готово."
			}
			a.memory.Append(chatID, userMsg, chatMessage{Role: "assistant", Content: final})
			return final, nil
		}

		messages = append(messages, msg)
		toolResults := make([]string, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			slog.Info("tool call", "name", tc.Function.Name, "user_id", userID)
			result, err := exec.execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = "Не получилось: " + err.Error()
			}
			toolResults = append(toolResults, result)
			messages = append(messages, chatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
		}
		// Reply with Go-formatted tool output only — free LLMs often break amounts/dates when rewriting.
		final = strings.TrimSpace(strings.Join(toolResults, "\n"))
		a.memory.Append(chatID, userMsg, chatMessage{Role: "assistant", Content: final})
		return final, nil
	}
	return "", fmt.Errorf("agent exceeded tool iterations")
}

func (a *Agent) systemPrompt() string {
	now := time.Now().In(a.loc)
	weekStart := startOfWeek(now)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, a.loc)

	return fmt.Sprintf(`Ты помощник учёта семейных расходов в Telegram.
Отвечай коротко по-русски. Не предлагай кнопки и команды.
Работай только с тратами текущего пользователя (его Telegram ID уже привязан к tools).
Категории: %s.
Сегодня: %s (timezone %s).
Начало этой недели (пн): %s.
Начало этого месяца: %s.
Если дата не указана — используй сегодня.
Для «вчера» считай дату сам и передавай spent_at в YYYY-MM-DD.
Для относительных периодов («неделя», «месяц») передавай from/to в tools.
При добавлении старайся заполнить comment (например «кофе», «такси»).
Никогда не выдумывай суммы, даты и траты. Используй только факты из ответов tools.
Не пиши id трат пользователю. Не извиняйся без причины.
`, domain.CategoryLabelsList(), now.Format("2006-01-02"), a.loc.String(), weekStart.Format("2006-01-02"), monthStart.Format("2006-01-02"))
}

func startOfWeek(t time.Time) time.Time {
	// Monday as first day
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.AddDate(0, 0, -(weekday - 1))
}
