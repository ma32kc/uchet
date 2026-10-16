package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/k1ra/uchet/internal/agent"
	"github.com/k1ra/uchet/internal/service"
)

type Bot struct {
	api   *tgbotapi.BotAPI
	agent *agent.Agent
	svc   *service.ExpenseService
}

func New(token string, ag *agent.Agent, svc *service.ExpenseService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("telegram bot: %w", err)
	}
	api.Debug = false
	return &Bot{api: api, agent: ag, svc: svc}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.api.GetUpdatesChan(u)

	slog.Info("telegram bot started", "username", b.api.Self.UserName)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil || update.Message.Text == "" {
				continue
			}
			if update.Message.Chat == nil || update.Message.From == nil {
				continue
			}
			// Only private chats for expense NL UX.
			if !update.Message.Chat.IsPrivate() {
				continue
			}
			go b.handleMessage(ctx, update.Message)
		}
	}
}

func (b *Bot) handleMessage(parent context.Context, msg *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

	userID := msg.From.ID
	name := strings.TrimSpace(strings.TrimSpace(msg.From.FirstName + " " + msg.From.LastName))
	if name == "" && msg.From.UserName != "" {
		name = msg.From.UserName
	}
	if _, err := b.svc.UpsertUser(ctx, userID, name); err != nil {
		slog.Error("upsert user", "err", err, "user_id", userID)
	}

	reply, err := b.agent.Handle(ctx, msg.Chat.ID, userID, msg.Text)
	if err != nil {
		slog.Error("agent", "err", err, "user_id", userID)
		reply = "Не удалось обработать сообщение. Попробуй ещё раз чуть позже."
	}
	if reply == "" {
		return
	}

	out := tgbotapi.NewMessage(msg.Chat.ID, reply)
	out.ReplyToMessageID = msg.MessageID
	if _, err := b.api.Send(out); err != nil {
		slog.Error("send message", "err", err)
	}
}
