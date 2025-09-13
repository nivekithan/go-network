FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY problems/budget-chat/ ./problems/budget-chat/
RUN cd problems/budget-chat && go build -o budget-chat .

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/problems/budget-chat/budget-chat .

EXPOSE 8000

CMD ["./budget-chat"]