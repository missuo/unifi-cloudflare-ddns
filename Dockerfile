FROM golang:1.23 AS builder
WORKDIR /go/src/github.com/missuo/unifi-cloudflare-ddns
COPY go.mod ./
COPY go.sum ./
COPY ddns.go ./
RUN go get -d -v ./
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o unifi-cloudflare-ddns .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /go/src/github.com/missuo/unifi-cloudflare-ddns/unifi-cloudflare-ddns /app/unifi-cloudflare-ddns
CMD ["/app/unifi-cloudflare-ddns"]