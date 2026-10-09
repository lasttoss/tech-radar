package digest

import (
	"strings"
	"testing"
)

func item(id string) Item {
	return Item{
		ID: id, Title: id, Desc: "one line about it",
		URL: "https://example.com/" + id, Meta: []string{"Go", "MIT", "⭐1.2k"},
		Reason: "matches game, server",
	}
}

func TestBlockShapeIsTheShapeThatSurvivedDelivery(t *testing.T) {
	got := Block(1, item("owner/name"))
	lines := strings.Split(got, "\n")

	if len(lines) != 4 {
		t.Fatalf("want 4 lines (title, desc, reason, source; the original link is optional), got %d:\n%s", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "1  • <b>owner/name</b> · Go · MIT") {
		t.Fatalf("title line lost its meta: %q", lines[0])
	}
	if !strings.HasPrefix(lines[2], "📌") {
		t.Fatalf("reason line must start with the pin emoji, got %q", lines[2])
	}
	if !strings.Contains(lines[3], "<a href=\"https://example.com/owner/name\">") {
		t.Fatalf("source line lost its link: %q", lines[3])
	}
}

func TestBlockWithOriginalAddsOneLine(t *testing.T) {
	it := item("owner/name")
	it.Home = "https://home.example.com"
	if lines := strings.Count(Block(1, it), "\n") + 1; lines != 5 {
		t.Fatalf("want 5 lines when an original link exists, got %d", lines)
	}
}

func TestAppliedVerdictReplacesTheReasonLine(t *testing.T) {
	it := item("owner/name")
	it.Applied = &Applied{Label: "adopt", Evidence: "6/7 fields correct, 9.1s", Repo: "some-repo", Commit: "1a2b3c4"}
	got := Block(1, it)

	if !strings.Contains(got, "📌 Verdict: adopt") {
		t.Fatalf("a closed item must show its verdict: %s", got)
	}
	if !strings.Contains(got, "some-repo@1a2b3c4") {
		t.Fatal("an adopted item must name the commit that took it")
	}
	if strings.Contains(got, "Why it matters") {
		t.Fatal("the reason line should be replaced by the verdict, not repeated")
	}
}

func TestEscapeProtectsTextAndNotHrefs(t *testing.T) {
	if got := Escape(`a & b <c> "d"`); got != `a &amp; b &lt;c&gt; "d"` {
		t.Fatalf("escaping is wrong: %q", got)
	}
	it := item("owner/name")
	it.Desc = "tags & <angles>"
	got := Block(1, it)
	if !strings.Contains(got, "tags &amp; &lt;angles&gt;") {
		t.Fatalf("description was not escaped: %s", got)
	}
	if !strings.Contains(got, `<a href="https://example.com/owner/name">`) {
		t.Fatalf("href must stay raw or every link breaks: %s", got)
	}
}

func TestValidateRejectsWhatHasBrokenBefore(t *testing.T) {
	cases := map[string]string{
		"markdown bold": "1  • <b>x</b> · a · b\nDesc\n📌 Why it matters: x\n🔗 Source: y **bold**",
		"arrow":         "1  • <b>x</b> · a · b\nDesc\n📌 Why it matters: x → y\n🔗 Source: y",
		"no reason":     "1  • <b>x</b> · a · b\nDesc\n🔗 Source: y",
		"no source":     "1  • <b>x</b> · a · b\nDesc\n📌 Why it matters: x",
		"no items":      "just a header",
	}
	for name, msg := range cases {
		if err := Validate(msg); err == nil {
			t.Fatalf("%s should have been rejected", name)
		}
	}
}

func TestValidateAcceptsARealMessage(t *testing.T) {
	msg := Build("📡 header", []Item{item("a/one"), item("b/two"), item("c/three")})
	if len(msg) != 1 {
		t.Fatalf("small digest should be one message, got %d", len(msg))
	}
	if err := Validate(msg[0]); err != nil {
		t.Fatalf("a real message was rejected: %v", err)
	}
}

func TestBuildChunksBeforeTheTransportLimit(t *testing.T) {
	long := item("owner/long")
	long.Desc = strings.Repeat("padding sentence. ", 120)
	var items []Item
	for i := 0; i < 6; i++ {
		items = append(items, long)
	}
	msgs := Build("header", items)

	if len(msgs) < 2 {
		t.Fatal("this content cannot fit in one message and must be split")
	}
	for i, m := range msgs {
		if err := Validate(m); err != nil {
			t.Fatalf("chunk %d is invalid: %v", i, err)
		}
	}
}
