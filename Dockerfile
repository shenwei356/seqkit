ARG GO_VERSION=1.25.5

FROM golang:${GO_VERSION}-bookworm AS builder

ARG GOPROXY=https://proxy.golang.org,direct

WORKDIR /src

COPY go.mod go.sum ./
RUN GOPROXY="${GOPROXY}" go mod download

COPY . .
RUN CGO_ENABLED=0 GOPROXY="${GOPROXY}" go build \
    -tags netgo \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/seqkit \
    ./seqkit

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/seqkit /usr/local/bin/seqkit

WORKDIR /data

ENTRYPOINT ["/usr/local/bin/seqkit"]
