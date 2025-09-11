FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY problems/prime-time/main.go ./
RUN go build -o prime-time main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/prime-time .

EXPOSE 8000

CMD ["./prime-time"]