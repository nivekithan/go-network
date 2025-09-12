FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY problems/means-to-end/ ./problems/means-to-end/
RUN cd problems/means-to-end && go build -o means-to-end main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/problems/means-to-end/means-to-end .

EXPOSE 8000

CMD ["./means-to-end"]