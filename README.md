# AI Study Tool

AI Study Tool is an AI-powered learning platform for study workflows, question generation, note capture, and social learning.

## Features

- AIによる問題生成（ノート・ハイライトから）
- 問題解答と採点（選択問題・記述問題）
- 学習タイムライン（推薦アルゴリズム）
- ソーシャル機能（フォロー・いいね・リポスト・コメント）
- ノート・ハイライト管理

## Tech Stack

- Go + Echo
- PostgreSQL + sqlc
- React + TypeScript + Tailwind CSS + Vite
- Firebase Auth
- AWS S3
- Gemini API

## Architecture
The project follows Clean Architecture
- backend/cmd: application entrypoint  
- backend/internal/domain: core business entities and rules  
- backend/internal/usecase: application logic  
- backend/internal/repository: sqlc-based persistence layer  
- backend/internal/handler: HTTP handlers  
- backend/internal/middleware: middleware  
- frontend/src: client application 

## Getting Started

### Prerequisites

- Docker
- Node.js
- Go

### Setup

1. backend/.env.example をコピーして backend/.env を作成
2. 必要な環境変数を設定
3. 以下を実行

```bash
docker compose up --build
```

### Access
- Frontend: http://localhost:3000  
- Backend: http://localhost:8080/health  


