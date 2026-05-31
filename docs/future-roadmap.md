# Future Roadmap

## Mobile IAP / Play Billing

- 目的: iOS / Androidのnative subscriptionや消耗型商品に対応する。
- 実装概要: App Store Server Notifications、Google Play Developer API、receipt validation、subscription state syncを追加する。
- 優先度: High
- 注意点: Web Stripeとmobile store billingのplan stateを統合し、二重課金やstatus不整合を避ける。

## Kindle Note Support

- 目的: ハイライトだけでなく、Kindle noteも学習素材として扱う。
- 実装概要: Extension parser、import payload、DB schema、question source resolverをnote対応に広げる。
- 優先度: Medium
- 注意点: note本文は機微情報になり得るため、ログやAdmin画面に出さない。

## Better Spaced Repetition

- 目的: 間違えた問題や苦手分野をより効率よく復習する。
- 実装概要: answer履歴、正答率、復習間隔、due dateを使ったschedulerを追加する。
- 優先度: High
- 注意点: 複雑な推薦より、説明可能で調整しやすいルールから始める。

## Admin Analytics

- 目的: 運用者が利用状況、失敗率、コスト傾向を把握しやすくする。
- 実装概要: LLM usage、job latency、import success rate、billing/admob eventの集計APIとグラフを追加する。
- 優先度: Medium
- 注意点: 個人情報やraw payloadを集計画面に出さない。

## User Notifications

- 目的: 問題生成完了、復習タイミング、token不足などをユーザーに伝える。
- 実装概要: Web notification、mobile push、emailのいずれかから段階的に導入する。
- 優先度: Medium
- 注意点: 通知頻度、unsubscribe、個人情報を含まない文面に注意する。

## Multi-Provider LLM Selection

- 目的: 品質、価格、rate limitに応じてLLM providerを切り替えられるようにする。
- 実装概要: provider adapterを拡張し、model policy、fallback、per-provider usage loggingを追加する。
- 優先度: Medium
- 注意点: providerごとのtoken計算とcost estimateの差を吸収する。

## Cost Dashboard

- 目的: LLMとインフラのコストを運用者が早期に把握する。
- 実装概要: global budget、usage log、provider/model別cost、job retry costをAdmin Dashboardに集約する。
- 優先度: High
- 注意点: 推定コストと実請求額の差分を明示する。

## Chrome Web Store Release

- 目的: Browser Extensionを一般配布できる状態にする。
- 実装概要: store screenshots、privacy policy、permission justification、review package、versioningを整備する。
- 優先度: High
- 注意点: host permissionsと取得データの説明を明確にする。

## App Store / Google Play Release

- 目的: Mobile appをstore配布できる状態にする。
- 実装概要: store metadata、privacy labels、App Check production設定、release signing、review assetsを整備する。
- 優先度: Medium
- 注意点: store billing、tracking disclosure、Firebase設定の環境分離が必要。

## Organization / Team Learning Features

- 目的: 個人利用だけでなく、チームや教育機関で学習セットを共有できるようにする。
- 実装概要: organization、role、shared question set、team analytics、invite flowを追加する。
- 優先度: Low
- 注意点: 個人データと共有データの境界、IDOR、権限管理が複雑になる。
