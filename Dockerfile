FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/menu-service ./cmd/server

FROM alpine:3.19
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates tzdata wget
RUN addgroup -g 1000 -S appuser &&     adduser -u 1000 -S appuser -G appuser
WORKDIR /app
COPY --from=builder /out/menu-service ./menu-service
COPY config.*.yaml ./
RUN chown -R appuser:appuser /app
USER appuser
ENV MENU_PORT=8096
ENV MENU_CONFIG_FILE=config.prod
EXPOSE 8096
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3   CMD wget --no-verbose --tries=1 --spider http://localhost:${MENU_PORT}/healthz || exit 1
CMD ["./menu-service", "-config", "config.prod"]
