FROM golang:1.25-alpine AS builder
WORKDIR /workspace/auth
COPY go.mod go.sum ./
COPY --from=shared . /workspace/shared
RUN go mod download
COPY . .
RUN mkdir -p /out
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth-service .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app
COPY --from=builder /out/auth-service /usr/local/bin/auth-service
USER app
EXPOSE 8080
ENTRYPOINT ["auth-service"]
