# Startwork Sub2API Fork Maintenance

This fork is the Startwork-maintained Sub2API distribution used by Startwork production.

## Repository Contract

- Fork repository: `startwork-im/sub2api`
- Upstream repository: official Sub2API repository, currently `Wei-Shaw/sub2api`
- Docker image: `startwork/sub2api`
- Canonical automation entrypoints:
  - `.github/workflows/startwork-upstream-sync.yml`
  - `.github/workflows/startwork-docker-release.yml`
- Automatic update semantics:
  - the upstream sync workflow tracks the newest upstream release tag and updates the canonical `upstream-v<tag>` runtime branch
  - pushes to the latest canonical upstream runtime branch auto-trigger the Docker build workflow
  - later commits pushed onto that runtime branch auto-trigger the same build path again
- Staging environment: not available yet

## Branch Contract

- `main` is the Startwork-maintained product branch.
- `upstream-v<tag>` is the canonical maintained runtime branch namespace for synced upstream releases.
- Existing `sync/upstream-<tag>` branches are legacy migration inputs only and must not be created for new tags.
- Startwork patches live on `main` and must survive upstream tag merges.
- The upstream sync workflow must create or update the latest canonical `upstream-v<tag>` branch from the newest available upstream tag.
- The Docker build workflow must auto-run for new canonical upstream branches and for new commits pushed onto them.
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
