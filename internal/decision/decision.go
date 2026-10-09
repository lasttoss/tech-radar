// Package decision is the half that makes this a radar and not a feed: every
// item that looked relevant has to be closed out, including the ones that lost.
package decision

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Verdict is one closed item.
type Verdict struct {
	Item     string `json:"item"`
	Verdict  string `json:"verdict"` // adopt | watch | reject
	Evidence string `json:"evidence"`
	Repo     string `json:"repo,omitempty"`
	Commit   string `json:"commit,omitempty"`
	Date     string `json:"date"`
}

// File is the on-disk shape of decisions/verdicts.json.
type File struct {
	Verdicts []Verdict `json:"verdicts"`
}

var valid = map[string]bool{"adopt": true, "watch": true, "reject": true}

// Load reads the verdict log and returns it keyed by item id, lower-cased so a
// source can ask about an item without caring how it was written down.
func Load(path string) (map[string]Verdict, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := make(map[string]Verdict, len(f.Verdicts))
	for _, v := range f.Verdicts {
		if v.Item == "" || !valid[v.Verdict] {
			return nil, fmt.Errorf("%s: item %q has verdict %q, want adopt/watch/reject", path, v.Item, v.Verdict)
		}
		if v.Evidence == "" {
			return nil, fmt.Errorf("%s: item %q has no evidence - a verdict without a measurement is an opinion", path, v.Item)
		}
		if v.Verdict == "adopt" && v.Commit == "" {
			return nil, errors.New(path + ": an adopted item must name the commit that took it")
		}
		out[strings.ToLower(v.Item)] = v
	}
	return out, nil
}

// Lookup returns the verdict for an item id, if it has one.
func Lookup(log map[string]Verdict, id string) (Verdict, bool) {
	v, ok := log[strings.ToLower(id)]
	return v, ok
}

// Counts summarises the log, which is what the README quotes.
func Counts(log map[string]Verdict) map[string]int {
	out := map[string]int{"adopt": 0, "watch": 0, "reject": 0, "total": len(log)}
	for _, v := range log {
		out[v.Verdict]++
	}
	return out
}
