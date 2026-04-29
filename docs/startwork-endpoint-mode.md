# Startwork Endpoint Mode Design

This document records the Startwork product requirement for provider-native endpoint support in the Sub2API fork.

## Problem

The current OpenAI chat completions path is not provider-native for all OpenAI-compatible providers.
For `/v1/chat/completions`, the current gateway path can route through the Responses API chain and then call `/v1/responses` upstream.
That works for providers that support OpenAI Responses API, but it fails for domestic OpenAI-compatible chat providers that only support chat completions.

Observed production evidence:

- DeepSeek official keys configured behind `api.nextrouter.io` can call `https://api.deepseek.com/v1/chat/completions` for `deepseek-v4-flash` and `deepseek-v4-pro`.
- DeepSeek official API returns 404 for `/v1/responses`.
- Xiaomi MiMo official OpenAI-compatible endpoint is `https://api.xiaomimimo.com/v1/chat/completions`.
- Xiaomi MiMo official curl examples use `api-key: $MIMO_API_KEY`, not `Authorization: Bearer ...`.
- Current Sub2API error logs showed inbound `/v1/chat/completions` being forwarded as upstream `/v1/responses` for the Xiaomi MiMo account.

## Goal

Provider/account routing must be controlled by explicit capabilities, not by model name hacks or Startwork-side rewrites.

Startwork must continue to preserve the client requested model id. Model lists and model mappings remain in Sub2API, not in the Startwork database.

## Account Capability Fields

The first implementation can store these in account `credentials` or `extra` before a schema-level UI migration is introduced.
Long term they may become first-class schema fields.

```json
{
  "endpoint_mode": "chat_completions_passthrough",
  "auth_scheme": "api-key",
  "auth_header": "api-key",
  "chat_completions_path": "/v1/chat/completions",
  "models_path": "/v1/models"
}
```

### endpoint_mode

Allowed values:

- `responses_api`: current Responses API upstream behavior. This remains the default for existing OpenAI Responses-compatible accounts.
- `chat_completions_passthrough`: forward inbound `/v1/chat/completions` to upstream chat completions without converting to `/v1/responses`.
- `anthropic_messages`: provider-native Anthropic messages path.

### auth_scheme

Allowed values:

- `bearer`: send `Authorization: Bearer <api_key>`.
- `api-key`: send `api-key: <api_key>` by default.
- `custom_header`: send the configured `auth_header: <api_key>`.

### Provider Defaults

DeepSeek official:

```json
{
  "endpoint_mode": "chat_completions_passthrough",
  "auth_scheme": "bearer",
  "base_url": "https://api.deepseek.com/v1",
  "chat_completions_path": "/chat/completions",
  "models_path": "/models"
}
```

Xiaomi MiMo official:

```json
{
  "endpoint_mode": "chat_completions_passthrough",
  "auth_scheme": "api-key",
  "base_url": "https://api.xiaomimimo.com/v1",
  "chat_completions_path": "/chat/completions",
  "models_path": "/models"
}
```

## Gateway Routing Rule

For inbound `/v1/chat/completions`:

1. Select account using existing group/account routing and failover.
2. Resolve account endpoint capability.
3. If `endpoint_mode == chat_completions_passthrough`, forward the original OpenAI chat completions body to upstream chat completions.
4. Apply channel model mapping only inside Sub2API when configured.
5. Use account auth scheme to sign the upstream request.
6. Preserve streaming behavior when `stream=true`.
7. Preserve usage/cost capture and failover handling.

For inbound `/v1/responses`:

- Keep existing Responses behavior.
- Do not silently route `/v1/responses` to chat completions unless a future explicit compatibility mode is designed.

## Non-goals

- Do not store model lists in Startwork.
- Do not rewrite model names in Startwork.
- Do not special-case DeepSeek or Xiaomi in Startwork runtime paths.
- Do not make `chat_completions_passthrough` the implicit default for all existing accounts.

## Implementation Slices

1. Add endpoint capability resolver for account credentials/extra.
2. Add upstream auth header builder based on `auth_scheme`.
3. Add chat completions passthrough forwarder.
4. Wire `GatewayHandler.ChatCompletions` to choose passthrough only when account capability says so.
5. Add tests for DeepSeek bearer and Xiaomi `api-key` header.
6. Add tests proving existing Responses accounts still use the current path.
