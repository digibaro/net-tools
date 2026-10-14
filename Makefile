COMPOSE      = docker compose
COMPOSE_DEV  = $(COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml
COMPOSE_LAN  = $(COMPOSE) -f docker-compose.yml -f docker-compose.lan.yml
COMPOSE_E2E  = $(COMPOSE) -f docker-compose.yml -f docker-compose.e2e.yml

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
