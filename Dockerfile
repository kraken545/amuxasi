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

# Create amuxasi user and pre-create data directories
RUN adduser -D -h /home/amuxasi amuxasi \
 && mkdir -p /home/amuxasi/.config/amuxasi \
 && mkdir -p /home/amuxasi/.cache/amuxasi/logs \
 && chown -R amuxasi:amuxasi /home/amuxasi/.config \
 && chown -R amuxasi:amuxasi /home/amuxasi/.cache

USER amuxasi
WORKDIR /home/amuxasi

COPY --from=builder /build/amuxasi-web /usr/local/bin/amuxasi-web

EXPOSE 7000

ENV AMUXASI_PORT=7000
ENV AMUXASI_WORKSPACE=/workspace

VOLUME ["/workspace", "/home/amuxasi/.config/amuxasi", "/home/amuxasi/.cache/amuxasi"]

ENTRYPOINT ["amuxasi-web"]
