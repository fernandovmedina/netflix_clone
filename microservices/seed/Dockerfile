FROM golang:1.25-alpine AS builder
WORKDIR /src/database/seed
COPY microservices/shared/ /src/microservices/shared/
COPY database/seed/ ./
RUN go mod download
RUN mkdir -p /out
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed .
FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app \
    && mkdir -p /media \
    && chown app:app /media
COPY --from=builder /out/seed /usr/local/bin/seed
COPY seed/ /app/seed/
USER app
ENTRYPOINT ["seed"]
