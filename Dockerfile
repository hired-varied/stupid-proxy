FROM golang:1.21 as builder


WORKDIR /app

COPY . ./
RUN go mod download


RUN CGO_ENABLED=0 GOOS=linux go build -o stupid-proxy-server ./server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o stupid-proxy-client ./client/main.go


FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*


COPY --from=builder /app/stupid-proxy-* /app/

CMD ["/app/stupid-proxy-server"]
