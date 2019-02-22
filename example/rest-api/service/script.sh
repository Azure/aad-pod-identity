#!/bin/sh

echo "Hello ${RESOURCE}"

i=0
while true
do
    echo "Iteration $i"

    jwt=$(curl http://169.254.169.254/metadata/identity/oauth2/token/?resource=$RESOURCE)
    token=echo $jwt | jq '.access_token'
    echo "Token:  $token"
    i=$((i+1))
    sleep 1
done