FROM golang:1.11 AS build
RUN go get github.com/golang/dep/cmd/dep
WORKDIR /go/src/github.com/Azure/aad-pod-identity
COPY Gopkg.toml ./
COPY Gopkg.lock ./
RUN dep ensure --vendor-only
COPY . ./
ARG NMI_VERSION=0.0.0-dev
ARG MIC_VERSION=0.0.0-dev
ARG DEMO_VERSION=0.0.0-dev
ARG IDENTITY_VALIDATOR_VERSION=0.0.0-dev
RUN make build

FROM alpine:3.8 AS base
RUN apk add --no-cache \
    ca-certificates \
    iptables \
    && update-ca-certificates

FROM base AS nmi
COPY --from=build /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/nmi /bin/

FROM base AS mic
COPY --from=build /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/mic /bin/

FROM base AS demo
COPY --from=build /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/demo /bin/

FROM base AS identityvalidator
COPY --from=build /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/identityvalidator /bin/
