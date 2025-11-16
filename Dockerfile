FROM golang:1.24 AS builder

WORKDIR /app

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/trashscanner ./cmd/trashscanner/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder app/bin/trashscanner ./bin/trashscanner
COPY --from=builder app/config ./config
COPY --from=builder app/internal/database/migrations ./migrations
COPY --from=builder app/docs ./docs

CMD ["./bin/trashscanner"]