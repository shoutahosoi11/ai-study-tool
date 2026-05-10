---
name: mobile-debug
description: Use when debugging or running the Expo mobile app on iOS/Android devices or simulators, fixing Metro/dev-client connection issues, configuring mobile .env, testing API connectivity from a phone, or handling iOS signing/provisioning problems.
---

# Mobile Debug Skill

## Preflight

- backend: `curl http://localhost:8080/health`
- 実機では `localhost` ではなくMacのLAN IPを使う。
- mobile `.env` に `EXPO_PUBLIC_API_BASE_URL` と Firebase公開envがあるか確認する。
- iPhoneとMacが同じネットワークか確認する。

## Commands

```bash
cd mobile && npm install
cd mobile && npx expo start --dev-client --lan --port 8081
cd mobile && npm run ios -- --device "<iPhoneの名前>"
```

## Output

- 実行したコマンド。
- 成功/失敗した箇所。
- 次に実機で確認すべき操作。
