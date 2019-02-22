#!/bin/sh

echo "Hello ${NAME}"

i=0
while true
do
    echo "Iteration $i"

    curl  http://169.254.169.254/metadata/identity/oauth2/token/?resource=https://vault.azure.net
    echo '{"a" : 3, "b" : "john"}' | jq '.a'
    i=$((i+1))
    sleep 1
done