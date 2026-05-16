# auth-playground developer entry points.
# `make help` lists targets.

.PHONY: help up down logs ps test test-signup test-signup-live discovery wait-hydra

help:                ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

up:                  ## Start Hydra + Kratos + Postgres + SSUI + Mailpit
	docker compose up -d

down:                ## Stop and remove containers (volumes preserved)
	docker compose down

logs:                ## Tail logs from the stack
	docker compose logs -f --tail=100

ps:                  ## Show stack status
	docker compose ps

test:                ## Run all Go tests in apps/api
	cd apps/api && go test ./...

test-signup:         ## Run only the SIGNUP-NN suite (hermetic)
	cd apps/api && go test -v ./internal/signup/...

test-signup-live:    ## Run SIGNUP-NN tests that talk to the live stack (requires `make up`)
	cd apps/api && AUTH_PLAYGROUND_LIVE=1 go test -v -run LiveDiscovery ./internal/signup/...

discovery:           ## Fetch Hydra's OpenID configuration (proves SIGNUP-01 / OV-01 plumbing)
	@curl -sS http://127.0.0.1:4444/.well-known/openid-configuration | python3 -m json.tool

wait-hydra:          ## Block until Hydra public port is healthy
	@until curl -sf http://127.0.0.1:4444/health/ready >/dev/null; do \
	  echo "waiting for hydra..."; sleep 1; \
	done; \
	echo "hydra ready"
