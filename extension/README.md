# AI Study Tool Kindle 取り込み拡張機能

Chrome / Chromium 向けの Manifest V3 拡張です。Kindle Notebook ページでユーザーが明示的にボタンを押した場合だけ、表示中のハイライトを抽出して backend の Extension import API に送信します。

## セットアップ

```bash
cd extension
npm install
npm run typecheck
npm test
npm run build
```

Chrome の `chrome://extensions` でデベロッパーモードを有効にし、この `extension/` ディレクトリを「パッケージ化されていない拡張機能を読み込む」で読み込んでください。読み込み前に `npm run build` が必要です。

`manifest.json` は本番配布用です。本番では backend host permission を確定 API origin のみに絞ります。開発中に localhost / staging / Cloud Run preview を使う場合は `manifest.development.json` を参照し、「パッケージ化されていない拡張機能を読み込む」の前に開発用 manifest を使ってください。本番配布時に `*.run.app` や localhost を残さないでください。

Chrome 102 以降が必須です。これは `chrome.storage.local.setAccessLevel({ accessLevel: "TRUSTED_CONTEXTS" })` で token 保存先を trusted contexts に限定するためです。

現時点では lint / format 専用scriptは置いていません。公開前チェックは `npm run typecheck`、`npm test`、`npm run build` を必須とし、ESLint / Prettier は後続PRで導入します。

## ペアリング

1. Options page を開く。
2. Backend API URL を設定する。未設定の場合は pairing / import を開始せず、Options page で設定を求めます。
3. 「接続を開始」を押す。
4. 表示された `user_code` を Web 側 `/extension/connect` で承認する。
5. Options page で `approved` になったら token を取得する。

raw `ext_` token は `chrome.storage.local` に保存します。保存前に `chrome.storage.local.setAccessLevel({ accessLevel: "TRUSTED_CONTEXTS" })` が成功することを必須にします。未対応ブラウザや失敗時は fail closed し、token を保存しません。Extension token は期限付きで、期限切れ後は再接続が必要です。Options page には token の接続期限を表示します。

## 権限

- `storage`: extension token、API URL、最終取り込み時刻の保存に使います。

現時点ではページ注入は manifest の `content_scripts` で行うため、`activeTab` / `scripting` は不要です。将来、ユーザー操作による明示的な再注入が必要になった時点で追加します。

Host permissions は Kindle Notebook (`read.amazon.co.jp` / `read.amazon.com`) と backend API origin に限定しています。`<all_urls>`、`cookies`、`history`、`webRequest`、`unlimitedStorage` は使いません。本番配布用 `manifest.json` は `https://api.ai-study-tool.com/*` のような確定 origin のみを許可します。実際の本番 API origin が異なる場合は、配布前にここだけを更新してください。

## 取り込み動作

- 自動巡回はしません。
- content script は Kindle Notebook ページ上に小さな「ai-study-toolへ取り込む」ボタンを出します。
- クリック時だけ DOM から book title / author / asin / highlight / note / location を抽出します。
- content script は token を読みません。
- content script は `chrome.runtime.sendMessage` で抽出データだけを service worker に渡します。
- service worker が token を読み、`Authorization: Bearer ext_...` を付けて backend に送信します。
- 1回の送信上限は `MAX_IMPORT_HIGHLIGHTS=100` です。上限を超える場合は先頭100件のみ送信します。明示操作の範囲で取り込みを進めつつ、Amazon側DOM変更や巨大ページによる過剰送信を避けるためです。
- 上限到達時は、ページ上の検出件数と送信件数をUIに表示します。
- 抽出0件の場合はAPI送信せず、「ハイライトが見つかりませんでした」「Kindle Notebookの表示形式が変わった可能性があります」「対象ページがKindle Notebookか確認してください」という趣旨のエラーを表示します。

Backend の現行 import API は note 専用フィールドを受け取りません。extension は note text を抽出可能ですが、現時点では backend へ送信しません。後続PRで backend DTO / DB / import usecase に note field を追加できます。note はユーザーの個人メモなので、送信・保存・LLM prompt投入時は prompt redaction とログ非保存の方針を維持してください。

## セキュリティメモ

- token、pairing_id、raw response body は console に出しません。
- backend error body全文は表示せず、401 / 403 / 429 / 5xx を抽象化したユーザー向けメッセージに変換します。
- UI表示は `textContent` / DOM node 生成のみで、`innerHTML` は使いません。
- API URL は `http` / `https` のみ許可します。
- message type は allowlist で検証します。
- `postMessage` は使わず、extension内の `chrome.runtime.sendMessage` のみを使います。

## トークン漏洩 / 失効

Options page の `revoke / disconnect` は backend の `DELETE /api/v1/extension/tokens/self` を呼び、成功・失敗に関わらず local token を削除します。漏洩が疑われる場合は server-side でも `extension_tokens.revoked_at` を設定してください。

## Chrome Web Store 配布前確認

本番配布前に以下を確認してください。

- 確定 API origin だけに host permission を絞る。`*.run.app` と localhost は本番配布用manifestに残さない。現在の想定値は `https://api.ai-study-tool.com/*` です。
- 拡張機能アイコンを追加する。
- Chrome Web Store 用説明文を用意する。
- プライバシーポリシーを用意する。
- `storage`, Kindle Notebook host, backend API host の権限要求理由を明記する。
- Amazon Kindle Notebookページのみで動くことを説明する。
- 自動巡回しないこと、ユーザー操作でのみ取り込むことを説明する。
- token漏洩時の revoke 手順を運用ドキュメントと揃える。

## Kindle Notebook 利用規約リスク

この拡張はユーザーが表示しているKindle Notebookページから、ユーザー操作でのみハイライトを抽出します。自動巡回、大量スクレイピング、バックグラウンドでのページ列挙は行いません。対象hostも Kindle Notebook に限定します。それでもAmazon / Kindle Notebook側の利用規約やDOM変更の影響を受ける可能性があります。
