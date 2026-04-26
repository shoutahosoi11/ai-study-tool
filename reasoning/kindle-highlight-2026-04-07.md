# KindleハイライトAPI 設計根拠

## book_idをNULL許可にした理由
Kindleハイライトインポート時はASINや書名でハイライトを作成するが、
システム内のbooksテーブルにまだ対応するレコードがない場合がある。
NULL許可にすることでbookレコードなしにハイライトを保存できる。

## sourceカラムを追加した理由
"kindle"と"manual"でハイライトの出処を区別する。
将来的にKindle API連携でインポートした場合とユーザーが手動入力した場合で
フィルタリングや表示を変えられるようにする。

## domain.ErrNotFoundを定義した理由
database/sql.ErrNoRowsはinfrastructure層の詳細。
usecaseやhandlerがsqlパッケージに依存しないよう、
domain層でErrNotFoundを定義してinfrastructureで変換する。
handlerでerrors.Is(err, domain.ErrNotFound)で統一的にハンドリングできる。

## CountHighlightsByUserIDを別クエリにした理由
LISTとCOUNTを分離することで、ページネーション情報（Total）を返せる。
COUNT(*)をサブクエリで同時実行する方法もあるが、
sqlcのシンプルさを維持するため別クエリを採用。

## sqlc/database/sqlの使い分け
sqlc generateが実行できる環境ではsqlcを使う。
できない場合はrepository層のみdatabase/sql直書きに切り替える。
混在させない。

## ページネーション設計
page/limitパラメータをhandlerで受け取り、
usecaseでoffsetに変換（offset = (page-1) * limit）。
limit上限は50、デフォルトは20。
