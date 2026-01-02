# Dialog Helper Service

API platform for real-time response suggestions for AAC clients. This repo contains two Go services:

- `inference`: public API for `/v1/infer`.
- `storage`: internal/admin API for config, API keys, and persistence.

Docs live in `docs/` and cover setup, deployment, and operations.

## Quick links
- `docs/01-overview.md`
- `docs/02-environment.md`
- `docs/03-yc-setup.md`
- `docs/04-db-schema.md`
- `docs/05-deploy.md`
- `docs/06-api.md`
- `docs/07-local-dev.md`
- `docs/08-testing.md`

## Repository layout
- `cmd/inference`: inference service entrypoint
- `cmd/storage`: storage/admin service entrypoint
- `internal/`: shared packages
- `api/`: OpenAPI and API Gateway config
- `scripts/`: bootstrap and helper scripts
- `testdata/`: small sample files for local tests

## Status
Work in progress.
