// Package digest turns ranked items into the message the radar sends: fixed
// blocks, HTML escaped, chunked under the transport limit that once cut a report
// in half mid-item.
package digest

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxChars is the limit this package enforces per message. It is a constant with
// a reason: a 6.3k reply through a rich-markdown transport was split mid-item,
// which is why this pipeline talks to the messaging API directly.
const MaxChars = 3000

// Applied is a recorded verdict. Attaching it to an item is the difference
// between a news feed and a radar: the message says where the finding went.
type Applied struct {
	Label    string // adopt | watch | reject
	Evidence string // the measurement, not an adjective
	Repo     string
	Commit   string
}

// Item is the slice of a ranked item the digest needs, kept as its own type so
// this package does not depend on the ranking package.
type Item struct {
	ID      string
	Title   string
	Desc    string
	URL     string
	Home    string
	Meta    []string
	Reason  string
	Applied *Applied
}

// Escape makes text safe for an HTML parse mode. Callers escape the text and
// never the href, which is the mistake that silently breaks every link.
func Escape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// Block renders one item. The shape is fixed on purpose: title line, description,
// the reason line, then the links on their own lines so a thumb cannot miss.
func Block(n int, it Item) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d  • <b>%s</b>", n, Escape(it.Title))
	if m := strings.Join(it.Meta, " · "); m != "" {
		b.WriteString(" · " + Escape(m))
	}
	b.WriteByte('\n')
	if it.Desc != "" {
		b.WriteString(Escape(it.Desc) + "\n")
	}
	if it.Applied != nil {
		evidence := it.Applied.Evidence
		if it.Applied.Commit != "" {
			evidence += fmt.Sprintf(" (%s@%s)", it.Applied.Repo, it.Applied.Commit)
		}
		fmt.Fprintf(&b, "📌 Verdict: %s — %s\n", Escape(it.Applied.Label), Escape(evidence))
	} else {
		fmt.Fprintf(&b, "📌 Why it matters: %s\n", Escape(it.Reason))
	}
	fmt.Fprintf(&b, "🔗 Source: <a href=\"%s\">%s</a>\n", it.URL, Escape(it.ID))
	if it.Home != "" {
		fmt.Fprintf(&b, "🌐 Original: <a href=\"%s\">%s</a>\n", it.Home, Escape(it.Home))
	}
	return strings.TrimRight(b.String(), "\n")
}

// Build returns one or more messages, each below MaxChars.
func Build(header string, items []Item) []string {
	var msgs []string
	cur := header
	for i, it := range items {
		block := "\n\n" + Block(i+1, it)
		if utf8.RuneCountInString(cur+block) > MaxChars {
			msgs = append(msgs, cur)
			cur = header + block
			continue
		}
		cur += block
	}
	if strings.TrimSpace(cur) != "" {
		msgs = append(msgs, cur)
	}
	return msgs
}

// Validate rejects the shapes that have broken delivery before. It runs before
// anything is sent, so a bad digest fails loudly here instead of quietly there.
func Validate(msg string) error {
	if utf8.RuneCountInString(msg) > MaxChars {
		return fmt.Errorf("message is %d runes, over the %d limit", utf8.RuneCountInString(msg), MaxChars)
	}
	for _, bad := range []string{"**", "→", "__"} {
		if strings.Contains(msg, bad) {
			return fmt.Errorf("message contains %q, which the transport mangles", bad)
		}
	}
	blocks := strings.Split(msg, "\n\n")
	seen := 0
	for _, b := range blocks {
		if !strings.Contains(b, "• <b>") {
			continue
		}
		seen++
		for _, need := range []string{"📌", "🔗"} {
			if !strings.Contains(b, need) {
				return fmt.Errorf("item block %d has no %s line", seen, need)
			}
		}
		if lines := strings.Count(b, "\n") + 1; lines < 4 || lines > 5 {
			return fmt.Errorf("item block %d has %d lines, want 4 or 5", seen, lines)
		}
	}
	if seen == 0 {
		return errors.New("message has no item blocks")
	}
	return nil
}
