FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o emailrss ./cmd/emailrss

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/emailrss .
COPY --from=builder /app/config.example.yaml ./config.example.yaml

RUN mkdir -p /data/feeds /data/db

VOLUME ["/data"]

EXPOSE 8080

CMD ["./emailrss", "serve", "-c", "/data/config.yaml"]