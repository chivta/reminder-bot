# Base stage: shared dependency download
FROM golang:1.26.2-alpine AS base
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

# Source stage: base + full source
FROM base AS source
COPY . .

# Dev stage
FROM base AS dev
RUN go install github.com/air-verse/air@latest
WORKDIR /app
CMD ["air"]

# Build stage
FROM source AS builder
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot ./cmd/bot && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o reminder ./cmd/reminder

# Production stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates && \
    adduser -D -s /sbin/nologin appuser

WORKDIR /home/appuser
COPY --from=builder --chown=appuser:appuser /build/bot .
COPY --from=builder --chown=appuser:appuser /build/reminder .

USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./bot"]
