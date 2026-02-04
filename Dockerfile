FROM golang:1.23-alpine AS builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

WORKDIR /app
COPY . .

RUN rm -rf go.sum && \
    make clean && \
    make build

FROM alpine:3.23

RUN apk add --no-cache \
    bash \
    jq \
    curl \
    wget \
    expect \
    ca-certificates \
    bind-tools

COPY --from=builder /app/build/nxqd /usr/local/bin/

# Copy production scripts
COPY scripts/seed_node_prod.sh /usr/local/bin/
COPY scripts/multi_seed_node_prod.sh /usr/local/bin/
COPY scripts/peer_node_prod.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/*.sh

# Create entrypoint
COPY scripts/docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

EXPOSE 26656 26657 8545 8546 9090

ENTRYPOINT ["docker-entrypoint.sh"]
