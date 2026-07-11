# AGENTS.md

このファイルは Codex がこのリポジトリで常に守る作業規約です。
タスク固有の手順は `.claude/skills/*/SKILL.md` を優先し、ここには常時適用するルールだけを置きます。
Codex がレビューする場合も、該当する `SKILL.md` のレビュー手順を同じように適用します。

## Scope

- 既存の挙動、APIパス、DBスキーマ、認証仕様を変えるときは、影響範囲を明示する。
- ユーザーの未コミット変更を勝手に戻さない。
- `backend/internal/repository/sqlcgen/` は生成物として扱い、必要なら元SQLや生成手順を確認してから変更する。
- secrets、APIキー、接続文字列をコードやフロントエンドにハードコードしない。

## Project Skills

- コードレビュー: `.claude/skills/code-review/SKILL.md`
- PR作成、CI確認: `.claude/skills/github-pr/SKILL.md`
- DB migration、SQL、repository変更: `.claude/skills/db-migration/SKILL.md`
- mobile実機確認、Expo、iOS署名: `.claude/skills/mobile-debug/SKILL.md`

該当する作業では、Codex も対象の `SKILL.md` を読んでから作業する。

## Architecture

依存方向は `handler -> usecase -> domain <- infrastructure` を守る。

- `handler`: HTTP入出力、認証済みユーザー取得、最低限のリクエスト検証、DTO変換だけを担当する。
- `usecase`: 業務ルール、トランザクション方針、外部サービス呼び出しの順序制御を担当する。
- `domain`: 純粋な型、定数、interface、ドメインエラーを置く。DBタグ、JSONタグ、外部SDK依存を入れない。
- `infrastructure` / `repository`: DB、Firebase、Gemini、Stripe、Cloud Runなど外部I/Oの実装を閉じ込める。
- request/response DTO と domain model を混ぜない。
- 単一ハンドラ専用のレスポンス型はハンドラ内ローカル定義、複数ハンドラで共有する型は `handler/dto` に置く。

## Go Backend

- DB接続は `DATABASE_URL` を使う。Cloud Run 本番では Secret Manager から注入する。
- Goは function 宣言を基本にし、無名関数の乱用を避ける。
- `context.Context` が必要な関数では第一引数に渡す。
- error は握りつぶさず、必要な文脈を付けて wrap する。
- `panic` を通常フローで使わない。
- `nil, nil` のような曖昧な返し方を避ける。
- transaction 内で外部APIを呼ばない。
- handler / usecase / repository の責務をまたぐショートカットをしない。

## DB / SQL / Repository

- SQL詳細は repository 層に閉じ込める。
- N+1、過剰SELECT、WHERE条件漏れ、pagination計算ミスを確認する。
- upsert / delete / migration は要件に対して冪等か確認する。
- 複数テーブルの整合性が必要な処理は transaction 境界を明示する。
- question sync では queue 更新と日次カウンタ予約を同一 transaction にする。
- migrations は `backend/db/migrations/NNN_*.sql` に置く。forward-only を基本にし、可能な限り `IF NOT EXISTS` などで冪等にする。

## Auth / Authorization

- 認証必須APIが middleware なしで公開されていないか確認する。
- middleware 初期化失敗時に fail-open しない。
- userID は認証済みcontextから取得し、リクエストボディ由来の userID を信用しない。
- 他人のデータを読む、更新する、削除する処理には明確な認可条件を置く。
- SNS的に他人の問題へ回答できる仕様は許可するが、他人の問題やハイライトへの note / explanation 書き込みは許可しない。

## Verification

backend変更時の基本確認:

```bash
cd backend && go build ./... && go test ./...
```

実行できなかった検証は、理由と残リスクを最終報告に書く。
