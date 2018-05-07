FROM alpine:3.6

RUN apk --no-cache add \
    ca-certificates \
    iptables

ADD build/bin/aad-pod-identity/nmi /bin/nmi

ENTRYPOINT ["nmi"]