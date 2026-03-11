FROM golang:1.23-alpine AS builder

# gcc and musl-dev are required for CGO (mattn/go-sqlite3).
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /bin/bruce ./cmd/bruce

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Persist SQLite databases on a host-mounted volume.
RUN mkdir -p /app/data

COPY --from=builder /bin/bruce .

CMD ["./bruce"]
