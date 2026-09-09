FROM golang:alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION

WORKDIR /src

COPY go.mod .
COPY cmd cmd
COPY pkg pkg

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.Version=${VERSION}" -o /tmp/agent/hdbbackint ./cmd/hdbbackint

FROM alpine

ARG TARGETOS
ARG TARGETARCH

LABEL org.opencontainers.image.source="https://github.com/kubedb/hanadb-init"

RUN apk add --no-cache bash ca-certificates

RUN mkdir -p /init-script /tmp/agent

COPY --from=builder /tmp/agent/hdbbackint /tmp/agent/hdbbackint
COPY init-script /init-script

RUN chmod 755 /tmp/agent/hdbbackint /init-script/*.sh

ENTRYPOINT ["/init-script/run.sh"]
