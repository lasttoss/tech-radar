// Package huggingface reads the public Hub API. Models and daily papers are the
// two signals worth having: what is being used, and what is being claimed.
package huggingface

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lasttoss/tech-radar/internal/rank"
)

const DefaultBaseURL = "https://huggingface.co/api"

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New() Client {
	return Client{BaseURL: DefaultBaseURL, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

type model struct {
	ID           string    `json:"id"`
	Likes        int       `json:"likes"`
	Downloads    int       `json:"downloads"`
	PipelineTag  string    `json:"pipeline_tag"`
	LibraryName  string    `json:"library_name"`
	Tags         []string  `json:"tags"`
	LastModified time.Time `json:"lastModified"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Trending returns the most used models right now.
func (c Client) Trending(ctx context.Context, limit int) ([]rank.Item, error) {
	url := fmt.Sprintf("%s/models?sort=trendingScore&direction=-1&limit=%d", c.BaseURL, limit)
	var models []model
	if err := c.get(ctx, url, &models); err != nil {
		return nil, err
	}
	out := make([]rank.Item, 0, len(models))
	for _, m := range models {
		out = append(out, rank.Item{
			Source: "huggingface", ID: m.ID, Title: m.ID,
			Meta:  compact(m.PipelineTag, license(m.Tags), likes(m.Likes), downloads(m.Downloads)),
			Desc:  strings.Join(firstN(m.Tags, 4), ", "),
			URL:   "https://huggingface.co/" + m.ID,
			Likes: m.Likes, Topics: m.Tags,
			Created: m.CreatedAt, LastMoved: m.LastModified,
		})
	}
	return out, nil
}

type dailyPaper struct {
	Title       string `json:"title"`
	PublishedAt string `json:"publishedAt"`
	Upvotes     int    `json:"upvotes"`
	Paper       struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Summary     string `json:"summary"`
		Upvotes     int    `json:"upvotes"`
		PublishedAt string `json:"publishedAt"`
	} `json:"paper"`
}

func (p dailyPaper) name() string {
	if p.Paper.Title != "" {
		return p.Paper.Title
	}
	return p.Title
}

// Papers returns the papers the Hub is featuring today. The summary is trimmed
// to one paragraph: a digest line is a pointer, not a reading list.
func (c Client) Papers(ctx context.Context, limit int) ([]rank.Item, error) {
	url := fmt.Sprintf("%s/daily_papers?limit=%d", c.BaseURL, limit)
	var papers []dailyPaper
	if err := c.get(ctx, url, &papers); err != nil {
		return nil, err
	}
	out := make([]rank.Item, 0, len(papers))
	for _, p := range papers {
		when := firstNonEmpty(p.Paper.PublishedAt, p.PublishedAt)
		up := p.Paper.Upvotes
		if up == 0 {
			up = p.Upvotes
		}
		desc := strings.TrimSpace(p.Paper.Summary)
		if i := strings.Index(desc, "\n"); i > 0 {
			desc = desc[:i]
		}
		if len(desc) > 220 {
			desc = strings.TrimSpace(desc[:217]) + "..."
		}
		out = append(out, rank.Item{
			Source: "huggingface-paper", ID: p.Paper.ID, Title: p.name(),
			Meta:    compact("paper", upvotes(up), when),
			Desc:    desc,
			URL:     "https://huggingface.co/papers/" + p.Paper.ID,
			Created: parseDay(when),
			Likes:   up,
		})
	}
	return out, nil
}

func (c Client) get(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("huggingface %s: %s", url, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("huggingface %s: %w", url, err)
	}
	return nil
}

func compact(parts ...string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

func likes(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("♥%s", human(n))
}

func upvotes(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d upvotes", n)
}

func downloads(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%s downloads", human(n))
}

func license(tags []string) string {
	for _, t := range tags {
		if strings.HasPrefix(t, "license:") {
			return strings.TrimPrefix(t, "license:")
		}
	}
	return ""
}

func firstN(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	return in[:n]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func parseDay(s string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func human(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
