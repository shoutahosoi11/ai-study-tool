# モバイルアプリ

iOS / Android の共有シートから Kindle ハイライトを取り込むための React Native / Expo アプリです。

## この scaffold で動くこと

- Firebase のメールアドレス / パスワードログイン。
- `/api/users/signup` による backend ユーザー初期化。
- 共有テキストの下書きフォーム。
- `/api/highlights/share` への保存。
- テキストと URL 共有に対応した Expo share-intent plugin 設定。

## セットアップ

1. `.env.example` を `.env` にコピーする。
2. Firebase 設定と `EXPO_PUBLIC_API_BASE_URL` を埋める。
3. 依存関係をインストールする。

```bash
npm install
```

4. postinstall patch を適用する。

```bash
npm run postinstall
```

5. native project を生成し、custom dev client で起動する。

```bash
npm run prebuild
npm run ios
npm run android
```

## 重要な注意

- share intent は Expo Go では動きません。
- `ios/` と `android/` は生成物で、git では無視します。
- mobile app は backend migration `028_add_highlights_mobile_share_metadata.sql` が適用済みであることを前提にしています。
- API request は呼び出し直前に Firebase Auth `getIdToken()` を呼び、`Authorization: Bearer <ID token>` として送信します。アプリ自身は Firebase refresh token を保存しません。
- `EXPO_PUBLIC_APP_VERSION` が設定されている場合、API request は `X-Firebase-AppCheck`、`X-Platform`、`X-App-Version` を送信します。アプリは fake value を default にせず、未設定なら `X-App-Version` を送らない方針です。本番 backend は missing version を拒否できます。この header は Firebase ID Token auth と App Check と組み合わせて初めて意味があります。
- biometric unlock はローカルアプリロックとしてのみ扱います。biometric 成功 header をサーバー認可として送らないでください。機密操作の backend 認可は Firebase `auth_time` に依存します。
