package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/smokeguap/pingwarden/internal/alert"
)

type Sender interface {
	Name() string
	Send(ctx context.Context, e alert.Event) error
}

func postJSON(ctx context.Context, c *http.Client, endpoint string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return errors.New("invalid request URL")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

type Telegram struct {
	token, chatID, baseURL string
	client                 *http.Client
}

func NewTelegram(token, chatID string) *Telegram {
	return &Telegram{
		token: token, chatID: chatID,
		baseURL: "https://api.telegram.org",
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *Telegram) Name() string { return "telegram" }

func (t *Telegram) Send(ctx context.Context, e alert.Event) error {
	endpoint := t.baseURL + "/bot" + t.token + "/sendMessage"
	return postJSON(ctx, t.client, endpoint, map[string]string{
		"chat_id": t.chatID,
		"text":    e.Text(),
	})
}

type Webhook struct {
	url    string
	client *http.Client
}

func NewWebhook(url string) *Webhook {
	return &Webhook{url: url, client: &http.Client{Timeout: 10 * time.Second}}
}

func (w *Webhook) Name() string { return "webhook" }

func (w *Webhook) Send(ctx context.Context, e alert.Event) error {
	return postJSON(ctx, w.client, w.url, e)
}
