# Startwork Sub2API Fork Maintenance

This fork is the Startwork-maintained Sub2API distribution used by Startwork production.

## Repository Contract

- Fork repository: `startwork-im/sub2api`
- Upstream repository: official Sub2API repository, currently `Wei-Shaw/sub2api`
- Docker image: `startwork/sub2api`
- Production deployment: fully automatic after the image workflow succeeds on `main`
- Staging environment: not available yet

## Branch Contract

- `main` is the Startwork-maintained product branch.
- `sync/upstream-<tag>` branches are created by the upstream tag sync workflow.
- Startwork patches live on `main` and must survive upstream tag merges.
- Merge conflicts, test failures, build failures, health check failures, or smoke test failures must block deployment or trigger rollback.

## Product Boundary

Startwork owns request ledgers, balance reservation, org charge ledgers, and customer-facing consumption display.
This fork owns provider gateway behavior and upstream usage/cost truth collection.
New provider channels must be added here, not as independent Startwork-side supplier plugins.

## Domestic Model Support Direction

Domestic OpenAI-compatible providers such as DeepSeek and Xiaomi MiMo require provider/account endpoint capability instead of a forced Responses API path.

Required capability shape:

- `endpoint_mode`: `responses_api`, `chat_completions_passthrough`, `anthropic_messages`
- `auth_scheme`: `bearer`, `api-key`, or custom header
- model names are forwarded as requested
- model lists remain in Sub2API or provider discovery, not in Startwork database

Known provider requirements:

- DeepSeek official API uses `https://api.deepseek.com/v1/chat/completions`.
- Xiaomi MiMo official API uses `https://api.xiaomimimo.com/v1/chat/completions` and `api-key` header.

## Production Rollout Guardrails

The Docker release workflow deploys by SSH to the production host and recreates only the Sub2API service.
It must preserve the existing Postgres, Redis, volumes, Docker network, and Startwork backend URL contract `http://sub2api:8080`.

Rollback requirement:

- record the previous container image id before replacing the service image
- if health or smoke checks fail, retag the previous image and recreate the Sub2API service
- fail the workflow after rollback so operators can inspect the failed image

## Required GitHub Secrets

Docker Hub:

- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

Production SSH:

- `PROD_SSH_HOST`
- `PROD_SSH_USER`
- `PROD_SSH_KEY`

Optional production overrides:

- `PROD_SUB2API_COMPOSE_DIR` defaults to `/opt/startwork/deployment`
- `PROD_SUB2API_SERVICE` defaults to `sub2api`
- `PROD_SUB2API_CONTAINER` defaults to `startwork-sub2api-1`
- `PROD_SUB2API_HEALTH_URL` defaults to `http://127.0.0.1:18080/health`
- `PROD_SUB2API_RUNTIME_BASE_URL` defaults to `http://127.0.0.1:18080`
- `PROD_SUB2API_SMOKE_KEY` enables `/v1/models` smoke test when present
- `PROD_DEEPSEEK_SMOKE_KEY` enables DeepSeek chat smoke test when present
- `PROD_XIAOMI_MIMO_SMOKE_KEY` enables Xiaomi MiMo chat smoke test when present
