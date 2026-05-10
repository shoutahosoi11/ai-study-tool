---
name: github-pr
description: Use when preparing, updating, pushing, or checking a GitHub pull request for this repository. Applies to PR creation, CI checks, merge blockers, and failing GitHub Actions.
---

# GitHub PR Skill

## Workflow

1. `git status -sb` と `git diff --stat` で差分を確認する。
2. main直pushはしない。feature branchからmainへPRを作る。
3. ユーザーの未コミット変更を勝手に戻さない。
4. backend変更時は `cd backend && go build ./... && go test ./...` を確認する。
5. commit、push、draft PR作成、PR URL報告まで行う。

## Guardrails

- `git reset --hard` を使わない。
- unrelated diffを混ぜない。
- マージはユーザーが明示した場合だけ行う。
