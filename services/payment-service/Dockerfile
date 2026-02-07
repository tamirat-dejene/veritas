FROM golang:1.25.1-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o main .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
	&& adduser -D -u 10001 appuser \
	&& mkdir -p /app/shared \
	&& chown -R appuser:appuser /app/shared

WORKDIR /home/appuser

COPY --from=builder /app/main ./main

USER appuser

EXPOSE 8080

CMD ["./main"]
