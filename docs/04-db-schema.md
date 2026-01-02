# YDB schema

Run these statements using the YDB CLI (or any YQL client). Adjust database path for your environment.

## Tables

### results
Stores idempotent cache results.
```
CREATE TABLE results (
  dialog_hash Utf8,
  user_hash Utf8,
  stt_transcript Utf8,
  answer_candidates_json Utf8,
  meta_json Utf8,
  created_at Timestamp,
  PRIMARY KEY (dialog_hash)
);
```

TTL (1440 minutes):
```
ALTER TABLE results SET TTL = Interval("PT1440M") ON created_at;
```

### api_keys
Stores hashed API keys and access policy.
```
CREATE TABLE api_keys (
  api_key_hash Utf8,
  client_id Utf8,
  status Utf8,
  scopes_json Utf8,
  rate_limit_per_minute Int32,
  created_at Timestamp,
  revoked_at Timestamp?,
  PRIMARY KEY (api_key_hash)
);
```

### config
Stores system prompt and admin key hash.
```
CREATE TABLE config (
  key Utf8,
  value Utf8,
  updated_at Timestamp,
  PRIMARY KEY (key)
);
```

## Bootstrap
After tables are created, run:
```
go run ./cmd/bootstrap --system-prompt-file docs/system_prompt.txt
```
This will generate the admin master key and store the system prompt.
