# データベース migration

このプロジェクトでは backend migration を `backend/db/migrations/` に forward-only SQL file として置きます。local `Makefile` は並び順に従ってすべての `*.sql` file を `psql` で実行します。そのため、このディレクトリには `.down.sql` のような paired file を置かないでください。

## セキュリティ強化 migration

Phase 2 では次を追加しました。

- `031_security_hardening.sql`

既存の migration layout に合わせるため、意図的に 1 つの forward-only file にしています。

### 変更内容

- SHA-256 hashing 用に `pgcrypto` が存在することを保証します。
- `highlights.content_hash TEXT` が存在することを保証します。
- `highlights.source TEXT` が存在し、nullable であることを保証します。
- legacy source value を rename します。
  - `mobile_share` -> `share`
  - `kindle` -> `extension`
- 既存の空でない highlight content に `content_hash` を backfill します。
- 空 content と、unique index に違反する重複 row は `content_hash` を `NULL` のままにします。
- 既存の `source IS NULL` row は `NULL` のままにします。
- partial unique index を作成します。
  - `(user_id, content_hash)` 上の `idx_highlights_user_content`
  - `content_hash IS NOT NULL` の場合のみ
- ユーザー別 ingest rate limit 用に `rate_limit_counters` を作成します。

### 事前チェックリスト

migration 実行前に以下を記録してください。

```sql
SELECT COUNT(*) AS highlights_count FROM highlights;

SELECT source, COUNT(*)
FROM highlights
GROUP BY source
ORDER BY source NULLS FIRST;

SELECT COUNT(*) AS duplicate_hash_candidates
FROM (
    SELECT user_id, encode(digest(regexp_replace(trim(content), '\s+', ' ', 'g'), 'sha256'), 'hex') AS content_hash
    FROM highlights
    WHERE content IS NOT NULL
      AND trim(content) <> ''
) candidates
GROUP BY user_id, content_hash
HAVING COUNT(*) > 1;
```

あわせて以下も確認してください。

- Neon automatic backups / PITR が有効。
- migration connection string は Neon direct connection string であり、runtime 用 pooled connection string ではない。
- Phase 2 の application deploy が準備済み。新しい ingest path は `rate_limit_counters` を前提にします。
- rollback window と記録済み timestamp がある。

### Neon Console で実行する場合

1. Neon project を開く。
2. 対象 branch と database を選ぶ。
3. SQL Editor を開く。
4. `backend/db/migrations/031_security_hardening.sql` を貼り付ける。
5. 1 回だけ実行する。
6. query result と timestamp を deploy note に保存する。

### psql で実行する場合

Neon direct connection string を使います。

```sh
MIGRATION_DATABASE_URL="postgresql://USER:PASSWORD@HOST.neon.tech/DB_NAME?sslmode=require"

psql "${MIGRATION_DATABASE_URL}" \
  -v ON_ERROR_STOP=1 \
  -f backend/db/migrations/031_security_hardening.sql
```

新規 database では、すべての migration を順番に実行します。

```sh
MIGRATION_DATABASE_URL="postgresql://USER:PASSWORD@HOST.neon.tech/DB_NAME?sslmode=require"

for file in $(ls backend/db/migrations/*.sql | sort); do
  echo "==> ${file}"
  psql "${MIGRATION_DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${file}"
done
```

### golang-migrate を使う場合

この repository は現在 `golang-migrate` の命名規則を使っていません。運用上どうしても使う場合は、SQL を外部の deployment-only migration directory にコピーし、単一の `up` migration として置いてください。その directory は `backend/db/migrations/` の外に置きます。

外部 directory 例:

```text
ops/migrations/031_security_hardening.up.sql
```

実行例:

```sh
migrate \
  -path ops/migrations \
  -database "${MIGRATION_DATABASE_URL}" \
  up
```

### 実行後チェック

```sql
SELECT COUNT(*) AS highlights_count FROM highlights;

SELECT source, COUNT(*)
FROM highlights
GROUP BY source
ORDER BY source NULLS FIRST;

SELECT COUNT(*) AS hashed_highlights
FROM highlights
WHERE content_hash IS NOT NULL;

SELECT indexname, indexdef
FROM pg_indexes
WHERE indexname = 'idx_highlights_user_content';

SELECT to_regclass('public.rate_limit_counters') AS rate_limit_table;
```

application の smoke test:

- 新しい highlight を insert または import し、`content_hash` が入ることを確認する。
- 同じ正規化済み text を 2 回 import し、2 row 目を作らず既存 highlight を返すことを確認する。
- 設定済み ingest limit を超えて送信し、`429` と `Retry-After: 86400` を確認する。
- 既存 highlight list と question flow が引き続き読み込めることを確認する。

### 手動 rollback SQL

production rollback は Neon PITR を優先してください。SQL rollback は、application version がこれらの column や table に依存しないことを確認してから使います。

```sql
DROP INDEX IF EXISTS idx_highlights_user_content;

DROP TABLE IF EXISTS rate_limit_counters;

UPDATE highlights
SET source = 'mobile_share'
WHERE source = 'share';

UPDATE highlights
SET source = 'kindle'
WHERE source = 'extension';

ALTER TABLE highlights
    DROP COLUMN IF EXISTS content_hash;

ALTER TABLE highlights
    DROP COLUMN IF EXISTS source;
```

PITR を使った場合は、database branch の restore 後すぐに 1 つ前の application revision を redeploy してください。
