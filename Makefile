# Everything runs in a container so a fresh clone works on a machine with no Go
# installed: that is the difference between a repository and a screenshot.
IMAGE ?= golang:1.25
CACHE ?= techradar-gocache
RUN   := docker run --rm -v $(PWD):/app -v $(CACHE):/go -w /app $(IMAGE)

.PHONY: help demo demo-closed test lint build record digest send

help:
	@echo "make demo         full pipeline on recorded signals, prints what it would send"
	@echo "make demo-closed  the same pipeline over items that already have verdicts"
	@echo "make test         go test -race ./..."
	@echo "make lint         gofmt + go vet"
	@echo "make build        build the binary and run it once offline"
	@echo "make record       refresh the fixtures from the live APIs (needs network)"
	@echo "make digest       write today's digest into digests/ from the fixtures"
	@echo "make send         deliver to the messaging API; needs .env, never the default"

demo:
	@$(RUN) go run ./cmd/radar --limit 6

demo-closed:
	@$(RUN) go run ./cmd/radar --items testdata/fixtures/items-scan-2026-09.json --limit 6

test:
	@$(RUN) go test -race ./...

lint:
	@$(RUN) sh -c 'test -z "$$(gofmt -l . | tee /dev/stderr)" && go vet ./... && echo "gofmt: clean · go vet: clean"'

# -buildvcs=false: inside the container, git refuses the mounted repository as a
# different owner and `go build` fails with "error obtaining VCS status". The build
# does not need the stamp; a fresh clone from GitHub keeps it in CI.
build:
	@$(RUN) sh -c 'go build -buildvcs=false -o /tmp/radar ./cmd/radar && /tmp/radar --items testdata/fixtures/items-scan-2026-09.json --limit 3 >/dev/null && echo "build ok, binary ran offline"'

record:
	@$(RUN) go run ./cmd/radar --record testdata/fixtures/items.json --since-days 30

digest:
	@$(RUN) go run ./cmd/radar --limit 12 > digests/$$(date -u +%F).generated.txt
	@echo "wrote digests/$$(date -u +%F).generated.txt"

send:
	@test -f .env || { echo "copy .env.example to .env and fill it in first"; exit 1; }
	@docker run --rm --env-file .env -v $(PWD):/app -v $(CACHE):/go -w /app $(IMAGE) go run ./cmd/radar --send
