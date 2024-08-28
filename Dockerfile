FROM golang:1.22 AS builder
WORKDIR /go/src/github.com/missuo/unifi-cloudflare-ddns
COPY main.go ./
COPY go.mod ./
COPY go.sum ./
COPY types.go ./
RUN go get -d -v ./
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o unifi-cloudflare-ddns .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /go/src/github.com/missuo/unifi-cloudflare-ddns/unifi-cloudflare-ddns /app/unifi-cloudflare-ddns
CMD ["/app/unifi-cloudflare-ddns"]