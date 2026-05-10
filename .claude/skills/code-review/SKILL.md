---
name: code-review
description: Use when reviewing a pull request, local diff, changed files, or existing code for bugs, regressions, security issues, data consistency problems, architecture violations, missing tests, or API contract mismatches.
---

# Code Review Skill

Codex / Claude 共通のコードレビュー専用手順。レビュー依頼では新規実装しない。

## Workflow

1. `AGENTS.md` と対象領域の既存実装・テストを読む。
2. 変更目的、呼び出し元、依存先、DBスキーマ、APIクライアントを確認する。
3. Critical / Major / Minor の順に、具体的なファイルパスと行番号で指摘する。
4. 推測だけの指摘、好みのスタイル指摘、将来不安だけの指摘は避ける。

## Priority

- Critical: データ破壊、課金爆発、認証認可バイパス、秘密情報漏洩、productionで高確率に落ちる問題。
- Major: API契約違反、retry/idempotency/concurrency欠陥、内部エラー漏洩、重要な異常系テスト不足。
- Minor: 低リスクな保守性問題、命名不整合、軽微なバリデーション漏れ。

## Output

### Summary

- 変更の目的と全体リスクを短く要約する。

### Good

- 妥当な設計、既存規約に沿っている点、良いテストを書く。

### Needs Improvement

- 重要度順、ファイル別に書く。
- 各項目に影響と修正案を書く。

### Questions

- 要件次第で判断が変わる点を書く。
