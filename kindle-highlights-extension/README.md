# Kindle Highlights Scraper - 旧研究用プロトタイプ

> 非推奨: このディレクトリは Kindle Notebook の DOM 出力を確認するための過去の研究用プロトタイプです。配布対象のブラウザ拡張機能ではなく、サーバー同期、ペアリング、トークン認証、現在のセキュリティモデルは実装していません。現在の Chrome 拡張機能の開発とリリースビルドには `../extension/` を使ってください。

Kindle Notebook から全書籍のハイライトを自動巡回で取得し、取得可能なフィールドを確認するための Chrome 拡張機能です。

このフェーズでは次を **実装していません**。

- サーバー送信
- Firebase 認証
- DB 保存
- 差分同期
- ローカルキャッシュ

目的は **「Kindle から実際に何が取れるかを JSON と console で確認すること」** です。

## 構成

```text
kindle-highlights-extension/
├── manifest.json
├── package.json
├── README.md
├── popup/
│   ├── popup.html
│   └── popup.js
├── content/
│   └── content.js
└── lib/
    ├── constants.js
    └── utils.js
```

今回は unpacked でそのまま読み込めるように、ビルド不要の素の JavaScript で作っています。

## 対応ドメイン

- `read.amazon.co.jp`
- `read.amazon.com`
- `read.amazon.co.uk`
- `read.amazon.de`
- `read.amazon.fr`
- `read.amazon.es`
- `read.amazon.it`
- `read.amazon.in`
- `read.amazon.ca`
- `read.amazon.com.au`
- `read.amazon.com.mx`
- `read.amazon.com.br`
- `read.amazon.nl`
- `read.amazon.sg`

別リージョンを使う場合は [manifest.json](./manifest.json) の `host_permissions` と `content_scripts.matches` に追加してください。

## インストール手順

1. Chrome で `chrome://extensions` を開く
2. 右上の `デベロッパーモード` をオンにする
3. `パッケージ化されていない拡張機能を読み込む` を押す
4. このディレクトリ `kindle-highlights-extension/` を選ぶ

## 使い方

1. Amazon にログイン済みの状態で `read.amazon.co.jp/notebook` などの Kindle Notebook を開く
2. 拡張アイコンをクリックする
3. `同期開始` を押す
4. ポップアップで進行状況を確認する
5. 完了後に Kindle Notebook タブの DevTools Console を開いて出力を見る
6. `JSONダウンロード` を押して結果を保存する

## 何を取るか

### 書籍単位

- `asin`
- `title`
- `author`
- `coverImageUrl`
- `notebookUrl`
- `sidebarText`
- `sidebarHtml`
- `sidebarAttributes`
- `sidebarDataset`
- `headerText`
- `headerHtml`
- `headerAttributes`
- `headerDataset`

### ハイライト単位

- `amazonAnnotationId`
- `amazonAnnotationIdCandidates`
- `highlightText`
- `note`
- `noteHtml`
- `color`
- `location`
- `page`
- `highlightedAt`
- `highlightedAtRaw`
- `metadataText`
- `metadataTokens`
- `domId`
- `classNames`
- `rawAttributes`
- `rawDataset`
- `rawText`
- `rawHtml`
- `links`
- `images`
- `sortOrder`
- `bookTitle`
- `bookAuthor`
- `bookAsin`

## Console 出力

同期完了後、Kindle Notebook タブの console に次を出します。

1. `=== Sample Highlight (Full Properties) ===`
2. `=== Summary ===`
3. `=== Per-book counts ===`
4. `=== Detected Fields ===`
5. 失敗書籍があれば `=== Errors ===`

## JSON 形式

ダウンロードされる JSON はおおむね次の形です。

```json
{
  "syncedAt": "2026-04-26T12:00:00Z",
  "amazonDomain": "read.amazon.co.jp",
  "summary": {
    "totalBooks": 23,
    "totalHighlights": 1847,
    "highlightsWithNotes": 234
  },
  "colorDistribution": {
    "yellow": 1500,
    "blue": 200
  },
  "perBookCounts": [],
  "detectedFields": {},
  "errors": [],
  "books": []
}
```

## 確認メモ

- 同期は Kindle Notebook タブ上の content script メモリに結果を保持しています
- JSON ダウンロードは **同期した Kindle Notebook タブを閉じる前** に実行してください
- Amazon 側の DOM が変わるとセレクタ修正が必要です

## 簡易チェック

構文チェックだけなら次で確認できます。

```bash
cd /Users/shout/ai-study-tool/kindle-highlights-extension
npm run check
```
