// Package telegram delivers a digest. It is deliberately thin: the messaging API
// called directly, one retry, and a dry-run path that prints instead of sending
// so the whole pipeline can be demonstrated without a token.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultBaseURL is the API root. It is a field rather than a constant so the
// delivery path can be tested against a local server instead of production.
const DefaultBaseURL = "https://api.telegram.org"

type Client struct {
	BaseURL string
	Token   string
	ChatID  string
	HTTP    *http.Client
}

func New(token, chatID string) Client {
	return Client{BaseURL: DefaultBaseURL, Token: token, ChatID: chatID, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

// message is typed rather than a map: the API rejects a boolean sent as the
// string "true", and a map of strings is exactly how that happens.
type message struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

// Send posts one message as HTML. It retries once: a delivery pipeline that gives
// up on the first 5xx loses the day, and the day is the point.
func (c Client) Send(ctx context.Context, text string) error {
	if c.Token == "" || c.ChatID == "" {
		return fmt.Errorf("missing token or chat id")
	}
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", base, c.Token)
	body, err := json.Marshal(message{
		ChatID: c.ChatID, Text: text, ParseMode: "HTML", DisableWebPagePreview: true,
	})
	if err != nil {
		return err
	}
	var last error
	for attempt := 1; attempt <= 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.HTTP.Do(req)
		if err != nil {
			last = err
		} else {
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			last = fmt.Errorf("telegram said %s: %s", resp.Status, string(raw))
			// A rejected entity will not succeed on retry; only retry transport trouble.
			if resp.StatusCode < 500 {
				return last
			}
		}
		if attempt == 1 {
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return last
}

// Print is the dry-run half: it writes what would be sent, and to whom, to any
// writer. Nothing here can reach the network, which is the point of a default.
func Print(w io.Writer, label, text string) {
	fmt.Fprintf(w, "--- would send %s (%d chars) ---\n%s\n", label, len([]rune(text)), text)
}
