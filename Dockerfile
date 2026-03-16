FROM golang:1.20.5-alpine3.18 AS build-env

WORKDIR /go/src/github.com/evmos/evmos

COPY go.mod ./

RUN set -eux; apk add --no-cache ca-certificates=20230506-r0 build-base=0.5-r3 git linux-headers=6.3-r0

RUN --mount=type=secret,id=github_token \
    git config --global url."https://x-access-token:$(cat /run/secrets/github_token)@github.com/".insteadOf "https://github.com/" && \
    go mod download

COPY . .

RUN rm -f go.sum

RUN touch go.sum
RUN GONOSUMDB="*" GOFLAGS="-mod=mod" make build

RUN go install github.com/MinseokOh/toml-cli@latest

FROM alpine:3.18

WORKDIR /root

COPY --from=build-env /go/src/github.com/evmos/evmos/build/nxqd /usr/bin/nxqd
COPY --from=build-env /go/bin/toml-cli /usr/bin/toml-cli
COPY --from=build-env /go/src/github.com/evmos/evmos/run_seed.sh /root/run_seed.sh


RUN apk add --no-cache ca-certificates jq curl bash vim lz4 \
    && addgroup -g 1000 evmos \
    && adduser -S -h /home/evmos -D evmos -u 1000 -G evmos

USER 1000
WORKDIR /home/evmos

EXPOSE 26656 26657 1317 9090 8545 8546

CMD ["nxqd"]