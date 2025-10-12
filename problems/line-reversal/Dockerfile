FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN cd problems/line-reversal && go build -o line-reversal .

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/problems/line-reversal/line-reversal .

EXPOSE 8000

CMD ["./line-reversal"]
