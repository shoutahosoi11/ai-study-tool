# Security Foundation

## 何を作ったか

- `backend/internal/domain/text_normalizer.go` を追加し、ハイライト本文とメタデータに共通利用できる正規化関数を実装した
- `backend/internal/domain/text_validator.go` を追加し、文字数、行数、行長、URL の純粋な検証関数を実装した
- 各関数に対するテストとして `text_normalizer_test.go` と `text_validator_test.go` を追加した

## なぜこの設計にしたか

- Phase 1 の目的が「既存・新規取り込みで共通利用する純粋関数の追加」だったため、I/O や `context.Context` を持たない domain 関数として切り出した
- domain 層に置くことで、handler や usecase から同じ正規化・検証ルールを再利用しやすくし、Phase 2 で既存フローへ安全に適用しやすくした
- エラーは既存規約に合わせて `domain.ErrInvalidInput` を返し、詳細文字列を増やさないことで外部への情報露出を避けた
- 正規化の順序を固定し、制御文字除去、双方向制御文字除去、ゼロ幅文字除去、NFC 正規化、URL 除去のように役割ごとに分離することで、テストしやすく予測可能な処理にした

## 他の選択肢と比較してなぜこれを選んだか

- middleware で正規化する案は、HTTP 経由以外の取り込み経路や将来のバッチ処理と共有しづらいため採用しなかった
- usecase に直接埋め込む案は、Kindle 取り込みと mobile share 取り込みでロジックが再分散しやすく、純粋関数としての単体テストもしづらいため採用しなかった
- repository 入力直前で整形する案は、永続化層に入力ルールを持ち込むことになり、責務分離が崩れるため採用しなかった
- URL 妥当性エラーに個別メッセージを持たせる案も考えられるが、今回は既存の sentinel error 規約と「詳細メッセージを外部に漏らさない」要件を優先して単一エラー種別に揃えた
