ARG BUILDPLATFORM="linux/amd64"
ARG BUILDERIMAGE="golang:1.16"
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

FROM k8s.gcr.io/build-image/debian-iptables:buster-v1.6.3 AS nmi
# upgrading libgcrypt20 due to CVE-2021-33560
# upgrading libgnutls30 due to CVE-2021-20231, CVE-2021-20232, CVE-2020-24659
# upgrading libhogweed4 due to CVE-2021-20305, CVE-2021-3580
# upgrading libnettle6 due to CVE-2021-20305, CVE-2021-3580
RUN clean-install ca-certificates libgcrypt20 libgnutls30 libhogweed4 libnettle6
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
