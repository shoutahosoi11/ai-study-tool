# Question Sharing Flow 実装根拠

## 何を作ったか

「問題を作る」導線（KindleBookSection → QuestionQuizSessionModal）における投稿フローを改善した。

### 変更ファイル

1. **`frontend/src/pages/question/QuestionQuizSessionModal.tsx`**
   - `sessionMode?: 'generate' | 'solve'` プロップを追加
   - `sessionMode === 'generate'` の場合：明示的「投稿する」ボタンを削除し、コンパクトなbodyテキスト入力のみ表示
   - 解答まとめ画面の「閉じる」ボタンを「投稿して閉じる」に変更（generate モード時）
   - クリックで自動投稿 → `onShareSuccess()` でタイムラインに遷移

2. **`frontend/src/pages/question/KindleBookSection.tsx`**
   - `QuestionQuizSessionModal` に `sessionMode="generate"` を追加

3. **`frontend/src/pages/timeline/PostCard.tsx`**
   - `question` タイプ投稿の「問題セットを共有」ラベルを非表示（X/Twitter スタイル化）

## なぜこの設計にしたか

### sessionMode プロップでモードを分岐
- 既存の `shareEnabled` + `onShare` の仕組みをそのまま流用できる
- `solve` モード（他者の投稿から解く）では明示的「投稿する」ボタンを維持（将来の用途を壊さない）
- `generate` モードのみ自動投稿に変更

### 「投稿して閉じる」ボタンの動作
- クリック → `handleShare()` → 成功: `onShareSuccess()` でナビゲート（モーダルがアンマウントされる）
- 失敗: `shareError` を表示してモーダルを維持（ユーザーが再試行できる）
- `entries.length === 0`（問題を解いていない）場合は通常の `onClose()` を呼ぶ

### タイムライン自動リフレッシュ
- `onShareSuccess` が `navigate('/?tab=timeline')` を呼ぶ
- `App.tsx` で `{tab === "timeline" && <TimelinePage />}` の条件付きレンダリングにより、タブ切り替えで `TimelinePage` がリマウント
- `useTimeline` の `useEffect` が `loadPosts(true)` を実行 → 最新投稿を取得

## 他の選択肢との比較

### 案A: グローバル状態で timeline refresh をトリガー
- Context や Zustand でタイムラインのリフレッシュ関数を共有する
- **却下理由**: コンポーネント間の結合が増える。現在の navigate による re-mount で十分機能する

### 案B: 別のモーダルコンポーネントを `generate` モード用に作る
- `GenerateSessionModal` と `SolveSessionModal` に分割する
- **却下理由**: コードの重複が多い。`sessionMode` による分岐の方が変更量が少ない

### 案C: `onClose` コールバックを拡張して投稿ロジックを受け渡す
- `onCloseWithPost?: () => void` のような prop を追加する
- **却下理由**: `onShare` / `onShareSuccess` の既存の仕組みを活かせない。`handleShare` は既にエラーハンドリングを含んでいる
