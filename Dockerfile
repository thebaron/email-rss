FROM --platform=$BUILDPLATFORM  golang:1.24-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOARCH=amd64 GOOS=linux go build -a -o emailrss ./cmd/emailrss

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/emailrss .
COPY --from=builder /app/config.example.yaml ./config.example.yaml

RUN mkdir -p /data/feeds /data/db

VOLUME ["/data"]

EXPOSE 8080

CMD ["./emailrss", "serve", "-c", "/data/config.yaml"]