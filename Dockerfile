# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/server ./cmd/server
RUN go build -o /out/frontend ./cmd/frontend

FROM node:22-alpine AS frontend-build
WORKDIR /app/frontend-react
COPY frontend-react/package*.json ./
RUN npm ci
COPY frontend-react ./
RUN npm run build

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
COPY --from=frontend-build /app/frontend-react/dist /app/frontend
USER appuser
ENV FRONTEND_HOST=0.0.0.0
ENV FRONTEND_PORT=5173
ENV FRONTEND_DIR=/app/frontend
ENV FRONTEND_API_URL=http://127.0.0.1:8080
EXPOSE 5173
CMD ["/app/frontend-server"]
