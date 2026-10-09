package decision

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "verdicts.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadAndLookupIsCaseInsensitive(t *testing.T) {
	p := write(t, `{"verdicts":[{"item":"Owner/Model","verdict":"watch","evidence":"9.1s, 6/7 fields","date":"2026-09-28"}]}`)
	log, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := Lookup(log, "owner/model"); !ok {
		t.Fatal("a source should not have to match the case it was written in")
	}
	if got := Counts(log)["watch"]; got != 1 {
		t.Fatalf("want 1 watch, got %d", got)
	}
}

func TestAVerdictWithoutEvidenceIsRefused(t *testing.T) {
	p := write(t, `{"verdicts":[{"item":"a/b","verdict":"watch","date":"2026-09-28"}]}`)
	if _, err := Load(p); err == nil {
		t.Fatal("a verdict without a measurement is an opinion and must not load")
	}
}

func TestAnAdoptionWithoutACommitIsRefused(t *testing.T) {
	p := write(t, `{"verdicts":[{"item":"a/b","verdict":"adopt","evidence":"works","date":"2026-09-28"}]}`)
	if _, err := Load(p); err == nil {
		t.Fatal("an adoption must name the commit that took it")
	}
}

func TestAnUnknownVerdictWordIsRefused(t *testing.T) {
	p := write(t, `{"verdicts":[{"item":"a/b","verdict":"maybe","evidence":"hmm","date":"2026-09-28"}]}`)
	if _, err := Load(p); err == nil {
		t.Fatal("only adopt, watch and reject exist")
	}
}

func TestTheRealLogLoads(t *testing.T) {
	log, err := Load("../../decisions/verdicts.json")
	if err != nil {
		t.Fatalf("the shipped verdict log must satisfy its own rules: %v", err)
	}
	counts := Counts(log)
	if counts["total"] < 3 {
		t.Fatalf("want at least 3 verdicts, got %d", counts["total"])
	}
	if counts["reject"] == 0 {
		t.Fatal("a log with no rejections is a log of things nobody tried")
	}
}
