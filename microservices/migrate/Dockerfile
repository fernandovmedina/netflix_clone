FROM golang:1.25-alpine AS builder
WORKDIR /src/microservices/shared
COPY microservices/shared/ ./
RUN go mod download
RUN mkdir -p /out
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./migrate
FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app
COPY --from=builder /out/migrate /usr/local/bin/migrate
COPY database/migrations/ /app/migrations/
USER app
ENTRYPOINT ["migrate"]
