FROM golang:1.24.4-alpine AS builder
WORKDIR /app
COPY dhcp/go.mod .
COPY . .

RUN go build -o /bin/app

FROM  golang:1.24.4-alpine
COPY --from=builder /bin/app /app
RUN apk add --no-cache go gcc musl-dev linux-headers bash tcpdump net-tools
RUN apk add tcpdump
EXPOSE 67/udp
ENTRYPOINT ["/app"]