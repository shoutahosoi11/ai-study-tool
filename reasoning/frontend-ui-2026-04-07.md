# Frontend UI 設計根拠 (2026-04-07)

## 何を作ったか
- 認証画面（ログイン・サインアップ）
- 3タブレイアウト（タイムライン・問題・プロフィール）
- BottomNav、AppLayout、ProtectedRoute
- PostCard、QuestionCard、AnswerModal
- 共通コンポーネント（Button、Input、Avatar、Card、Spinner）
- デザイントークン（src/theme/index.ts）

## なぜこの設計にしたか

### 認証ガードの分離
useAuth はFirebase状態の購読のみに責務を限定し、
リダイレクト制御はProtectedRouteに分離した。
useAuth内でnavigateを呼ぶと、テスタビリティが低下し
Storybook等での単体確認ができなくなるため。

### タブ切り替えをクエリパラメータで管理
/?tab=question 方式を採用。
独立ルート（/questions等）にするとBottomNavの
アクティブ状態管理が複雑化するため却下。

### テーマトークンをsrc/theme/index.tsで一元管理
コンポーネント内に色・余白をハードコードしない方針。
Tailwindクラスに直接カラー値を書くとデザイン変更時の
修正コストが増大するため。

### 状態管理はuseState + カスタムフック
zustand等のグローバルストアは今回のスコープでは過剰。
useAuth・useTimelineのカスタムフックで十分に管理できる。

## トレードオフ
- テーマ値をstyle属性で参照しているため、
  Tailwindのユーティリティクラスと混在している箇所がある

## 既知の制約（別PRで対応）
- AnswerModalを閉じた後に問題一覧の正答率が更新されない
  → Question型にanswer_count/correct_countを追加し、
    close時にリストを再fetchする
- GET /api/questions バックエンドハンドラーが未実装（TODO状態）
  → バックエンドにListUserQuestionsハンドラーを追加する必要あり

## 想定リスク
- Firebase Authの初期化タイミングによりloading中に
  一瞬ルートが/loginにフラッシュする可能性がある
  → ProtectedRouteのSpinnerで対処済み

## 将来の拡張性
- いいね・リポスト・コメント送信はPostCardに追加しやすい構造
- プロフィール編集はProfilePageに編集モードを追加するだけで対応可能
- Kindleハイライト管理画面は問題タブ内に追加するか4タブ目として追加できる
