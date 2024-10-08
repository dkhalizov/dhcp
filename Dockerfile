FROM golang:1.23.2-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY . .

RUN go build -o /bin/app

FROM  golang:1.23.2-alpine
COPY --from=builder /bin/app /app
RUN apk add --no-cache go gcc musl-dev linux-headers bash tcpdump net-tools

EXPOSE 67/udp
ENTRYPOINT ["/app"]