# Kindle ブラウザ拡張機能 + フロントエンド導線

## 何を作ったか

`read.amazon.co.jp/kp/notebook` からハイライトを取得するブラウザ拡張機能（Chrome MV3）と、フロントエンドの「Kindle 本から問題を作る」導線を実装した。

### ファイル構成

```
extension/
  manifest.json          MV3 manifest
  background.js          メッセージハブ・API POST
  content/
    notebook.js          Amazon notebook ページのスクレイパー（2モード）
    webapp.js            web app ↔ background のブリッジ
  popup/
    popup.html           拡張ステータス表示

frontend/src/
  types/kindle.ts        KindleBook / ExtensionKindleBook 型
  api/kindle.ts          listKindleBooks()
  api/questions.ts       generateQuestions() 追加
  hooks/useKindleSync.ts 拡張検出・本一覧取得・同期フロー
  pages/question/
    KindleBookSection.tsx  本一覧セクション（extension/backend 切替）
    KindleBookCard.tsx     本カード（同期・問題生成ボタン）
    QuestionPage.tsx       KindleBookSection を追加
```

---

## なぜその設計にしたか

### 2段階フロー（本一覧取得 → ハイライト同期）

v1 は `?asin=X` を直接開く設計だったが、初回同期前はフロントエンドの `/api/highlights/books` が空になる問題があった。

v2 では:
1. `LIST_BOOKS_REQUEST`: `read.amazon.co.jp/kp/notebook`（ASIN なし）を silent tab で開き、本一覧をスクレイプ
2. `SYNC_BOOK_REQUEST`: 選択した本の `?asin=X` を silent tab で開き、ハイライトをスクレイプして backend に POST

これにより初回でも Kindle 本一覧を表示できる。

### postMessage ブリッジ方式（extension ID 不要）

拡張機能 ID をフロントにハードコードしなくて済む。`content/webapp.js` が web app の `window` に接続し、postMessage を双方向にブリッジする。

### origin 制限 + requestId

- 送信側: `window.postMessage(data, window.location.origin)` で origin を指定
- 受信側: `event.origin !== window.location.origin` でチェック
- `requestId = crypto.randomUUID()` を各リクエストに付与し、`pendingRef` で未完了リクエストを追跡。未知の requestId のレスポンスは無視（stale / spoofed 防止）

### token 使い捨て

Firebase token を `pendingSyncs[tabId].token` に一時格納し、`handleNotebookHighlightData` で `var token = sync.token; delete pendingSyncs[tabId];` と先に削除してから `postImport(token, ...)` に渡す。Amazon ログインは不要（ブラウザの既存 Cookie を使用）。

### extension なし時のフォールバック

`extensionInstalled = false` のとき `/api/highlights/books`（保存済み本のみ）を表示する。同期ボタンは非表示。これにより拡張機能未インストール環境でも既存の問題生成フローは動作する。

---

## 他の選択肢と比較してなぜこれを選んだか

### `externally_connectable` + 固定 extension ID

- メリット: セキュアなチャネル
- 却下理由: extension ID をフロントの環境変数にハードコードする必要がある。Chrome Web Store 未配布の個人ツールでは ID が環境依存になる

### Extension popup で完結させる

- メリット: シンプル
- 却下理由: ユーザーが web app を離れてポップアップを操作する必要があり、UX が断絶する

### Backend が Amazon をスクレイピング

- 却下理由: Amazon Cookie を backend に送る必要があり、セキュリティリスクが高い

### `?asin=X` 直リンク前提（v1 の方式）

- 却下理由: 初回同期前は本一覧が空になる。Amazon notebook トップから本一覧を取得する2段階フローが必要

---

## トレードオフ

- `scrapeBookList` のセレクタは Amazon の DOM に依存。Amazon がページ構造を変更した場合はセレクタの修正が必要
- `waitForBookList` / `waitForHighlights` は最大15秒（30回 × 500ms / 20回 × 500ms）待つ。ネットワークが遅い環境では silent tab が残る可能性がある（`closeTab` は `chrome.runtime.lastError` を void して安全に処理）
- `pendingSyncs` は service worker のメモリに存在し、MV3 の service worker は非アクティブ時に終了する。長時間放置した後の同期はタイムアウトする可能性がある

## 将来の拡張性

- `notebook.js` の URL チェックに `read.amazon.com` を追加し、manifest の `host_permissions` を追加するだけで US 版対応可能
- `source_type` は backend 側で `SourceType` として拡張済み（Kobo 等も追加可能）
- `ImportHighlightItem` に `source` フィールドを追加することで、Kindle 以外のプラットフォームも同じエンドポイントで対応可能
