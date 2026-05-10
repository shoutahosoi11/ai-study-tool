---
name: db-migration
description: Use when creating, reviewing, or modifying backend database migrations, SQL schema changes, sqlc queries, repository persistence logic, or data backfills.
---

# DB Migration Skill

## Workflow

1. 既存migrationの最新番号と命名規則を確認する。
2. 対象テーブル、index、repository/queryの使われ方を読む。
3. migrationは `backend/db/migrations/NNN_*.sql` に置く。
4. forward-onlyを基本にし、可能な限り `IF NOT EXISTS` / `IF EXISTS` で冪等にする。
5. repository変更ではSQL詳細をrepository層に閉じ込める。

## Checks

- N+1、WHERE条件漏れ、pagination計算ミスがないか。
- transaction境界が妥当か。
- transaction内で外部APIを呼んでいないか。
- user_id条件が抜けていないか。
- question syncではqueue更新と日次カウンタ予約が同一transactionか。

## Verification

```bash
cd backend && go build ./... && go test ./...
```
