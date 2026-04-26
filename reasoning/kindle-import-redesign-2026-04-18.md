# Kindle ハイライトインポート 全面改修

## 何を作ったか

既存の `POST /api/highlights/import` を改修し、`GET /api/highlights/books` と book 単位での問題生成（`source_type: "kindle_book"`）を追加した。

### 追加・変更したファイル

| ファイル | 内容 |
|---|---|
| `backend/db/migrations/022_update_highlights_content_hash.sql` | 旧インデックス削除、既存ハッシュをNULLに、新インデックス作成 |
| `backend/internal/domain/kindle_book.go` | `KindleBook` struct 新規 |
| `backend/internal/domain/question.go` | `SourceTypeKindleBook = "kindle_book"` 追加 |
| `backend/internal/domain/highlight_repository.go` | `ListByUserIDAndASIN` / `ListBooksWithHighlightsByUserID` インターフェース追加 |
| `backend/internal/infrastructure/persistence/highlight_repository.go` | 上記2メソッドを生SQLで実装 |
| `backend/internal/usecase/highlight_usecase.go` | `ImportKindleResult` 型変更、新ハッシュ関数、`ListKindleBooks` 追加 |
| `backend/internal/usecase/question_source_resolver.go` | `SourceTypeKindleBook` case追加、`resolveKindleBookText` 実装 |
| `backend/internal/handler/dto/highlight_dto.go` | `ImportHighlightsResponse` 更新、`KindleBookResponse` 追加 |
| `backend/internal/handler/highlight_handler.go` | `Import` 修正（200/422）、`ListBooks` 追加 |
| `backend/cmd/main.go` | `GET /api/highlights/books` ルート追加 |
| `backend/internal/usecase/highlight_usecase_test.go` | 新フィールド・新ハッシュ対応、全件重複正常系テスト追加 |
| `backend/internal/usecase/question_source_resolver_test.go` | mockに不足メソッドのスタブ追加 |

---

## なぜその設計にしたか

### content_hash アルゴリズムの変更

旧: `sha256(userID + ":" + ASIN + ":" + content)`
新: `sha256("asin:" + asin + ":loc:" + location + ":content:" + normalize(content))`

理由:
- `user_id` はユニーク制約カラムとして別管理されており、ハッシュに含めるのは冗長
- 同書の同箇所を別ユーザーがインポートした場合、ハッシュが一致する方がデータの健全性が高い
- `location` を含めることで、同書内の同一文章が異なるロケーションに出現した場合の誤重複排除を防ぐ
- `normalizeContent`（`strings.Fields` + `ToLower`）で空白・大小文字の揺れを吸収する

### レスポンス構造の変更（`saved_count` / `duplicate_count` / `copy_protected_count`）

旧: `saved` + `skipped`（skipped は copy_protected と duplicate の合算）
新: 3つの独立したカウント

理由:
- コピー制限（Amazon の制約）と重複（再インポート）はクライアントにとって異なる意味を持つ
- フロントエンドが「新規XX件保存、XX件は既に保存済み、XX件はコピー制限」と表示できる

### book 単位での問題生成

`SourceTypeKindleBook = "kindle_book"` を追加し、`question_source_resolver.go` で ASIN に紐づく全ハイライトを連結して問題生成テキストとする。

理由:
- ハイライト単体は文章が短すぎて問題生成品質が低い
- 同書のハイライトをまとめることで文脈のある問題が生成できる
- 既存の `SourceTypeHighlight` の延長として実装でき、変更範囲が最小

### HTTP ステータスの整理

- 空リスト → 400
- 全コピー制限 → 422
- 部分成功・全成功・全重複 → 200（エラーではないため）

旧実装では成功時に 201 を返していたが、BulkUpsert は必ずしも新規作成ではないため 200 が正確。

---

## 他の選択肢と比較してなぜこれを選んだか

### content-only ハッシュ案（却下）

同書の異なるロケーションに同一文章が出現する可能性があり、誤って重複判定される。asin + location を組み合わせることで識別精度が高い。

### duplicate をエラーとして扱う案（却下）

全件重複は通常の再インポートシナリオ（週次同期など）で発生する。エラーにすると UX が悪化する。`duplicate_count` をレスポンスに含めることで、クライアントが状況を把握できる。

### 新規エンドポイント `GET /api/highlights/books` ではなく `GET /api/highlights?group_by=asin`（却下）

既存の List エンドポイントの拡張よりも、責務を分けた専用エンドポイントの方が Clean Architecture に沿っている。

---

## DBマイグレーション適用状況

- **適用済み**: migrations 018〜022 すべて Docker PostgreSQL コンテナに適用完了
- `highlights` テーブルに `book_title`, `book_author`, `asin`, `highlighted_at`, `source`, `updated_at`, `content_hash` カラムが存在することを確認
- `highlights_user_id_content_hash_idx` 部分ユニークインデックスが存在することを確認

---

## トレードオフ

- `duplicate_count = len(non_empty) - saved` の計算は BulkUpsert の RETURNING 行数に依存している。RETURNING が実装の詳細として変わった場合に影響する。
- `resolveKindleBookText` は全ハイライトをメモリに読み込む。ハイライト数が非常に多いユーザーでは負荷になる可能性があるが、現時点の規模では許容範囲。

## 将来の拡張性

- `ImportHighlightItem` に `source` フィールドを追加すれば Kindle 以外（楽天 Kobo など）も対応可能
- `buildContentHashKey` の分岐を `source` 別にすれば異なるプラットフォームのハッシュ衝突を防げる
- `ListBooksWithHighlightsByUserID` の `WHERE source = 'kindle'` を外せば全ソースに対応可能
