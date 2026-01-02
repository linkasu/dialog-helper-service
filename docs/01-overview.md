# Overview

This document describes the system scope, responsibilities, and high-level architecture.

## Scope
- Accepts audio + client prompt.
- Runs STT and LLM.
- Returns transcript + answer candidates.
- Minimal persistence via YDB (hashes and results only).

## Components
- API Gateway (Yandex API Gateway)
- Inference service (serverless container)
- Storage service (serverless container)
- YDB (serverless)
- Yandex SpeechKit STT and YandexGPT LLM

## Data flow
1. Client sends multipart request to `/v1/infer`.
2. Inference checks idempotency cache.
3. Inference calls STT.
4. Inference merges prompts and calls LLM.
5. Inference returns transcript + suggestions.
6. Inference asynchronously persists results via Storage.

