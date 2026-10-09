# Design notes

## Dry run is the default

`make demo` prints exactly what would be sent, and nothing in the repository can
reach the network without an explicit `--send` and a token. A pipeline whose
default action is to publish is a pipeline nobody dares run while debugging.

## The rubric is written down, and it says why

Impact, novelty and relevance, weighted, all three in `internal/rank` with tests.
The reason is not that a score is more objective than a person - it is that a score
can be argued with. When it ranks something stupid, the digest prints the words
that matched, so the argument is about the rubric instead of about taste.

Two decisions inside it are deliberate and tested:

- **Impact is log scaled.** The gap between 1k and 10k stars is a signal; the gap
  between 90k and 99k is noise.
- **Relevance matches whole words, with no stemming.** Substring matching had `go`
  matching `google` and `test` matching `latest`. Stemming would have fixed
  `tested`/`test` but made the rule unpredictable, and an unpredictable rubric
  cannot be defended when it misranks something. So `tested` does not match `test`,
  and the test says so.

## Fixtures and a golden digest

`testdata/fixtures/` holds real recorded responses, so the suite runs offline in
about a second and CI needs no token. `testdata/golden/payload.txt` is the digest
for a fixed instant: any change to ranking, escaping, chunking or verdict
attachment shows up as a diff instead of as a slightly different message nobody
reads.

That is also why the default scoring instant is the newest timestamp **in the
data**, not the wall clock. Scoring against the clock would make every recorded run
produce a different digest, and an artefact that changes on its own cannot be
diffed or reviewed.

## 3000 characters, and why

A digest of about 6.3k characters once went through a rich-markdown transport and
arrived split in the middle of an item. This pipeline talks to the messaging API
directly and enforces the limit in code, with a unit test for the chunking.

## The payload is a typed struct

`disable_web_page_preview` is a boolean. Building the request body from a
`map[string]string` is how it becomes the string `"true"`, and how the API rejects
it. The type is the fix, and there is a test that decodes the body back.

## Retry only what can succeed

A 5xx gets one retry; a 4xx does not. A rejected HTML entity will be rejected
identically the second time, so retrying it only doubles the noise. There is a test
for each path.

## gofmt and go vet instead of a pinned linter

A pinned linter version that lags the Go release turns CI red for reasons unrelated
to the code. The checks here are the ones that ship with the toolchain.

## Known gaps

- **No deduplication across sources.** A repository can appear both in the search
  results and in a release feed. The digest would show it twice; nothing merges
  them yet.
- **No scheduling in the repository.** The daily run is an external timer, so this
  is a program rather than a service.
- **No delivery state.** A failed send does not persist, so a rerun sends the day
  again. Idempotency belongs here and is not written yet.
- **The search API is the fragile source** without a token: rate limited per IP,
  and it failed on the first attempt during development.
