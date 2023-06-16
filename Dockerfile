ARG BUILDPLATFORM="linux/amd64"
ARG BUILDERIMAGE="golang:1.19"
ARG BASEIMAGE=gcr.io/distroless/static:nonroot

FROM --platform=$BUILDPLATFORM $BUILDERIMAGE as builder

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /go/src/github.com/Azure/aad-pod-identity
ADD . .
RUN go mod download
ARG IMAGE_VERSION
RUN export GOOS=$TARGETOS && \
    export GOARCH=$TARGETARCH && \
    export GOARM=$(echo ${TARGETPLATFORM} | cut -d / -f3 | tr -d 'v') && \
    make build

FROM registry.k8s.io/build-image/debian-iptables:bullseye-v1.5.7 AS nmi
RUN apt update && \
    apt upgrade -y && \
    clean-install ca-certificates
COPY --from=builder /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/nmi /bin/
RUN useradd -u 10001 nonroot
USER nonroot
ENTRYPOINT ["nmi"]

FROM $BASEIMAGE AS mic
COPY --from=builder /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/mic /bin/
ENTRYPOINT ["mic"]

FROM $BASEIMAGE AS demo
COPY --from=builder /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/demo /bin/
ENTRYPOINT ["demo"]

FROM $BASEIMAGE AS identityvalidator
COPY --from=builder /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/identityvalidator /bin/
ENTRYPOINT ["identityvalidator"]
