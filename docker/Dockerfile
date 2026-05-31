# ── Stage 1: Build Go binary ─────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# ── Stage 2: Minimal runtime image ───────────────────────────
FROM alpine:3.21

RUN apk add --no-cache tzdata ca-certificates wget
ENV TZ=Asia/Manila

WORKDIR /app

COPY --from=builder /app/server .
COPY web/truck_label ./web/truck_label

EXPOSE 8080

ENTRYPOINT ["./server"]
