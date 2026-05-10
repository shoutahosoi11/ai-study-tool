# Router Consolidation

## 何を作ったか

- Backend の API ルート登録を `/api/v1` に統一した。
- 既存の `/api` group と `/api/v1` group の併用をやめ、users / posts / highlights / questions / tokens / checkout を同じ versioned API 配下に集約した。
- `registerSocialRoutes` を削除し、ユーザー操作は `/users/:id/follow`、投稿操作は `/posts/:id/like` など、それぞれの resource route に分散した。
- デッドコードだった `POST /questions/:id/grade` と、重複していた manual generation route 登録を削除した。
- Frontend の API base URL fallback を `/api/v1` に変更した。
- Mobile の API base URL を `/api/v1` 前提にし、既存 `.env` が `/api` 終わりでも `/api/v1` に正規化するようにした。
- Mobile API client の `/v1/...` 直書きを削除し、base URL との責務を分けた。

## なぜこの設計にしたか

- API version を router の入口で統一すると、backend / frontend / mobile の接続先が明確になり、今後 `/api` と `/api/v1` の二重管理が発生しにくい。
- social action は独立 resource ではなく users / posts に対する action なので、`registerSocialRoutes` にまとめるより、該当 resource の route 登録に置いた方が URL と実装の対応が読みやすい。
- Mobile は環境変数運用で実機確認するため、`.env` の既存値が `/api` のままでも動く正規化を入れた。これにより移行時の手戻りを減らしつつ、新しい推奨値は `/api/v1` に揃えられる。
- `/questions/:id/answer` が回答提出の canonical endpoint なので、同じ handler に流す `/questions/:id/grade` は互換維持よりデッドコード削除を優先した。

## 他の選択肢との比較

- `/api` を残して `/api/v1` に alias する案は、短期的な互換性は高いが、混在状態を温存するため今回の目的に合わない。
- `registerSocialRoutes` を名前だけ変えて残す案は、関数名の違和感は解消できるが、users / posts の route 定義が分散したままになる。
- Mobile の各 API path に `/api/v1` を直書きする案は、環境ごとの base URL 管理と重複するため採用しなかった。base URL が API prefix、各 API file が resource path を担当する形にした。
