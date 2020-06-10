ARG BASEIMAGE=us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:v2.1.0

FROM golang:1.14 AS build
WORKDIR /go/src/github.com/Azure/aad-pod-identity
ADD . .
RUN go mod download
ARG NMI_VERSION
ARG MIC_VERSION
ARG DEMO_VERSION
ARG IDENTITY_VALIDATOR_VERSION
RUN make build

FROM $BASEIMAGE AS base
RUN clean-install ca-certificates

FROM base AS nmi
COPY --from=build /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/nmi /bin/
ENTRYPOINT ["nmi"]

FROM base AS mic
COPY --from=build /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/mic /bin/
ENTRYPOINT ["mic"]

FROM base AS demo
COPY --from=build /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/demo /bin/
ENTRYPOINT ["demo"]

FROM base AS identityvalidator
COPY --from=build /go/src/github.com/Azure/aad-pod-identity/bin/aad-pod-identity/identityvalidator /bin/
ENTRYPOINT ["identityvalidator"]
