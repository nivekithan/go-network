FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN cd problems/speed-daemon && go build -o speed-daemon .

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/problems/speed-daemon/speed-daemon .

EXPOSE 8000

CMD ["./speed-daemon"]