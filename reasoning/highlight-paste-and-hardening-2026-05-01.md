# Highlight Paste And Hardening

## 何を作ったか

- `POST /api/highlights/paste` を追加し、既存 DTO に合わせて `content`, `book_title`, `book_author`, `source_url`, `source_app` を受け取るようにした。
- `POST /api/highlights/import` と `POST /api/highlights/share` に Phase 1 の正規化・検証を追加した。
- highlight の `source` 値を経路ベースの `extension`, `share`, `paste` に統一した。
- `content_hash` を正規化後本文の SHA-256 hex に統一し、paste 重複時は既存 ID を返すようにした。
- Postgres の `rate_limit_counters` を使う per-user ingest rate limit middleware を追加し、ingest 系 route に適用した。
- `backend/db/migrations/031_security_hardening.sql` を forward-only で追加し、source 移行、content_hash 再計算、部分 UNIQUE index、rate limit table を作成するようにした。
- `backend/db/sqlc/highlights.sql` と `backend/db/sqlc/rate_limits.sql` に query を追加し、sqlc 生成物を更新した。

## なぜこの設計にしたか

- 既存改修は「後段追加」にした。既存の空文字チェックや copy-protected 扱いを先に残し、その後で正規化・長さ・行数・URL 検証をかけることで、既存挙動を読みやすく保ちながら入力境界を強化できるため。
- レート制限のストレージには Postgres を使った。既存の永続化基盤が Neon/Postgres で、Phase 2 のために Redis を増やすと運用面と障害面が増えるため。`INSERT ... ON CONFLICT DO UPDATE` によって atomic increment も満たせる。
- `content_hash` は `(user_id, content_hash) WHERE content_hash IS NOT NULL` の部分 UNIQUE にした。既存データは NULL を許容しつつ、新規の重複検出だけを DB 制約で守れるため。
- `source` は経路ベースの名前に統一した。本番未公開の今が修正コストの底で、`kindle` や `mobile_share` のような具体 UI/アプリ由来の名前より、`extension`, `share`, `paste` の方が保存経路を表せる。
- 将来 `image_ocr` などを追加できるよう、DB enum や CHECK 制約で厳密には縛っていない。コード上の定数は使うが、永続化層は拡張余地を残している。

## 他の選択肢と比較してなぜこれを選んだか

- Redis を使う案は高速だが、この段階では新しいインフラ依存が増える。1日100件の ingest 制限なら Postgres の atomic update で十分。
- handler だけで正規化する案は、import/share/paste で重複しやすい。usecase で共通化すると保存前のドメイン入力を一箇所で守れる。
- `content_hash` に source や title を含める案は、同じ本文が経路違いで重複保存される。今回は「正規化後本文の重複」を止める要件を優先した。
- source を旧値のまま残す案は後方互換には強いが、本番未公開の今なら後から全クライアントに旧名を背負わせるより、ここで統一する方が保守しやすい。

## 既存の挙動が変わる入力ケース

- `import` / `share` で 301 文字以上の本文は `400` になる。
- `import` / `share` で 21 行以上の本文は `400` になる。
- `import` / `share` で 1 行が 301 文字以上の本文は `400` になる。
- `share` / `paste` で `javascript:`, `data:`, `file:` など http/https 以外の `source_url` は `400` になる。
- 本文に URL が含まれる場合、保存前に本文から URL が除去される。URL 除去後に空になる本文は `400` になる。
- `book_title` は 201 文字以上、`book_author` は 101 文字以上なら `400` になる。
- 同じユーザーで正規化後本文が同じ highlight は、source や title が違っても重複扱いになる。
- response の `source` は `kindle` から `extension`、`mobile_share` から `share` に変わる。

## 手動ロールバック手順

障害時は Phase 3 で人間が確認してから、必要なものだけを手動実行する。

```sql
DROP INDEX IF EXISTS idx_rate_limit_period;
DROP TABLE IF EXISTS rate_limit_counters;

DROP INDEX IF EXISTS idx_highlights_user_content;

UPDATE highlights
SET source = 'mobile_share'
WHERE source = 'share';

UPDATE highlights
SET source = 'kindle'
WHERE source = 'extension';

UPDATE highlights
SET content_hash = NULL
WHERE content_hash IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS highlights_user_id_content_hash_idx
    ON highlights (user_id, content_hash)
    WHERE content_hash IS NOT NULL;
```

## 別 PR で追従が必要な参照箇所

- `frontend/src/types/highlight.ts`: `source: 'kindle'` の literal type。
- `extension/background.js`: `source:kindle:asin:...` 形式で content hash を計算している。
- `docs/mobile-share-api.md`: response 例と説明で `mobile_share` を参照している。
- `mobile/src/api/highlights.ts`: `source` は string だが response 契約として新値を受ける。
- `mobile/src/api/kindle.ts` と `frontend/src/types/kindle.ts`: `source` は string だが book response に新値 `extension` が入る。
- `mobile/App.tsx`: `HighlightResponse.source` を受け取る画面がある。現状は明示比較は少ないが表示・集計の追従確認が必要。
