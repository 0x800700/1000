# Thousand (Тысяча)

Offline browser card game. Go backend + React/TypeScript + PixiJS frontend.

## Requirements
- Go 1.22+
- Node 20+
- Docker + Docker Compose

## Run with Docker
```bash
docker-compose up --build
```
Open `http://localhost:8080`.

## Run for Development
Backend:
```bash
go run ./cmd/server
```
Frontend (in another terminal):
```bash
cd web
npm install
# or, once lockfile exists:
# npm ci
npm run dev
```
Open `http://localhost:5173` for the Vite dev server.

## Build
Frontend:
```bash
cd web
npm run build
```
Backend:
```bash
go build ./cmd/server
```

## Project Structure
- `cmd/server`: HTTP + WebSocket server
- `internal/engine`: deterministic rules engine (pure Go)
- `internal/bots`: bots (to be implemented)
- `internal/server`: WS protocol + session (to be implemented)
- `web`: React/TypeScript + PixiJS frontend

## Notes
- Current MVP scaffolds the engine deal logic and a placeholder UI.
- Rules are configurable via `ClassicPreset()` in `internal/engine`.
