# Verdicts

Every item that looked relevant gets closed out: `adopt`, `watch`, or `reject`. The machine-readable
copy is `verdicts.json`, and `internal/decision` refuses to load a verdict with no evidence, or an
`adopt` that does not name the commit that took it - a verdict without a measurement is an opinion, and
an adoption without a commit is a promise.

| item | verdict | evidence | took effect |
|---|---|---|---|
| `Qwen/Qwen3-VL-8B-Instruct` | **reject** | state reading came back empty: 3165 chars of thinking and no answer | - |
| `Qwen/Qwen2.5-VL-7B-Instruct` | **watch** | 6/7 frame fields correct, 9.1s, 62 tokens, local | baseline kept |
| `Anionex/agent-vision-toolkit` | **watch** | harness proven working first (200, 1.18 MB), so the failure was the model, not the pipeline | - |

Why this public copy has no `adopt` row: the real log does, and every one of them names a commit in a
repository that is not public, so they are left out rather than invented. The validation in
`internal/decision` is what keeps that honest - it will not load an `adopt` without a commit.

## What a verdict has to contain

1. A number or a log line. Not "looks promising".
2. The date, because a verdict about a model from six months ago is history, not advice.
3. For `adopt`: the repository and the commit that took it, so the claim can be checked.
4. A `reject` is a result, not a failure - the reject above is the reason a whole model class was
   ruled out before it reached a product.
