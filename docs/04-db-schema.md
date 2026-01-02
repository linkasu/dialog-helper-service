# YDB schema

Apply schema using the migration tool:
```
export YDB_DSN="grpcs://ydb.serverless.yandexcloud.net:2135/?database=/ru-central1/.../dialog-helper-db"

go run ./cmd/migrate
```

The tool runs these statements:

## results
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

## api_keys
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

## config
```
CREATE TABLE config (
  config_key Utf8,
  config_value Utf8,
  updated_at Timestamp,
  PRIMARY KEY (config_key)
);
```

## Bootstrap
After tables are created, run:
```
go run ./cmd/bootstrap --system-prompt-file docs/system_prompt.txt
```
This will generate the admin master key and store the system prompt.
