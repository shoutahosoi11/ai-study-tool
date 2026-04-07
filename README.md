# AI Study Tool

AI Study Tool is an AI-powered learning platform for study workflows, question generation, note capture, and social learning.

## Tech Stack

- Go + Echo
- PostgreSQL + sqlc
- React + TypeScript + Tailwind CSS + Vite
- Firebase Auth
- AWS S3
- Gemini API

## Getting Started

1. Copy `backend/.env.example` to `backend/.env` and fill in the required values.
2. Start the development stack with `docker compose up --build`.
3. Access the frontend at `http://localhost:3000` and the backend health check at `http://localhost:8080/health`.

## Architecture

The project follows Clean Architecture:

- `backend/cmd` contains the application entrypoint.
- `backend/internal/domain` contains core business entities and rules.
- `backend/internal/usecase` contains application-specific business logic.
- `backend/internal/repository` contains persistence adapters and sqlc-generated access code.
- `backend/internal/handler` contains HTTP handlers.
- `backend/internal/middleware` contains cross-cutting HTTP middleware.
- `frontend/src` contains the client application organized by components, pages, hooks, theme, types, and API access.
