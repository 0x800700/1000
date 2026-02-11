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

## Run for Development (Docker, hot reload)
```bash
docker-compose -f docker-compose.dev.yml up
```
Backend: `http://localhost:8080`  
Frontend (Vite): `http://localhost:5173`

## Run for Development (local)
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
- Rules are configurable; default preset is `TisyachaPreset()` in `internal/engine`.
- Dev Docker uses bind mounts for fast iteration.
