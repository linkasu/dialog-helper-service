# AGENTS.md

This repository contains the serverless API platform for dialog helper prompts (STT + LLM).

## Project layout
- `cmd/inference` - API service for `/v1/infer`.
- `cmd/storage` - storage/admin service for persistence + API keys.
- `internal/` - shared Go packages (providers, storage, auth, config).
- `api/` - OpenAPI spec and API Gateway spec.
- `docs/` - setup, deploy, schema, API docs.
- `scripts/` - build/deploy helpers.

## Local development
- Run tests: `go test ./...`
- Apply schema (requires `YDB_DSN`, optional `YC_IAM_TOKEN`): `go run ./cmd/migrate`
- Bootstrap system prompt (requires `YDB_DSN`): `go run ./cmd/bootstrap --system-prompt-file docs/system_prompt.txt`
- Mock local inference: see `docs/07-local-dev.md`

## Build and deploy
- Build/push images: `REGISTRY_ID=... ./scripts/build-images.sh`
- Deploy containers: `./scripts/deploy-storage.sh` then `./scripts/deploy-inference.sh`
- API Gateway spec: `api/api-gateway.yaml` uses container IDs; replace placeholders before create/update (see `docs/05-deploy.md`).

## Conventions
- Go modules, gofmt, `context.Context` for external calls, log with stdlib.
- Stateless services: no in-memory persistence between requests.
- No secrets in repo; use environment variables or Yandex Cloud IAM/Lockbox.
