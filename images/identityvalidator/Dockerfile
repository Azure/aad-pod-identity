FROM alpine

RUN apk add --no-cache bash && \
    apk add --no-cache curl

ADD ./identityvalidator ./bin/identityvalidator

# kubectl exec <identity validator pod name> -- identityvalidator
#                                            --subscription-id "$SUBSCRIPTION_ID" \
#                                            --resource-group "$RESOURCE_GROUP" \
#                                            --identity-client-id "$AZURE_CLIENT_ID" \
#                                            --keyvault-name "$KEYVAULT_NAME" \
#                                            --keyvault-secret-name "$KEYVAULT_SECRET_NAME" \
#                                            --keyvault-secret-version "$KEYVAULT_SECRET_VERSION"
ENTRYPOINT [ "/bin/bash", "-c" ]