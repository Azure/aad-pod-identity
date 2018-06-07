FROM k8s-gcrio.azureedge.net/hyperkube-amd64:v1.7.7

# install, remove cached list
RUN apt-get update && \
    apt-get install -y bash && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ADD ./demo /bin/demo

ENTRYPOINT ["demo"]