package backup

import (
	"context"
	"fmt"
	"os"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type TelegramSender struct {
	bot *telego.Bot
}

func NewTelegramSender(bot *telego.Bot) *TelegramSender {
	return &TelegramSender{bot: bot}
}

func (s *TelegramSender) SendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := s.bot.SendMessage(ctx, tu.Message(telego.ChatID{ID: chatID}, text))
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	return nil
}

func (s *TelegramSender) SendDocument(ctx context.Context, chatID int64, path, filename, caption string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open telegram document: %w", err)
	}
	defer file.Close()

	params := tu.Document(telego.ChatID{ID: chatID}, tu.FileFromReader(file, filename)).
		WithCaption(caption).
		WithDisableContentTypeDetection()

	_, err = s.bot.SendDocument(ctx, params)
	if err != nil {
		return fmt.Errorf("send telegram document %s: %w", filename, err)
	}
	return nil
}
