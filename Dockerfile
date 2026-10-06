FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/main.go

FROM alpine:3.16

WORKDIR /app

COPY --from=builder /app/main .

COPY --from=builder /app/api/casbin/model.conf ./api/casbin/
COPY --from=builder /app/api/casbin/policy.csv ./api/casbin/

RUN chmod +x ./main

CMD ["./main"]
