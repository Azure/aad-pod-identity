FROM alpine

RUN apk add --no-cache bash && \
    apk add --no-cache curl

ADD ./identityvalidator ./bin/identityvalidator

# Use "kubectl exec <pod name> -- identityvalidator --subscription-id=<...> --identity-client-id=<...> --resource-group=<...>" to execute the validator
ENTRYPOINT [ "/bin/bash", "-c" ]