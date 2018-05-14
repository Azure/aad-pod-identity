FROM k8s-gcrio.azureedge.net/hyperkube-amd64:v1.7.7

ARG WD=.
ENV WD ${WD}
WORKDIR $WD

#RUN apk --no-cache add \
   # ca-certificates \
   # iptables \
   # bash

# install, remove cached list
RUN apt-get update && \
    apt-get install -y ca-certificates iptables bash && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir "/lib/nmi/"
COPY ./nmi /lib/nmi/nmi

WORKDIR /lib/nmi/
CMD ["/lib/nmi/nmi", "--logtostderr=1"]