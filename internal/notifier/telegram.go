package notifier

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	telegramAPI  = "https://api.telegram.org/bot%s/sendMessage"
	notifTimeout = 30 * time.Second
	maxRespBody  = 4096
)

// SendNotification mengirim pesan Telegram via Bot API.
// botToken: token dari @BotFather
// chatID: ID chat tujuan (dari @userinfobot)
// message: teks yang akan dikirim (markdown supported)
func SendNotification(ctx context.Context, botToken, chatID, message string) error {
	if botToken == "" || chatID == "" || message == "" {
		return fmt.Errorf("notifier: botToken, chatID, dan message wajib diisi")
	}

	apiURL := fmt.Sprintf(telegramAPI, botToken)
	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)
	data.Set("parse_mode", "Markdown")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return fmt.Errorf("notifier: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: notifTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("notifier: HTTP call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxRespBody))
		return fmt.Errorf("notifier: Telegram API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// FormatNewTaskMessage membuat pesan notifikasi untuk tugas baru (Markdown).
func FormatNewTaskMessage(courseName, title, dueDateStr, taskURL string) string {
	return fmt.Sprintf("*🔔 TUGAS BARU!*\n📚 %s\n📝 %s\n⏰ Deadline: %s\n🔗 %s",
		courseName, title, dueDateStr, taskURL)
}

// FormatDeadlineChangedMessage membuat pesan notifikasi untuk deadline yang berubah (Markdown).
func FormatDeadlineChangedMessage(title, taskURL string) string {
	return fmt.Sprintf("*⚠️ DEADLINE BERUBAH!*\n📝 %s\n🔗 %s",
		title, taskURL)
}

