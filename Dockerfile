# ---------- Build stage ----------
FROM golang:1.25 AS builder

WORKDIR /build

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o containerlab-api ./cmd/api

# ---------- Runtime stage ----------
FROM alpine:3.22

RUN apk update && \
    apk upgrade && \
    apk add --no-cache wget

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/containerlab-api .

RUN chown appuser:appgroup /app/containerlab-api

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./containerlab-api"]
