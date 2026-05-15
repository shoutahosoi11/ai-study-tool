# Item 4: 構造化ログ（slog）+ Cloud Logging 連携

## 目的

- Cloud Logging でクエリ可能な構造化 JSON ログを出力する
- 既存の `log.Printf` 呼び出しを最小限の変更で JSON 化する
- 重要なビジネスイベントは slog の key-value attrs を使う

## 現状 (Before)

```
// 非構造化テキスト
2024/01/01 12:00:00 question_generation_event=job_created user_id=xxx ...
```

- Cloud Logging で `jsonPayload` ではなく `textPayload` に格納される
- フィルタリング・アラート設定が困難

## 変更後 (After)

```json
{
  "severity": "INFO",
  "message": "question_generation_event=job_created",
  "timestamp": "2024-01-01T12:00:00Z",
  "user_id": "xxx",
  "job_id": "yyy",
  "book_key": "asin:zzz"
}
```

- `severity` フィールドで Cloud Logging のログレベルが自動認識される
- key-value attrs でフィルタリング可能

## 実装方針

### Cloud Logging との連携方法

**SDK 不使用**（Cloud Run stdout 自動転送を活用）:
- Cloud Run は stdout を Cloud Logging に自動転送する
- `slog.JSONHandler` + Cloud Logging 向けフィールド名変換で十分
- `cloud.google.com/go/logging` SDK は不要（依存追加なし）

### 既存 `log.Printf` の扱い

Go 1.21 の仕様:
> `slog.SetDefault(logger)` を呼ぶと、`log.Printf` 等も slog にリダイレクトされる

→ **既存 58 箇所の `log.Printf` を変更せずに JSON 化できる**。
重要な構造化イベントのみ `slog.Info/Error` に移行。

## 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `backend/cmd/main.go` | `logging.Setup()` を早期呼び出し |
| `backend/internal/usecase/question_sync_usecase.go` | key イベントを slog attrs に移行 |
| `backend/internal/usecase/question_worker_usecase.go` | 同上 |

## 新規ファイル

| ファイル | 内容 |
|---------|------|
| `backend/internal/logging/logger.go` | slog セットアップ + Cloud Logging 変換 |
| `backend/internal/logging/logger_test.go` | フィールド名変換のユニットテスト |

## 実装詳細

### `backend/internal/logging/logger.go`

```go
package logging

import (
    "log/slog"
    "os"
)

// Setup initializes the global slog logger.
// APP_ENV=production → JSON (Cloud Logging compatible)
// otherwise         → Text (human readable)
func Setup(appEnv string) {
    var handler slog.Handler

    if appEnv == "production" {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level:       slog.LevelInfo,
            ReplaceAttr: cloudLoggingReplaceAttr,
        })
    } else {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelDebug,
        })
    }

    slog.SetDefault(slog.New(handler))
}

// cloudLoggingReplaceAttr maps slog field names to Cloud Logging conventions.
// https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
func cloudLoggingReplaceAttr(_ []string, a slog.Attr) slog.Attr {
    switch a.Key {
    case slog.LevelKey:
        a.Key = "severity"
        if level, ok := a.Value.Any().(slog.Level); ok {
            a.Value = slog.StringValue(cloudLoggingSeverity(level))
        }
    case slog.MessageKey:
        a.Key = "message"
    case slog.TimeKey:
        a.Key = "timestamp"
    }
    return a
}

func cloudLoggingSeverity(level slog.Level) string {
    switch {
    case level >= slog.LevelError:
        return "ERROR"
    case level >= slog.LevelWarn:
        return "WARNING"
    case level >= slog.LevelInfo:
        return "INFO"
    default:
        return "DEBUG"
    }
}
```

### 各 cmd での呼び出し

```go
// main() の最初に追加
logging.Setup(os.Getenv("APP_ENV"))
```

### question_sync_usecase.go の移行例

```go
// Before:
log.Printf("question_generation_event=job_created user_id=%s job_id=%s ...", ...)

// After:
slog.Info("question_generation_event=job_created",
    "user_id", userID.String(),
    "job_id", job.ID.String(),
    "book_key", job.BookKey,
    "highlight_count", len(highlightIDs),
    "reason", job.Reason,
)
```

移行対象: `question_sync_usecase.go` と `question_worker_usecase.go` の key イベントのみ。
他の `log.Printf` は slog.SetDefault でリダイレクトされるため移行不要。

## 動作確認方法

```bash
# ローカル: テキスト出力
APP_ENV=development go run ./cmd/main.go

# 本番模擬: JSON 出力確認
APP_ENV=production go run ./cmd/main.go 2>&1 | head -5 | python3 -m json.tool

# Cloud Logging でのフィルタ例
# severity="ERROR"
# jsonPayload.user_id="xxx"
```

## ハマりポイント

- `log/slog` は Go 1.21 以上が必要。`go.mod` の Go バージョンを確認すること
- `slog.SetDefault` は `log.Printf` もリダイレクトするが、`log.Fatal/Fatalf` は `os.Exit(1)` を内部で呼ぶため引き続き機能する
- Cloud Run の構造化ログ認識には `timestamp` フィールドが文字列でなく RFC3339 形式が望ましい（`slog` のデフォルトは time.Time → 自動変換される）

## 代替案

| 案 | 不採用理由 |
|----|----------|
| Cloud Logging Go SDK | 依存追加・SA 権限追加が必要、stdout 転送で十分 |
| zerolog/zap | 外部依存、slog で要件を満たせる |
| 全 log.Printf を slog に移行 | 58 箇所 → リスク大、SetDefault で不要 |
