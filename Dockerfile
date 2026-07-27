FROM golang:1.22-alpine AS go-builder
WORKDIR /app
COPY main.go .
RUN go mod init nyx-gateway && \
    go get github.com/gorilla/websocket && \
    go build -ldflags="-s -w" -o go-proxy main.go

FROM ghcr.io/sagernet/sing-box:latest
RUN apk add --no-cache ca-certificates bash

WORKDIR /app
COPY --from=go-builder /app/go-proxy /app/go-proxy
COPY config.json /app/config.json
COPY index.html /app/index.html
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

EXPOSE 7860

ENTRYPOINT ["/app/entrypoint.sh"]