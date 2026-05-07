# This Dockerfile is for deploying.
# Since fly.io needs a single docker image to deploy the app,
# We need to write it containing frontend, backend, and environment.
# syntax=docker/dockerfile:1

# Build Stage: build frontend and backend.
FROM node:22-alpine AS frontend
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o /server ./cmd

# Run Stage: set frontend/backend and run the backend.
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /server ./server
COPY --from=frontend /frontend/dist ./web/dist
ENV WEB_DIST_DIR=/app/web/dist
ENV PORT=8080
EXPOSE 8080
CMD ["./server"]
