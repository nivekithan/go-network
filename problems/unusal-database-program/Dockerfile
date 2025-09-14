FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY problems/unusal-database-program/ ./problems/unusal-database-program/
RUN cd problems/unusal-database-program && go build -o unusal-database-program .

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/problems/unusal-database-program/unusal-database-program .

EXPOSE 8000

CMD ["./unusal-database-program"]