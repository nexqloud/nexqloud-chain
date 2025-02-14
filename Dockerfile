FROM golang:1.22.5-alpine3.20 AS build-env

ARG DB_BACKEND=goleveldb
ARG ROCKSDB_VERSION="9.2.1"

WORKDIR /go/src/github.com/evmos/evmos

COPY go.mod go.sum ./

RUN set -eux; apk add --no-cache ca-certificates=20240705-r0 build-base=0.5-r3 git linux-headers=6.6-r0 bash=5.2.26-r0

RUN go mod download

COPY . .

RUN mkdir -p /target/usr/lib /target/usr/local/lib /target/usr/include

RUN make build

RUN go install github.com/MinseokOh/toml-cli@latest

FROM alpine:3.20

WORKDIR /root

COPY --from=build-env /go/src/github.com/evmos/evmos/build/nxqd /usr/bin/nxqd
COPY --from=build-env /go/bin/toml-cli /usr/bin/toml-cli

RUN apk add --no-cache jq curl bash vim lz4 rclone

EXPOSE 26656 26657 1317 9090 8545 8546
HEALTHCHECK CMD curl --fail http://localhost:26657 || exit 1

ENV SEED_NODE_IP
ENV MONIKER

COPY peer_node.sh .

RUN sh peer_node.sh init

CMD ["sh", "peer_node.sh"]
