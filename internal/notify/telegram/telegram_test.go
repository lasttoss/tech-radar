package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSendPostsTypedHTMLAndRetriesOnce(t *testing.T) {
	var calls int32
	var got message
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/bottok/sendMessage") {
			http.NotFound(w, r)
			return
		}
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusBadGateway) // transport trouble: worth one retry
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("body was not JSON: %v", err)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := Client{BaseURL: srv.URL, Token: "tok", ChatID: "42", HTTP: srv.Client()}
	if err := c.Send(context.Background(), "<b>hi</b>"); err != nil {
		t.Fatalf("a 502 should have been retried and then succeeded: %v", err)
	}
	if calls != 2 {
		t.Fatalf("want 2 attempts, got %d", calls)
	}
	if got.ChatID != "42" || got.ParseMode != "HTML" || got.Text != "<b>hi</b>" {
		t.Fatalf("wrong payload: %+v", got)
	}
	if !got.DisableWebPagePreview {
		t.Fatal("previews must stay off or a digest turns into a wall of cards")
	}
}

func TestARejectedEntityIsNotRetried(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"description":"Bad Request: can't parse entities"}`))
	}))
	defer srv.Close()

	c := Client{BaseURL: srv.URL, Token: "tok", ChatID: "42", HTTP: srv.Client()}
	err := c.Send(context.Background(), "bad <markup>")
	if err == nil {
		t.Fatal("a 400 must be reported")
	}
	if calls != 1 {
		t.Fatalf("a malformed entity will fail identically, so it must not be retried: %d attempts", calls)
	}
	if !strings.Contains(err.Error(), "parse entities") {
		t.Fatalf("the upstream reason should survive into the error: %v", err)
	}
}

func TestMissingCredentialsFailBeforeTheNetwork(t *testing.T) {
	c := Client{BaseURL: "http://127.0.0.1:1", HTTP: http.DefaultClient}
	if err := c.Send(context.Background(), "x"); err == nil {
		t.Fatal("no token should fail, and fail before trying to connect")
	}
}

func TestPrintIsTheDryRunHalf(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, "digest 1/2", "<b>x</b>")
	out := buf.String()

	if !strings.Contains(out, "would send digest 1/2") {
		t.Fatalf("dry run must say what it would have sent: %s", out)
	}
	if !strings.Contains(out, "<b>x</b>") {
		t.Fatal("dry run must print the payload itself, so it can be diffed")
	}
}
