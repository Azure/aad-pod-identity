ARG BASEIMAGE=gcr.io/distroless/static:nonroot

FROM golang:1.15 AS builder
WORKDIR /go/src/github.com/Azure/aad-pod-identity
ADD . .
RUN go mod download
ARG IMAGE_VERSION
RUN make build

FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-iptables-amd64:v12.1.2 AS nmi
# upgrading apt &libapt-pkg5.0 due to CVE-2020-27350
# upgrading libssl1.1 due to CVE-2020-1971
# upgrading libp11-kit0 due to CVE-2020-29362, CVE-2020-29363 and CVE-2020-29361
RUN apt-mark unhold apt && \
    clean-install ca-certificates apt libapt-pkg5.0 libssl1.1 libp11-kit0
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
