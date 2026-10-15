COMPOSE      = docker compose
COMPOSE_DEV  = $(COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml
COMPOSE_LAN  = $(COMPOSE) -f docker-compose.yml -f docker-compose.lan.yml
COMPOSE_E2E  = $(COMPOSE) -f docker-compose.yml -f docker-compose.e2e.yml

# Stamp local builds from git: 0.4.0 exactly on the v0.4.0 tag, 0.4.0-2-gabc1234 after it (-dirty
# with uncommitted changes). The commit date keeps the build cache valid for unchanged code.
ifndef NETTOOLS_BUILD_VERSION
NETTOOLS_BUILD_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
endif
ifndef NETTOOLS_BUILD_COMMIT
NETTOOLS_BUILD_COMMIT := $(shell git rev-parse HEAD 2>/dev/null)
endif
ifndef NETTOOLS_BUILD_DATE
NETTOOLS_BUILD_DATE := $(shell git log -1 --format=%cI 2>/dev/null)
endif
# Newest release tag before this commit and its date ("0.4.0@2026-10-14"): the "new" badge in the
# web app marks tools added after it (on tag v0.5.0 that is v0.4.0, so 0.5.0 shows its own additions).
ifndef NETTOOLS_LATEST_RELEASE
NETTOOLS_LATEST_RELEASE := $(shell t=$$(git describe --tags --abbrev=0 --match 'v[0-9]*' HEAD^ 2>/dev/null) && echo "$$t@$$(git log -1 --format=%cs $$t)" | sed 's/^v//')
endif
export NETTOOLS_BUILD_VERSION NETTOOLS_BUILD_COMMIT NETTOOLS_BUILD_DATE NETTOOLS_LATEST_RELEASE

.PHONY: dev up up-lan up-e2e down logs build test test-web test-api e2e lint new-tool gen-api

dev:      ; $(COMPOSE_DEV) up
up:       ; $(COMPOSE) up -d --build
up-lan:   ; $(COMPOSE_LAN) up -d --build
up-e2e:   ; $(COMPOSE_E2E) up -d --build --wait   # relaxed rate limits for the full e2e suite
down:     ; $(COMPOSE) down
logs:     ; $(COMPOSE) logs -f --tail=100
build:    ; $(COMPOSE) build

test: test-web test-api
test-web: ; cd apps/web && pnpm test --run
test-api: ; cd apps/api && go test -race ./...
e2e:      ; cd apps/web && pnpm test:e2e

lint:
	cd apps/web && pnpm lint && pnpm typecheck
	cd apps/api && go vet ./... && golangci-lint run && govulncheck ./...

gen-api:  ; cd apps/web && pnpm gen:api

# make new-tool name=dns-lookup [server=true] [category="DNS"]
new-tool:
	cd apps/web && node scripts/create-tool.mjs $(name) $(if $(server),--server) $(if $(category),--category "$(category)")
