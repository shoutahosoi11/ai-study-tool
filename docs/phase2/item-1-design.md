# Item 1: Neon 接続セキュリティ強化

## 目的

本番 DB 接続が常に TLS を使うことをコードレベルで強制する。
接続文字列・Secret Manager の設定は既に完了済みのため、
今回の追加価値は **Go 起動時バリデーション** と **ドキュメント整備**。

## 現状 (Before)

```go
// cmd/main.go, cmd/question-worker/main.go, cmd/highlight-importer/main.go
db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
```

- sslmode の検証なし → 誤って `sslmode=disable` が設定されても無音で通る
- ローカルと本番で同じ接続コードを使っており、区別がない
- `DATABASE_URL` はすでに Secret Manager に登録済み、`sslmode=require` で運用中

## 変更後 (After)

```go
// 3つの cmd/*/main.go 共通
db, err := dbinfra.Open(os.Getenv("DATABASE_URL"))
```

- `APP_ENV=production` かつ `sslmode` が `disable` / 未指定の場合は起動を拒否
- ローカル開発（`APP_ENV` 未設定 or `development`）では従来通り動作
- 3つの cmd で同じファクトリ関数を共有

## 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `backend/cmd/main.go` | `dbinfra.Open()` を使用 |
| `backend/cmd/question-worker/main.go` | 同上 |
| `backend/cmd/highlight-importer/main.go` | 同上 |
| `backend/.env.example` | ローカルと本番の記載を分かりやすく整備 |

## 新規ファイル

| ファイル | 内容 |
|---------|------|
| `backend/internal/infrastructure/db/connection.go` | DB 接続ファクトリ + TLS バリデーション |
| `backend/internal/infrastructure/db/connection_test.go` | バリデーションロジックのユニットテスト |

## 実装手順

### 1. `backend/internal/infrastructure/db/connection.go`

```go
package db

import (
    "database/sql"
    "fmt"
    "net/url"
    "os"
    "strings"

    _ "github.com/lib/pq"
)

// Open は DATABASE_URL を受け取り、TLS バリデーション後に sql.DB を返す。
// APP_ENV=production の場合、sslmode=disable は起動エラー。
func Open(databaseURL string) (*sql.DB, error) {
    if err := validateTLS(databaseURL); err != nil {
        return nil, err
    }
    return sql.Open("postgres", databaseURL)
}

func validateTLS(databaseURL string) error {
    if os.Getenv("APP_ENV") != "production" {
        return nil // ローカル開発では不問
    }

    u, err := url.Parse(databaseURL)
    if err != nil {
        return fmt.Errorf("db: invalid DATABASE_URL: %w", err)
    }

    sslmode := u.Query().Get("sslmode")
    switch strings.ToLower(sslmode) {
    case "require", "verify-ca", "verify-full":
        return nil
    case "disable":
        return fmt.Errorf("db: sslmode=disable is not allowed in production")
    default:
        // 未指定は Neon のデフォルトが prefer なので本番では明示を要求
        return fmt.Errorf("db: sslmode must be explicitly set to 'require' or higher in production (got: %q)", sslmode)
    }
}
```

### 2. 各 cmd の変更

`sql.Open("postgres", os.Getenv("DATABASE_URL"))` を
`dbinfra.Open(os.Getenv("DATABASE_URL"))` に置き換える（3ファイル）。

import に `dbinfra "github.com/shout/ai-study-tool/backend/internal/infrastructure/db"` を追加。

### 3. `.env.example` 更新

```
# ── ローカル開発 ─────────────────────────────────────────────
DATABASE_URL=postgres://postgres:postgres@localhost:5432/ai_study_tool?sslmode=disable

# ── 本番 Neon (Secret Manager で管理) ───────────────────────
# Cloud Run からは必ず -pooler 付き URL (PgBouncer経由) を使うこと
# DATABASE_URL=postgresql://USER:PASS@ep-xxx-pooler.ap-southeast-1.aws.neon.tech/DB?sslmode=require
# APP_ENV=production
```

## 動作確認方法

```bash
# ローカル: APP_ENV未設定 → sslmode=disable でも通る
DATABASE_URL=postgres://...?sslmode=disable go run ./cmd/main.go

# 本番模擬: APP_ENV=production → sslmode=disable でエラー
APP_ENV=production DATABASE_URL=postgres://...?sslmode=disable go run ./cmd/main.go
# → "db: sslmode=disable is not allowed in production" で終了

# テスト
cd backend && go test ./internal/infrastructure/db/...
```

## ハマりポイント

- Neon の `-pooler` URL と通常 URL を混在させない。Cloud Run は必ず `-pooler` を使う
- `sslmode=prefer`（デフォルト）は TLS を試みるが失敗しても接続する。本番では `require` 以上を強制
- `url.Parse` は postgres スキームを普通に解析できる

## 代替案

| 案 | 不採用理由 |
|----|----------|
| `pq` の TLS config を直接設定 | 接続文字列に既に含まれるので二重管理になる |
| ENV チェックを起動スクリプトで行う | Go 外での検証はテスト不可 |
| `sslmode=verify-full` を強制 | Neon は `require` で十分、`verify-full` は証明書配布が必要 |
