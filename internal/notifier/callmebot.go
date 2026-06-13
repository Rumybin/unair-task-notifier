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
	callmebotURL   = "https://api.callmebot.com/whatsapp.php"
	notifTimeout   = 30 * time.Second
	maxRespBody    = 4096
)

// SendNotification mengirim pesan WhatsApp via Callmebot API.
// phone: nomor WA dengan kode negara (contoh: 628xxxxxxxxx)
// apikey: API key dari callmebot.com
// message: teks yang akan dikirim (akan di-URL-encode otomatis)
func SendNotification(ctx context.Context, phone, apikey, message string) error {
	if phone == "" || apikey == "" || message == "" {
		return fmt.Errorf("notifier: phone, apikey, dan message wajib diisi")
	}

	reqURL := fmt.Sprintf("%s?phone=%s&text=%s&apikey=%s",
		callmebotURL,
		url.QueryEscape(phone),
		url.QueryEscape(message),
		url.QueryEscape(apikey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("notifier: create request: %w", err)
	}

	client := &http.Client{Timeout: notifTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("notifier: HTTP call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxRespBody))
		return fmt.Errorf("notifier: API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// FormatNewTaskMessage membuat pesan notifikasi untuk tugas baru.
func FormatNewTaskMessage(courseName, title, dueDateStr, taskURL string) string {
	return fmt.Sprintf("🔔 TUGAS BARU!\n📚 %s\n📝 %s\n⏰ Deadline: %s\n🔗 %s",
		courseName, title, dueDateStr, taskURL)
}

// FormatDeadlineChangedMessage membuat pesan notifikasi untuk deadline yang berubah.
func FormatDeadlineChangedMessage(title, taskURL string) string {
	return fmt.Sprintf("⚠️ DEADLINE BERUBAH!\n📝 %s\n🔗 %s",
		title, taskURL)
}

