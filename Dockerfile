FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o gateway ./cmd/gateway

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/gateway /gateway
RUN adduser -D -u 10001 app
USER app

EXPOSE 9090
ENTRYPOINT ["/gateway"]
