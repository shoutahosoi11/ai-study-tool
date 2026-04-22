# Kindle ハイライト一括インポート機能

## 何を作ったか

`POST /api/highlights/import` エンドポイントを追加し、Kindle の Amazon クラウドから取得したハイライトを一括保存できるようにした。

### 追加・変更したファイル

| ファイル | 内容 |
|---|---|
| `db/migrations/021_add_highlights_content_hash.sql` | `content_hash` カラム + `(user_id, content_hash)` 部分ユニークインデックス |
| `domain/highlight.go` | `ContentHash *string` フィールド追加 |
| `domain/highlight_repository.go` | `BulkUpsert` インターフェース追加 |
| `domain/errors.go` | `ErrAllCopyProtected` 追加 |
| `infrastructure/persistence/highlight_repository.go` | `BulkUpsert` 実装（batch INSERT ON CONFLICT DO NOTHING） |
| `usecase/highlight_usecase.go` | `ImportKindleHighlights` メソッド追加 |
| `handler/dto/highlight_dto.go` | `ImportHighlightsRequest` / `ImportHighlightsResponse` 追加 |
| `handler/highlight_handler.go` | `Import` ハンドラ追加 |
| `cmd/main.go` | `POST /api/highlights/import` ルート登録 |
| `usecase/highlight_usecase_test.go` | 3ケースのユニットテスト（新規） |

---

## なぜその設計にしたか

### 取得主体: クライアントサイド抽出

バックエンドが直接 Amazon をスクレイピングするのではなく、フロントエンドが Amazon 読書メモページをパースして整形済み JSON をバックエンドに POST する方式を採用。

理由:
- Amazon Cookie をバックエンドに渡す必要がなく、セキュリティリスクが低い
- Amazon の HTML 構造が変わってもフロントエンドのパースロジックだけ修正すればよい
- バックエンドは純粋にデータを受け取って保存するだけで済む

### 重複排除: content_hash（SHA256）

Amazon Kindle のハイライトに固有 ID がないわけではないが、非公式 API 経由で取得できる ID の安定性が不明なため採用しなかった。代わりに `sha256(userID + ":" + ASIN + ":" + content)` を `content_hash` として計算し、`(user_id, content_hash)` のユニーク部分インデックスで重複を防ぐ。

INSERT ON CONFLICT DO NOTHING により再インポートが冪等になる。

### 3状態のハンドリング

| 状態 | 条件 | レスポンス |
|---|---|---|
| 全成功 | 空 content = 0 | 201, warning なし |
| 部分取得 | 空 content > 0 かつ saved > 0 | 201, warning あり |
| 全失敗 | 全 content が空 | 422, ErrAllCopyProtected |

「コピー制限」= content が空白のみのアイテムとして扱い、バックエンドでスキップしてカウントする。

### 既存フローへの接続

`highlights` テーブルに保存されれば、既存の `POST /api/questions { source_type: "highlight", source_id: "<id>" }` でそのまま問題生成できる。接続のための追加変更は不要。

---

## 他の選択肢と比較してなぜこれを選んだか

### バックエンドスクレイピング案（却下）

- Amazon Cookie をフロントエンドから送る必要があり、セキュリティリスクが高い
- バックエンドに Amazon への外部 HTTP 依存が生じる
- Amazon のレート制限・ブロックへの対処が必要

### My Clippings.txt アップロード案（対象外）

- クラウドハイライトのみが今回の要件
- デバイスから USB 経由で取得するフローはユーザー体験が悪い

### kindle_highlight_id での重複管理（却下）

- 非公式 API での ID は構造が変わる可能性がある
- content_hash の方がデータ内容に基づく確実な重複排除ができる

---

## トレードオフ

- `result.Highlights` に含まれるのは「新規保存されたもの」のみ。重複でスキップされたハイライトはレスポンスに含まれない。`saved` とレスポンスの `highlights` 配列長が一致しない場合がある（クライアントに周知が必要）。
- batch INSERT のパラメータ数 `10` はマジックナンバー。カラム追加時に漏れる可能性あり（後続 PR で定数化）。

## 想定リスク

- Amazon がページ構造を変更するとフロントエンドのパーサーが壊れる。バックエンドは影響を受けない。
- 大量インポートを繰り返す悪用への対策（レート制限）は現時点で未実装。

## 将来の拡張性

- `ImportHighlightItem` に `source` フィールドを追加することで Kindle 以外（楽天 Kobo など）も同じエンドポイントで対応可能。
- `content_hash` の計算ロジックを `source` 別に変えれば、異なるプラットフォームのハイライトが衝突しない設計に拡張できる。
