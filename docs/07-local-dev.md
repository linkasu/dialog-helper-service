# Local development

Local run uses in-memory storage and mock providers.

## 1) Start storage (memory backend)
```
export STORAGE_BACKEND=memory
export INTERNAL_AUTH_TOKEN=local-token
export ADMIN_MASTER_KEY=local-admin-key
export PORT=8081

go run ./cmd/storage
```

## 2) Create API key (admin)
```
curl -s -X POST http://localhost:8081/v1/admin/keys \
  -H "X-ADMIN-KEY: local-admin-key" \
  -H "Content-Type: application/json" \
  -d '{"client_id":"local","scopes":["infer"],"rate_limit_per_minute":60}'
```
Copy `api_key` from the response.

## 3) Start inference (mock STT/LLM)
```
export STORAGE_URL=http://localhost:8081
export INTERNAL_AUTH_TOKEN=local-token
export YC_FOLDER_ID=b1go18ta52hg00o51kkg
export STT_PROVIDER=mock
export LLM_PROVIDER=mock
export MOCK_STT_TRANSCRIPT="Hello there"
export MOCK_LLM_TEXTS="Yes,No,Maybe"
export PORT=8080

go run ./cmd/inference
```

## 4) Call /v1/infer
```
API_KEY=<api_key_from_admin>

curl -s -X POST http://localhost:8080/v1/infer \
  -H "X-API-KEY: ${API_KEY}" \
  -F "audio=@testdata/sample.mp3" \
  -F "payload={\"prompt\":\"Suggest short replies\",\"language\":\"ru-RU\"}" 
```
