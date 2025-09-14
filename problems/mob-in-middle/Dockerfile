FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY problems/mob-in-middle/ ./problems/mob-in-middle/
RUN cd problems/mob-in-middle && go build -o mob-in-middle .

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/problems/mob-in-middle/mob-in-middle .

EXPOSE 8000

CMD ["./mob-in-middle"]