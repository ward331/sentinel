# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.Version=3.0.0 -s -w" \
    -o /sentinel ./cmd/sentinel/

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -h /app sentinel
USER sentinel
WORKDIR /app
COPY --from=builder /sentinel /usr/local/bin/sentinel
EXPOSE 8080
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1
ENTRYPOINT ["sentinel"]
CMD ["--host", "0.0.0.0", "--data-dir", "/app/data"]
