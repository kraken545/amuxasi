# Multi-stage build
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git tmux
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o amuxasi-web ./cmd/web

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache tmux git ca-certificates

# Create amuxasi user
RUN adduser -D -h /home/amuxasi amuxasi
USER amuxasi
WORKDIR /home/amuxasi

COPY --from=builder /build/amuxasi-web /usr/local/bin/amuxasi-web
COPY --from=builder /build/web/static /home/amuxasi/web/static

EXPOSE 7000

ENV AMUXASI_PORT=7000
ENV AMUXASI_WORKSPACE=/workspace

VOLUME ["/workspace", "/home/amuxasi/.config/amuxasi", "/home/amuxasi/.cache/amuxasi"]

ENTRYPOINT ["amuxasi-web"]
