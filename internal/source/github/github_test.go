package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func serveFixture(t *testing.T, path, wantPath string) *httptest.Server {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("recorded fixture %s missing: %v", path, err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestNewlyCreatedReadsTheSearchAPIShape(t *testing.T) {
	srv := serveFixture(t, "../../../testdata/fixtures/gh-search.json", "/search/repositories")
	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}

	items, err := c.NewlyCreated(context.Background(), time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("recorded search fixture produced nothing")
	}
	for _, it := range items {
		if !strings.Contains(it.ID, "/") {
			t.Fatalf("a repository needs owner/name, got %q", it.ID)
		}
		if !strings.HasPrefix(it.URL, "https://github.com/") {
			t.Fatalf("repository link should point at github.com: %s", it.URL)
		}
		if len(it.Meta) == 0 {
			t.Fatal("an item with no meta line tells a reader nothing")
		}
	}
}

func TestReleasesCarryTheirTagAndDate(t *testing.T) {
	srv := serveFixture(t, "../../../testdata/fixtures/gh-releases.json", "/repos/vllm-project/vllm/releases")
	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}

	items, err := c.Releases(context.Background(), "vllm-project/vllm", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("recorded releases fixture produced nothing")
	}
	for _, it := range items {
		if !strings.Contains(it.ID, "@") {
			t.Fatalf("a release id must carry the tag, got %q", it.ID)
		}
		if it.Created.IsZero() {
			t.Fatalf("release %s has no date, so recency would score it as ancient", it.ID)
		}
	}
}

func TestRateLimitAndOtherErrorsSurface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded for 203.0.113.7."}`))
	}))
	defer srv.Close()
	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}

	_, err := c.NewlyCreated(context.Background(), time.Now().AddDate(0, 0, -30), 5)
	if err == nil {
		t.Fatal("rate limiting must be reported, not turned into an empty day")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("the upstream reason should be in the error, got: %v", err)
	}
}
