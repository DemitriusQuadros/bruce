FROM golang:1.23-alpine AS builder

# build-base and sqlite-dev are required for CGO (mattn/go-sqlite3).
RUN apk add --no-cache build-base sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build \
  -ldflags="-s -w -extldflags '-static'" \
  -o /bin/bruce ./cmd/bruce

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Persist SQLite databases on a host-mounted volume.
RUN mkdir -p /app/data

COPY --from=builder /bin/bruce .

EXPOSE 8080

ENTRYPOINT ["./bruce"]
