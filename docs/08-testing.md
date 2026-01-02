# Testing

## Unit tests
```
go test ./...
```

## Local smoke test
See `docs/07-local-dev.md` and run the `curl` request after starting both services.

## Remote smoke test
After deployment:
```
curl -s -X POST https://<gateway-or-inference-url>/v1/infer \
  -H "X-API-KEY: <client-key>" \
  -F "audio=@testdata/sample.mp3" \
  -F "payload={\"disabled_person_biography\":\"Adult user, polite tone\",\"messages\":[{\"role\":\"speaker\",\"audio\":true}],\"language\":\"ru-RU\"}" 
```
