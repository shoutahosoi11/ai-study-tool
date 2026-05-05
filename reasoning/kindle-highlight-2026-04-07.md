# Kindle Highlight API 設計メモ

## 現在の対象

highlight 周りは、Kindle 拡張 / モバイル共有 / モバイル WebView 同期から入ってくる学習素材を保存し、一覧表示・問題生成・解説追記に使う層になっている。

## 現在のデータモデル

`domain.Highlight` は次を持つ。

- 書誌情報: `book_title`, `book_author`, `asin`
- 本文情報: `content`, `location`, `highlighted_at`
- 追記情報: `explanation`
- 取り込み元: `source`, `source_app`, `source_url`
- 生成状態: `status`, `retry_count`, `requested_at`, `processing_at`, `completed_at`, `failed_at`, `last_error`
- 重複検知: `content_hash`

## source を分ける理由

- `kindle`: Notebook 由来の通常ハイライト
- `mobile_share`: 共有シート由来のテキスト

取り込み経路ごとに hash の作り方や UI の扱いを変えられるようにするため、source を残している。

## status を highlight に持つ理由

問題生成の初回待ちを highlight 単位で管理するため。

- `pending`: 未生成または再キュー待ち
- `processing`: worker が処理中
- `completed`: 初回生成済み
- `failed`: リトライ上限到達

これにより、保存と AI 生成を同期処理にせず分けられる。

## content_hash を使う理由

同じ本文を何度も保存しないため。現在は source ごとに hash の材料を変えている。

### Kindle

`source + asin + location + normalized content`

### Mobile Share

`source + source_app + source_url + book_title + book_author + normalized content`

### 理由

- Kindle は location を含めないと、同じ本の別箇所で同文が出た時に誤重複になる
- モバイル共有は ASIN がないことがあるので、共有元と書誌情報も混ぜる

## 現在の API 役割

- `POST /api/highlights/import`: Kindle ハイライト一括保存
- `POST /api/highlights/share`: モバイル共有から保存
- `POST /api/highlights/sync/check`: 既存 hash の事前確認
- `GET /api/highlights/books`: 保存済み Kindle 本一覧
- `GET /api/highlights/books/:asin/items`: ASIN 指定の highlight 一覧
- `GET /api/highlights/books/search/items`: 書名 / 著者での fallback 一覧
- `PUT /api/highlights/:id/explanation`: ユーザー解説の保存

## ASIN と書誌 fallback を両方持つ理由

ASIN が取れる時はそれを主キー的に使う方が安定するが、共有シートや DOM 状況によっては ASIN が欠けるケースがある。そのため UI から本ごとに辿れるよう、書名 + 著者の fallback を残している。

## explanation を highlight 側に持つ理由

問題の解説とは別に、「元ハイライトに対するユーザー自身の理解メモ」を保持したいから。これを次の問題生成時の context にも再利用できる。

## Clean Architecture 上の扱い

- handler は request/response と user 解決だけを担当
- usecase は hash 正規化、入力整形、重複件数の組み立てを担当
- repository は SQL と upsert を担当
- domain は JSON タグや DB タグを持たない

## 古い前提として捨てたもの

- 汎用の `CountHighlightsByUserID` ベースのページネーション中心設計
- 手動入力ハイライトだけを前提にした source モデル
- highlight 保存時点でその場で問題生成する前提
