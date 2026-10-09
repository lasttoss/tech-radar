// Package rank scores radar items with a rubric that is written down and tested,
// so "this looks interesting" is a number with reasons instead of a feeling.
package rank

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Item is one signal, normalised by a source adapter. Every source produces this
// shape, so ranking and digest never learn where an item came from.
type Item struct {
	Source    string    `json:"source"`
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Meta      []string  `json:"meta,omitempty"`
	Desc      string    `json:"desc,omitempty"`
	URL       string    `json:"url"`
	Home      string    `json:"home,omitempty"`
	Stars     int       `json:"stars,omitempty"`
	Likes     int       `json:"likes,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	LastMoved time.Time `json:"lastMoved,omitempty"`
	Topics    []string  `json:"topics,omitempty"`
}

// Weights is the rubric. Rank normalises them, so they need not sum to one.
type Weights struct{ Impact, Novelty, Relevance float64 }

func DefaultWeights() Weights { return Weights{Impact: 0.45, Novelty: 0.25, Relevance: 0.30} }

// Stack is what the radar is pointed at: words from the systems a finding would
// have to land in. Relevance measures against these, which is why the digest can
// print a reason instead of an opinion.
type Stack struct{ Keywords []string }

func DefaultStack() Stack {
	return Stack{Keywords: []string{
		"agent", "game", "server", "backend", "go", "inference", "vision",
		"embedding", "quant", "gguf", "mcp", "eval", "test", "observability",
	}}
}

// Scored is an item with the numbers that put it where it is.
type Scored struct {
	Item
	Score                      float64
	Impact, Novelty, Relevance float64
	Matched                    []string
}

// Impact is log-scaled: 1k against 10k stars matters, 90k against 99k does not.
func Impact(it Item) float64 {
	n := it.Stars
	if n == 0 {
		n = it.Likes
	}
	if n <= 0 {
		return 0
	}
	v := math.Log10(float64(n)+1) / 5 // 100k is 1.0
	if v > 1 {
		v = 1
	}
	return v
}

// Novelty decays with age: something created today is news, last year is not.
func Novelty(it Item, now time.Time) float64 {
	t := it.Created
	if t.IsZero() {
		t = it.LastMoved
	}
	if t.IsZero() {
		return 0
	}
	days := now.Sub(t).Hours() / 24
	if days < 0 {
		days = 0
	}
	return 1 / (1 + days/30)
}

// Relevance returns how much of the stack an item touches and which words matched.
func (s Stack) Relevance(it Item) (float64, []string) {
	if len(s.Keywords) == 0 {
		return 0, nil
	}
	hay := strings.ToLower(strings.Join([]string{
		it.ID, it.Title, it.Desc, strings.Join(it.Topics, " "), strings.Join(it.Meta, " "),
	}, " "))
	var matched []string
	for _, kw := range s.Keywords {
		// Whole words only. Substring matching had "go" matching google and "test"
		// matching latest, which is how a rubric quietly fills up with nonsense
		// reasons; the digest prints those reasons, so they have to be true.
		if containsWord(hay, strings.ToLower(kw)) {
			matched = append(matched, kw)
		}
	}
	return float64(len(matched)) / float64(len(s.Keywords)), matched
}

// containsWord reports whether kw appears in hay as a word, where a word also
// breaks on hyphen, underscore and digits so "vector-database" and "gguf-2" work.
func containsWord(hay, kw string) bool {
	if kw == "" {
		return false
	}
	for from := 0; ; {
		i := strings.Index(hay[from:], kw)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(kw)
		beforeOK := start == 0 || !isWordByte(hay[start-1])
		afterOK := end == len(hay) || !isWordByte(hay[end])
		if beforeOK && afterOK {
			return true
		}
		from = start + 1
	}
}

func isWordByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '_' || b == '-':
		return true
	default:
		return false
	}
}

// Rank sorts by score, with ties broken on ID: two runs over the same input must
// produce the same digest, because a demo that reorders itself cannot be diffed.
func Rank(items []Item, w Weights, s Stack, now time.Time) []Scored {
	sum := w.Impact + w.Novelty + w.Relevance
	if sum <= 0 {
		sum = 1
	}
	out := make([]Scored, 0, len(items))
	for _, it := range items {
		rel, matched := s.Relevance(it)
		imp, nov := Impact(it), Novelty(it, now)
		out = append(out, Scored{
			Item: it, Impact: imp, Novelty: nov, Relevance: rel, Matched: matched,
			Score: (w.Impact*imp + w.Novelty*nov + w.Relevance*rel) / sum,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}
