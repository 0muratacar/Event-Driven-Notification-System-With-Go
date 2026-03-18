FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /notifier ./cmd/notifier

FROM gcr.io/distroless/static-debian12

COPY --from=builder /notifier /notifier
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/notifier"]
