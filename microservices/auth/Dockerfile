FROM golang:1.25-alpine AS builder

WORKDIR /auth

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /auth-service .

FROM alpine:3.21

COPY --from=builder /auth-service /usr/local/bin/auth-service

EXPOSE 8080

CMD ["auth-service"]
