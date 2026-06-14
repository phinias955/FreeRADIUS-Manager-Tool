# ---- Build Stage ----
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o radius-manager ./main.go

# ---- Runtime Stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

COPY --from=builder /app/radius-manager .

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./radius-manager"]
