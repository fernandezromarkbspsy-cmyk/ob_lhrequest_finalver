# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/server ./cmd/server
RUN go build -o /out/frontend ./cmd/frontend

FROM alpine:3.22 AS backend
WORKDIR /app
RUN adduser -D -H appuser
COPY --from=build /out/server /app/server
USER appuser
EXPOSE 8080
CMD ["/app/server"]

FROM alpine:3.22 AS frontend
WORKDIR /app
RUN adduser -D -H appuser
COPY --from=build /out/frontend /app/frontend-server
COPY frontend /app/frontend
USER appuser
ENV FRONTEND_HOST=0.0.0.0
ENV FRONTEND_PORT=5173
ENV FRONTEND_DIR=/app/frontend
EXPOSE 5173
CMD ["/app/frontend-server"]
