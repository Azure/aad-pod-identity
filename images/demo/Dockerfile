FROM k8s-gcrio.azureedge.net/hyperkube-amd64:v1.7.7

ARG WD=.
ENV WD ${WD}
WORKDIR $WD

# install, remove cached list
RUN apt-get update && \
    apt-get install -y bash && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir "/lib/demo/"
COPY ./demo /lib/demo/demo

WORKDIR /lib/demo/
CMD ["/lib/demo/demo"]