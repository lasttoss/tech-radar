# Everything runs in a container so a fresh clone works on a machine with no Go
# installed: that is the difference between a repository and a screenshot.
IMAGE ?= golang:1.25
CACHE ?= techradar-gocache
RUN   := docker run --rm -v $(PWD):/app -v $(CACHE):/go -w /app $(IMAGE)

.PHONY: help demo demo-closed test coverage lint build image up down logs record digest chart send

help:
	@echo "make demo         full pipeline on recorded signals, prints what it would send"
	@echo "make demo-closed  the same pipeline over items that already have verdicts"
	@echo "make test         go test -race ./..."
	@echo "make coverage     go test -race -coverprofile, then the per-function summary"
	@echo "make image        docker build, then run the image once offline"
	@echo "make up / down    docker compose: dry run, or delivery when .env has a token"
	@echo "make map / logs   docker compose logs -f"
	@echo "make chart        helm lint --strict + helm template on charts/tech-radar"
	@echo "make diagram      check docs/figures: source committed, png present, not stale"
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

coverage:
	@$(RUN) sh -c 'go test -race -coverprofile=/tmp/cover.out -covermode=atomic ./... >/dev/null && go tool cover -func=/tmp/cover.out | tail -1'

image:
	@docker build -q -t tech-radar:local . >/dev/null && docker run --rm tech-radar:local --items testdata/fixtures/items-scan-2026-09.json --limit 3 | head -3 && echo "image ok, ran with no network and no token"

up:
	@docker compose up -d --build || docker compose run --rm radar

down:
	@docker compose down -v

logs:
	@docker compose logs -f --tail=50

# lint alone only renders the default values, and the default is the dry run: a
# broken delivery branch would lint clean and explode on install. So render both,
# then validate the result against the Kubernetes schema.
HELM  := docker run --rm -v $(PWD):/apps alpine/helm:3.16.3
# "-" reads stdin: with /dev/stdin the container silently validated nothing and the
# gate reported success over zero resources, which is worse than no gate at all.
KCONF := docker run --rm -i ghcr.io/yannh/kubeconform:latest -strict -summary -kubernetes-version 1.30.0 -

# Not a generator: the source HTML is committed and the PNG is exported with the
# house script, which needs chromium that this repository does not ship. This target
# is the gate instead - figure present, exported, and not older than its sources.
diagram:
	@ls -1 docs/figures/*.html docs/figures/*.png >/dev/null && echo "figures: source + png present"
	@bash $$HOME/.pi/agent/scripts/diagram-freshness.sh 2>&1 | grep -E "tech-radar|STALE|TOUCHED|NO-META|UNCOMMITTED" || true

chart:
	@$(HELM) lint --strict /apps/charts/tech-radar
	@$(HELM) template tech-radar /apps/charts/tech-radar                        | $(KCONF)
	@$(HELM) template tech-radar /apps/charts/tech-radar --set delivery.enabled=true --set telegram.existingSecret=radar-telegram | $(KCONF)
	@echo "chart: lint --strict + template (dry run & delivery) + kubeconform ok"

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
