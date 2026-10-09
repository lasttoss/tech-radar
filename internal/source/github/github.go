// Package github reads the two GitHub signals that matter for a radar: what was
// created recently and got attention, and what shipped a release.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lasttoss/tech-radar/internal/rank"
)

const DefaultBaseURL = "https://api.github.com"

type Client struct {
	BaseURL string
	Token   string // optional: unauthenticated search is rate limited per IP
	HTTP    *http.Client
}

func New(token string) Client {
	return Client{BaseURL: DefaultBaseURL, Token: token, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

type repo struct {
	FullName    string    `json:"full_name"`
	HTMLURL     string    `json:"html_url"`
	Description string    `json:"description"`
	Stars       int       `json:"stargazers_count"`
	CreatedAt   time.Time `json:"created_at"`
	PushedAt    time.Time `json:"pushed_at"`
	Topics      []string  `json:"topics"`
	Language    string    `json:"language"`
	License     *struct {
		SPDXID string `json:"spdx_id"`
	} `json:"license"`
}

// NewlyCreated finds repositories created since a date, most starred first. This
// is the signal a trending page gives, but as data with dates attached.
func (c Client) NewlyCreated(ctx context.Context, since time.Time, limit int) ([]rank.Item, error) {
	url := fmt.Sprintf("%s/search/repositories?q=created:%%3E%s&sort=stars&order=desc&per_page=%d",
		c.BaseURL, since.Format("2006-01-02"), limit)
	var resp struct {
		Items []repo `json:"items"`
	}
	if err := c.get(ctx, url, &resp); err != nil {
		return nil, err
	}
	out := make([]rank.Item, 0, len(resp.Items))
	for _, r := range resp.Items {
		lic := ""
		if r.License != nil {
			lic = r.License.SPDXID
		}
		out = append(out, rank.Item{
			Source: "github", ID: r.FullName, Title: r.FullName,
			Meta:      compact(r.Language, lic, stars(r.Stars), "created "+r.CreatedAt.Format("2006-01-02")),
			Desc:      strings.TrimSpace(r.Description),
			URL:       r.HTMLURL,
			Stars:     r.Stars,
			Topics:    r.Topics,
			Created:   r.CreatedAt,
			LastMoved: r.PushedAt,
		})
	}
	return out, nil
}

type release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
}

// Releases returns recent releases of a repository the radar has decided to watch.
func (c Client) Releases(ctx context.Context, ownerRepo string, limit int) ([]rank.Item, error) {
	url := fmt.Sprintf("%s/repos/%s/releases?per_page=%d", c.BaseURL, ownerRepo, limit)
	var releases []release
	if err := c.get(ctx, url, &releases); err != nil {
		return nil, err
	}
	out := make([]rank.Item, 0, len(releases))
	for _, r := range releases {
		title := r.Name
		if title == "" {
			title = r.TagName
		}
		desc := strings.TrimSpace(strings.SplitN(r.Body, "\n", 2)[0])
		if len(desc) > 200 {
			desc = strings.TrimSpace(desc[:197]) + "..."
		}
		out = append(out, rank.Item{
			Source: "github-release", ID: ownerRepo + "@" + r.TagName, Title: title,
			Meta:      compact(ownerRepo, "released "+r.PublishedAt.Format("2006-01-02")),
			Desc:      desc,
			URL:       r.HTMLURL,
			Created:   r.PublishedAt,
			LastMoved: r.PublishedAt,
		})
	}
	return out, nil
}

func (c Client) get(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var body struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return fmt.Errorf("github %s: %s (%s)", url, resp.Status, body.Message)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("github %s: %w", url, err)
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

func stars(n int) string {
	if n == 0 {
		return ""
	}
	switch {
	case n >= 1000:
		return fmt.Sprintf("⭐%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("⭐%d", n)
	}
}
