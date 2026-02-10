# Build frontend
FROM node:20-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web ./
RUN npm run build

# Build backend
FROM golang:1.22-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/server

# Final image
FROM alpine:3.19
WORKDIR /app
COPY --from=go-builder /app/bin/server ./server
COPY --from=web-builder /app/web/dist ./web/dist
EXPOSE 8080
ENV ADDR=:8080
CMD ["./server"]
