package huggingface

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// serveFixture replays a recorded API response. The fixtures are the real bodies,
// so these tests fail if the upstream shape changes under us.
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

func TestTrendingMapsRecordedModels(t *testing.T) {
	srv := serveFixture(t, "../../../testdata/fixtures/hf-models.json", "/models")
	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}

	items, err := c.Trending(context.Background(), 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 5 {
		t.Fatalf("recorded fixture should yield several models, got %d", len(items))
	}
	var withLikes int
	for _, it := range items {
		if it.ID == "" || it.Title == "" {
			t.Fatalf("item without identity: %+v", it)
		}
		if !strings.HasPrefix(it.URL, "https://huggingface.co/") {
			t.Fatalf("model link should point at the Hub: %s", it.URL)
		}
		if it.Source != "huggingface" {
			t.Fatalf("wrong source tag: %s", it.Source)
		}
		if it.Likes > 0 {
			withLikes++
		}
	}
	if withLikes == 0 {
		t.Fatal("no model carried likes, so the popularity term would be blind")
	}
}

func TestPapersUseTheNestedPaperAndKeepTheSummaryToOneLine(t *testing.T) {
	srv := serveFixture(t, "../../../testdata/fixtures/hf-papers.json", "/daily_papers")
	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}

	items, err := c.Papers(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("recorded papers fixture produced nothing")
	}
	for _, it := range items {
		if it.Title == "" || it.ID == "" {
			t.Fatalf("paper without title or id: %+v", it)
		}
		if strings.Contains(it.Desc, "\n") {
			t.Fatalf("summary must be one line, got %q", it.Desc)
		}
		if len([]rune(it.Desc)) > 223 {
			t.Fatalf("summary is %d runes, longer than the digest line allows", len([]rune(it.Desc)))
		}
	}
}

func TestAnHTTPErrorIsNotSilentlyAnEmptyDay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()
	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}

	if _, err := c.Trending(context.Background(), 5); err == nil {
		t.Fatal("a 403 must surface: an empty digest that looks like a quiet day is how a dead pipeline goes unnoticed")
	}
}
