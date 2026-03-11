FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.Version=3.0.0 -s -w" -o /sentinel ./cmd/sentinel/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /sentinel /usr/local/bin/sentinel
EXPOSE 8080
ENTRYPOINT ["sentinel"]
