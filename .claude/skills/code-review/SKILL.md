---
name: code-review
description: Pull Requestやコード差分、ファイル単位のレビューを行う時に使用する。レビューして、PRチェックして、このコード見て、改善点を出して、バグがないか見て、設計的に問題ないか見て、などの依頼に適用する。
---

# Code Review Skill

このスキルは、コード・差分・Pull Request をレビューするためのものです。単なる感想ではなく、バグ、設計、保守性、規約準拠まで含めて評価してください。

## レビューの基本方針

- まず変更の目的を推定する
- その変更が要件に対して妥当か確認する
- バグ・設計・保守性・テスト観点をレビューする
- 指摘は本当に重要なものを優先する
- スタイルの好みより、正しさ・安全性・保守性を優先する
- 改善点がある場合は、可能なら具体的な修正方針まで示す

## 優先順位

1. ランタイムエラー、panic、nil参照、未定義動作
2. データ不整合、トランザクション境界の誤り、排他制御ミス
3. 認証・認可漏れ、情報漏洩、インジェクションなどのセキュリティ問題
4. Clean Architecture違反、責務分離の崩れ、依存方向の破壊
5. API仕様不整合、レスポンス形式の揺れ、バリデーション漏れ
6. テスト不足、異常系未検証
7. 可読性、命名、重複、軽微な改善点

## このリポジトリで特に見る観点

### Clean Architecture

依存関係は `handler -> usecase -> domain <- infrastructure` を守る。

- handlerにビジネスロジックが入っていないか
- usecaseから直接DBや外部APIを触っていないか
- domainにDBタグ、JSONタグ、外部依存が入っていないか
- infrastructureがdomainのinterfaceを実装しているか
- request/response DTO と domain が分離されているか

### Go Backend

- 本番DBは Neon PostgreSQL。Cloud SQL 接続、Cloud SQL Auth Proxy、`--add-cloudsql-instances` 前提の追加は禁止
- 接続文字列は `DATABASE_URL` として Secret Manager から Cloud Run に注入する
- Goはfunction宣言を基本とし、無名関数の乱用を避ける
- context.Contextが必要な層では第一引数に渡されているか
- errorは握りつぶさず、wrapして返す
- panicを使わない
- nil, nil のような危険な返し方をしない
- S3は使用禁止。Cloud Storage（GCS）Signed URL を使う
- GORMは禁止。sqlcを優先し、sqlcが合わない箇所だけ限定的に database/sql を使う
- AWS SDKを新たにimportしない

### DB / SQL / Repository

- N+1が起きていないか
- select対象やwhere条件が妥当か
- paginationのlimit/offset/page計算にミスがないか
- トランザクション境界が妥当か
- transaction内で外部APIを呼んでいないか
- repository層にSQL詳細が閉じているか
- upsertやdeleteの冪等性が要件に合っているか
- question sync では queue 更新と日次カウンタ予約が同一 transaction になっているか

### 認証・認可

- 認証必須のAPIが素通りしていないか
- middleware初期化失敗時にfail-openになっていないか
- userID取得が安全か
- 他人のデータを読めたり削除できたりしないか

### Frontend

- TypeScriptの型定義が適切か。anyは避け、unknown経由で型を絞る
- コンポーネントはfunction宣言を基本にする
- UIコンポーネントに色・余白・フォントを直書きしすぎていないか
- API通信が `src/api/` に閉じているか
- loading / error / empty stateが考慮されているか
- Firebase AuthのIDトークンをAPIリクエストヘッダに付与しているか
- 環境変数は `VITE_` プレフィックスを使い、シークレットをフロントに持たせていないか

### セキュリティ

- SQL injectionの危険
- XSSの危険
- 認証・認可漏れ
- secretsやAPIキーのハードコード
- 個人情報をログ出力していないか
- 無制限なCORS設定になっていないか
- Cloud StorageバケットがPublic accessになっていないか
- Cloud Runのサービスアカウントが最小権限になっているか

## 出力フォーマット

### Summary

- この変更が何をしようとしているかを短く要約

### Good

- 良い点、設計や実装で妥当な点

### Needs Improvement

- 改善点を重要な順に並べる
- 各項目に理由を書く
- 可能なら具体的な修正案を書く

### Questions

- 不明点
- 要件次第で判断が変わる点
- バックエンド仕様やAPI仕様の確認が必要な点
