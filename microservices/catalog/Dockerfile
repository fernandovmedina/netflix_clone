FROM golang:1.25-alpine AS builder
WORKDIR /workspace/catalog
COPY go.mod go.sum ./
COPY --from=shared . /workspace/shared
RUN go mod download
COPY . .
RUN mkdir -p /out
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/catalog-service .
FROM alpine:3.22
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app \
    && mkdir -p /media \
    && chown app:app /media
COPY --from=builder /out/catalog-service /usr/local/bin/catalog-service
USER app
EXPOSE 8080
ENTRYPOINT ["catalog-service"]
