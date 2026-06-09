# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates gcc musl-dev

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build gateway service
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /build/gateway ./cmd/gateway/

# ---- Runtime ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata sqlite

WORKDIR /app

COPY --from=builder /build/gateway .
COPY --from=builder /build/config.yaml .
COPY --from=builder /build/templates/ ./templates/

VOLUME ["/app/data"]

EXPOSE 8080

ENTRYPOINT ["/app/gateway"]
CMD ["-config", "/app/config.yaml"]
