# Multi-stage build — kichik va xavfsiz image.
# Builder: golang:alpine — full toolchain
# Runtime: alpine — minimal (~10MB)

# ---- Builder ----
FROM golang:1.22-alpine AS builder

WORKDIR /build

# go.sum bor → checksum o'sha fayldan tekshiriladi, sum.golang.org kerak emas.
ENV GOSUMDB=off

# Dependency cache qatlami: botmodule-go SDK GitHub'dan yuklanadi (public modul).
COPY go.mod go.sum ./
RUN go mod download

# Manba kod.
COPY . .

# CGO o'chiq → statik binary.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /module .

# ---- Runtime ----
FROM alpine:3.20

# Non-root foydalanuvchi (xavfsizlik).
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /module ./module

# CA sertifikatlari (HTTPS so'rovlar uchun — ixtiyoriy, kerak bo'lsa saqlang).
RUN apk --no-cache add ca-certificates

USER appuser

ENV PORT=8100
EXPOSE 8100

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8100/health || exit 1

CMD ["/app/module"]
