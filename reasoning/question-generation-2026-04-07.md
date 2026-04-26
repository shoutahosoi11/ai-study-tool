# 問題生成API 設計根拠

## 目的
ユーザーがアップロードした画像・ノート・Kindleハイライトから、AIが自動で問題を生成する。
Gemini APIを2段階プロンプトで使用し、品質の高い問題を並列生成する。

## 層構成と依存の向き
```
handler/dto → usecase → domain(interface) ← infrastructure
                ↓
          domain/entity (純粋Go struct, タグなし)
```
Clean Architectureの原則に従い、外側のレイヤーが内側に依存する。

## Questionエンティティの責務分離の判断
- Question: 問題そのもの（内容・選択肢・答え・解説）
- QuestionMeta: 問題のメタ情報（作成者・出典・AI生成フラグ）
- QuestionStats: 統計情報（解答数・正解率）
3つに分けることで単一責任の原則を守り、各エンティティが独立してテスト・変更できる。

## 2段階プロンプトにした理由
- Step1（ExtractPoints）: テキスト全体から重要ポイントを抽出
- Step2（GenerateQuestion）: 各ポイントから問題を生成（goroutineで並列）
単一プロンプトより各ステップを分離することでデバッグ容易性が高まり、
goroutineによる並列化でレイテンシを大幅削減できる。

## Gemini Flash/Proの使い分けの理由
- free plan → gemini-1.5-flash: 無料枠、十分な品質
- pro plan → gemini-1.5-pro: より高精度、200万トークンコンテキスト（大容量ノート対応）
プランに応じたモデル選択でコストと品質のバランスを取る。

## OCRをインターフェース経由にした理由
OCRの実装（Gemini Vision / AWS Textract / Tesseract）を後から差し替えられるように。
インターフェースにすることでusecaseのテスト時にモックに差し替えられる。

## sqlcとdomainの変換層の設計
sqlcが生成するRow型はDB都合のフィールド（null可能型など）を含む。
infrastructure/persistence層でdomain.Questionに変換することで、
domainがDBの詳細に依存しない。

## エラー処理方針
- 外部サービス障害（Gemini, S3）: 502 + GEMINI_ERROR / S3_ERROR コード
- バリデーション: 400 + VALIDATION_ERROR
- リソース不在: 404 + NOT_FOUND
- 想定外: 500 + INTERNAL_ERROR
全APIで統一フォーマット: {"data": ..., "error": {"code": "...", "message": "..."}}

## HTTPステータスコードの統一ルール
- 200: 成功
- 201: リソース作成成功
- 400: バリデーションエラー
- 401: 認証エラー
- 404: リソースが存在しない
- 502: Gemini/OCR外部依存失敗
- 500: 想定外エラー

## 不明点
- Kindleハイライト取得の実装方法（API未確認）

## 仮定
- 1ソースから最大5問まで生成する（過負荷防止）
- 選択肢は常に4択固定
- 記述問題はGeminiによる自動採点

## 採用案
- 2段階プロンプト + goroutine並列生成
- LLMClientインターフェースによるGemini実装の分離
- DIコンテナによる依存解決

## 却下案
- usecaseにプロンプト文面を直書き: プロンプト変更のたびにusecaseを変更する必要がある
- GORM使用: N+1問題・パフォーマンスの不透明さ

## 実装順序
1. DBマイグレーション
2. domain layer（entity + interface）
3. sqlcクエリ
4. infrastructure layer（Gemini, S3, OCR, persistence）
5. usecase layer
6. DI container
7. handler layer（DTO + handler）
8. tests
