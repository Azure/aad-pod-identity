FROM alpine:3.8

# install without cache
RUN apk update && apk add --no-cache \
    ca-certificates \
    iptables \
    && update-ca-certificates

ADD ./nmi /bin/nmi

ENTRYPOINT ["nmi"]