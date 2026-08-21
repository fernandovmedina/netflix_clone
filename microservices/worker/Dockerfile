FROM golang:1.25-alpine AS builder
WORKDIR /workspace/worker
COPY --from=shared . /workspace/shared
COPY . .
RUN mkdir -p /out && if [ -f go.mod ]; then \
      go mod download && \
      CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker-service .; \
    else \
      printf '#!/bin/sh\nexec sleep infinity\n' > /out/worker-service && chmod +x /out/worker-service; \
    fi
FROM alpine:3.22
RUN apk add --no-cache ca-certificates ffmpeg \
    && ffmpeg -version \
    && ffprobe -version \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app \
    && mkdir -p /media \
    && chown app:app /media
COPY --from=builder /out/worker-service /usr/local/bin/worker-service
USER app
CMD ["worker-service"]
