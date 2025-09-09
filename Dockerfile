# Build stage
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN go build -o main ./src

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8000

CMD ["./main"]