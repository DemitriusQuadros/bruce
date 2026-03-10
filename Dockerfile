FROM golang:1.22 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGET

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/app ./cmd/$TARGET

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /bin/app .

CMD ["./app"]