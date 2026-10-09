package rank

import (
	"testing"
	"time"
)

var now = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

func TestImpactIsLogScaled(t *testing.T) {
	small := Item{Stars: 1_000}
	large := Item{Stars: 10_000}
	huge := Item{Stars: 100_000}

	if Impact(small) <= 0 || Impact(large) <= Impact(small) || Impact(huge) <= Impact(large) {
		t.Fatalf("expected monotonic impact, got %v %v %v", Impact(small), Impact(large), Impact(huge))
	}
	if Impact(huge) > 1 {
		t.Fatalf("impact must stay in [0,1], got %v", Impact(huge))
	}
	if Impact(Item{}) != 0 {
		t.Fatal("no popularity should score zero impact")
	}
}

func TestImpactFallsBackToLikes(t *testing.T) {
	// Hub models have likes, not stars: a source that only fills one must still rank.
	if Impact(Item{Likes: 5_000}) == 0 {
		t.Fatal("likes should count as popularity when stars are absent")
	}
}

func TestNoveltyDecaysAndNeverGoesNegative(t *testing.T) {
	today := Novelty(Item{Created: now}, now)
	month := Novelty(Item{Created: now.AddDate(0, 0, -30)}, now)
	if !(today > month) {
		t.Fatalf("a newer item must score higher: %v vs %v", today, month)
	}
	if got := Novelty(Item{Created: now.Add(time.Hour)}, now); got > 1 {
		t.Fatalf("future timestamps must not exceed 1, got %v", got)
	}
	if Novelty(Item{}, now) != 0 {
		t.Fatal("an item with no date has no novelty")
	}
}

func TestRelevanceReportsWhichWordsMatched(t *testing.T) {
	s := Stack{Keywords: []string{"game", "agent", "vector-database"}}
	rel, matched := s.Relevance(Item{ID: "x/y", Desc: "an agent that plays a game"})

	if len(matched) != 2 {
		t.Fatalf("want 2 matched words, got %v", matched)
	}
	if want := 2.0 / 3.0; rel < want-1e-9 || rel > want+1e-9 {
		t.Fatalf("want %v, got %v", want, rel)
	}
}

func TestRankIsDeterministicOnTies(t *testing.T) {
	// Two identical scores must come out in the same order every run: a demo that
	// reorders itself between runs cannot be diffed.
	items := []Item{
		{ID: "b/repo", Stars: 100, Created: now},
		{ID: "a/repo", Stars: 100, Created: now},
	}
	first := Rank(items, DefaultWeights(), DefaultStack(), now)
	second := Rank([]Item{items[1], items[0]}, DefaultWeights(), DefaultStack(), now)

	if first[0].ID != second[0].ID || first[1].ID != second[1].ID {
		t.Fatalf("order changed with input order: %v then %v", first[0].ID, second[0].ID)
	}
	if first[0].ID != "a/repo" {
		t.Fatalf("ties should break on id, got %s", first[0].ID)
	}
}

func TestRankOrdersByRubricNotByPopularityAlone(t *testing.T) {
	popular := Item{ID: "big/popular", Stars: 100_000, Created: now.AddDate(0, 0, -365)}
	relevant := Item{ID: "small/relevant", Stars: 500, Created: now, Desc: "game server inference for an agent"}
	desc := Item{ID: "other/thing", Stars: 400, Created: now, Desc: "a font library"}

	got := Rank([]Item{popular, relevant, desc}, DefaultWeights(), DefaultStack(), now)
	if got[0].ID != "small/relevant" {
		t.Fatalf("rubric should beat raw popularity here, got order %v", []string{got[0].ID, got[1].ID, got[2].ID})
	}
	if len(got[0].Matched) == 0 {
		t.Fatal("the winner must say why it won")
	}
}

func TestWeightsAreNormalised(t *testing.T) {
	items := []Item{{ID: "x", Stars: 1_000, Created: now, Desc: "game"}}
	scaled := Rank(items, Weights{Impact: 9, Novelty: 9, Relevance: 9}, DefaultStack(), now)
	if scaled[0].Score > 1 {
		t.Fatalf("weights that do not sum to one must still produce a score in [0,1], got %v", scaled[0].Score)
	}
}

func TestRelevanceMatchesWholeWordsOnly(t *testing.T) {
	// Substring matching put "go" in google and "test" in latest, and the digest
	// prints the matched words as the reason an item is in the message at all.
	s := Stack{Keywords: []string{"go", "test", "game"}}

	if _, matched := s.Relevance(Item{ID: "google/embeddinggemma-2"}); len(matched) != 0 {
		t.Fatalf(`"go" must not match "google", got %v`, matched)
	}
	if _, matched := s.Relevance(Item{Desc: "the latest release"}); len(matched) != 0 {
		t.Fatalf(`"test" must not match "latest", got %v`, matched)
	}
	// all three keywords, each present as a whole word
	if _, matched := s.Relevance(Item{Desc: "a game server in Go, and a test for it"}); len(matched) != 3 {
		t.Fatalf("want all three words, got %v", matched)
	}
	// Deliberately no stemming: "tested" does not match "test". A rubric whose
	// matching cannot be predicted is one you cannot defend when it ranks
	// something stupid, and the digest prints the matched words as its reason.
	if _, matched := s.Relevance(Item{Desc: "a tested thing"}); len(matched) != 0 {
		t.Fatalf(`"test" must not match "tested", got %v`, matched)
	}
	hyphen := Stack{Keywords: []string{"vector-database"}}
	if _, matched := hyphen.Relevance(Item{Desc: "uses a vector-database for recall"}); len(matched) != 1 {
		t.Fatalf("hyphenated keywords must still match, got %v", matched)
	}
}
