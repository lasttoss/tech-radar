# Recorded fixtures

Every file here is a recording of a live response, kept so the pipeline is
reproducible and testable without a network. Nothing is hand-written except the
curated scan rows, which are marked below.

| file | what it is | recorded |
|---|---|---|
| `hf-models.json` | `huggingface.co/api/models?sort=trendingScore&direction=-1&limit=12` | 2026-10-09 |
| `hf-papers.json` | `huggingface.co/api/daily_papers?limit=5` | 2026-10-09 |
| `gh-search.json` | `api.github.com/search/repositories?q=created:>2026-09-01&sort=stars&per_page=12` | 2026-10-09 |
| `gh-releases.json` | `api.github.com/repos/vllm-project/vllm/releases?per_page=3` | 2026-10-09 |
| `items.json` | normalised output of `make record`, all four sources (32 items) | 2026-10-09 |
| `items-scan-2026-09.json` | six rows from a twelve-month scan of the Hub and GitHub, the items the verdicts are about | 2026-09-28 |

## Two things the recordings taught us

1. **The search API is the fragile source.** It is rate limited per IP without a
   token, and it failed on the first attempt here: `API rate limit exceeded for
   210.245.74.82`. One retry a minute later succeeded. The repository does not hide
   this - `GITHUB_TOKEN` is optional and the error surfaces instead of turning into
   an empty digest, because a pipeline that reports a quiet day when it is actually
   blocked is worse than one that fails.
2. **Newest-and-most-starred is not the same as interesting.** The first recorded
   search returned a star-farmed repository at the top of the list. That is exactly
   what the relevance term in the rubric is for, and it is why the digest prints
   which words matched rather than just a rank.

## Re-recording

```
make record           # rewrites items.json from the live sources
UPDATE_GOLDEN=1 make test   # only if the digest changed on purpose
```

The raw files are re-recorded by hand, since each one is here to pin a shape that
the decoder tests depend on.
