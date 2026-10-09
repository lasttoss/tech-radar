// Command radar ingests, scores, builds and delivers a digest - and closes the
// loop, because every item that already has a verdict carries it into the
// message. Dry run is the default: printing costs nothing, sending costs trust.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lasttoss/tech-radar/internal/decision"
	"github.com/lasttoss/tech-radar/internal/digest"
	"github.com/lasttoss/tech-radar/internal/notify/telegram"
	"github.com/lasttoss/tech-radar/internal/rank"
	"github.com/lasttoss/tech-radar/internal/source/github"
	"github.com/lasttoss/tech-radar/internal/source/huggingface"
)

func main() {
	var (
		itemsFlag   = flag.String("items", "testdata/fixtures/items.json,testdata/fixtures/items-scan-2026-09.json", "recorded item files, comma separated")
		recordTo    = flag.String("record", "", "fetch live sources, write them to this file and exit")
		send        = flag.Bool("send", false, "deliver to the messaging API (default: print what would be sent)")
		verdictPath = flag.String("verdicts", "decisions/verdicts.json", "verdict log")
		limit       = flag.Int("limit", 8, "items per digest")
		nowFlag     = flag.String("now", "", "RFC3339 instant to score against (default: newest item, so output is stable)")
		since       = flag.String("since-days", "30", "how far back -record looks for new repositories")
	)
	flag.Parse()

	ctx := context.Background()

	if *recordTo != "" {
		days, err := time.ParseDuration(*since + "h")
		must(err)
		items, err := record(ctx, time.Now().Add(-days*24/24))
		must(err)
		must(os.MkdirAll(filepath.Dir(*recordTo), 0o755))
		must(writeJSON(*recordTo, items))
		fmt.Fprintf(os.Stderr, "recorded %d items to %s\n", len(items), *recordTo)
		return
	}

	items, err := loadItems(strings.Split(*itemsFlag, ","))
	must(err)
	if len(items) == 0 {
		must(fmt.Errorf("no items in %s", *itemsFlag))
	}

	now := newest(items)
	if *nowFlag != "" {
		now, err = time.Parse(time.RFC3339, *nowFlag)
		must(err)
	}

	log := map[string]decision.Verdict{}
	if _, err := os.Stat(*verdictPath); err == nil {
		log, err = decision.Load(*verdictPath)
		must(err)
	}

	msgs, err := buildDigest(items, log, *limit, now)
	must(err)

	if *send {
		client := telegram.New(os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"))
		for _, m := range msgs {
			must(client.Send(ctx, m))
		}
		fmt.Fprintf(os.Stderr, "sent %d message(s)\n", len(msgs))
		return
	}
	for i, m := range msgs {
		telegram.Print(os.Stdout, fmt.Sprintf("digest %d/%d", i+1, len(msgs)), m)
	}
}

// record pulls every live source once and normalises them into the shape the rest
// of the pipeline uses, which is what makes the recorded run reproducible.
func record(ctx context.Context, since time.Time) ([]rank.Item, error) {
	hf := huggingface.New()
	gh := github.New(os.Getenv("GITHUB_TOKEN"))

	var out []rank.Item
	steps := []struct {
		name string
		run  func() ([]rank.Item, error)
	}{
		{"huggingface trending", func() ([]rank.Item, error) { return hf.Trending(ctx, 12) }},
		{"huggingface papers", func() ([]rank.Item, error) { return hf.Papers(ctx, 5) }},
		{"github new repositories", func() ([]rank.Item, error) { return gh.NewlyCreated(ctx, since, 12) }},
		{"github releases", func() ([]rank.Item, error) { return gh.Releases(ctx, "vllm-project/vllm", 3) }},
	}
	for _, s := range steps {
		got, err := s.run()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.name, err)
		}
		fmt.Fprintf(os.Stderr, "  %-26s %d items\n", s.name, len(got))
		out = append(out, got...)
	}
	return out, nil
}

// buildDigest is the pipeline minus I/O: rank, attach verdicts, render, validate.
// The binary and the golden test call this same function, so the test proves the
// code that runs rather than a copy of it.
func buildDigest(items []rank.Item, log map[string]decision.Verdict, limit int, now time.Time) ([]string, error) {
	scored := rank.Rank(items, rank.DefaultWeights(), rank.DefaultStack(), now)
	if limit > len(scored) {
		limit = len(scored)
	}
	digestItems := make([]digest.Item, 0, limit)
	for _, s := range scored[:limit] {
		digestItems = append(digestItems, toDigest(s, log))
	}
	header := fmt.Sprintf("📡 <b>tech-radar</b> — %s UTC\n%d items scored from %d signals · rubric impact/novelty/relevance",
		now.UTC().Format("2006-01-02"), len(digestItems), len(items))
	msgs := digest.Build(header, digestItems)
	for _, m := range msgs {
		if err := digest.Validate(m); err != nil {
			return nil, err
		}
	}
	return msgs, nil
}

func toDigest(s rank.Scored, log map[string]decision.Verdict) digest.Item {
	it := digest.Item{
		ID: s.ID, Title: s.Title, Desc: s.Desc, URL: s.URL, Home: s.Home, Meta: s.Meta,
	}
	if len(s.Matched) > 0 {
		it.Reason = "matches " + strings.Join(s.Matched, ", ") + fmt.Sprintf(" (score %.2f)", s.Score)
	} else {
		it.Reason = fmt.Sprintf("no stack match, scored on popularity and recency (%.2f)", s.Score)
	}
	if v, ok := decision.Lookup(log, s.ID); ok {
		it.Applied = &digest.Applied{Label: v.Verdict, Evidence: v.Evidence, Repo: v.Repo, Commit: v.Commit}
	}
	return it
}

func loadItems(paths []string) ([]rank.Item, error) {
	var out []rank.Item
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var items []rank.Item
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		out = append(out, items...)
	}
	return out, nil
}

func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// newest is the default scoring instant: the data's own latest timestamp. Using
// the wall clock would make every recorded run score differently.
func newest(items []rank.Item) time.Time {
	var max time.Time
	for _, it := range items {
		for _, t := range []time.Time{it.Created, it.LastMoved} {
			if t.After(max) {
				max = t
			}
		}
	}
	if max.IsZero() {
		return time.Unix(0, 0).UTC()
	}
	return max
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "radar:", err)
		os.Exit(1)
	}
}
