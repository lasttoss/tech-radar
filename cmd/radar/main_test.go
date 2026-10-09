package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lasttoss/tech-radar/internal/decision"
)

const goldenPath = "../../testdata/golden/payload.txt"

// TestGoldenDigest is the whole pipeline on recorded data: if the ranking, the
// rubric, the escaping or the verdict attachment change, this diff shows it.
func TestGoldenDigest(t *testing.T) {
	items, err := loadItems([]string{
		"../../testdata/fixtures/items.json",
		"../../testdata/fixtures/items-scan-2026-09.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	log, err := decision.Load("../../decisions/verdicts.json")
	if err != nil {
		t.Fatal(err)
	}

	msgs, err := buildDigest(items, log, 6, time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(msgs, "\n\n====\n\n")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("golden digest rewritten")
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("no golden file yet - run: UPDATE_GOLDEN=1 go test ./cmd/radar (%v)", err)
	}
	if got != string(want) {
		t.Fatalf("digest changed:\n--- got ---\n%s\n--- recorded ---\n%s", got, want)
	}
}

func TestSameInputScoresTheSameWayTwice(t *testing.T) {
	items, err := loadItems([]string{"../../testdata/fixtures/items-scan-2026-09.json"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	first, err := buildDigest(items, nil, 4, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildDigest(items, nil, 4, now)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(first, "") != strings.Join(second, "") {
		t.Fatal("two runs over one input gave two digests")
	}
}

func TestDefaultScoringInstantComesFromTheData(t *testing.T) {
	// Scoring against the wall clock would make every recorded run different, so
	// the default instant is the newest timestamp in the data itself.
	items, err := loadItems([]string{"../../testdata/fixtures/items-scan-2026-09.json"})
	if err != nil {
		t.Fatal(err)
	}
	got := newest(items)
	if got.IsZero() {
		t.Fatal("newest timestamp should come from the items")
	}
	if got.Year() != 2026 {
		t.Fatalf("unexpected instant: %s", got)
	}
}
