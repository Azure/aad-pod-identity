#!/bin/sh

echo "Hello ${NAME}"

i=0
while true
do
    echo "Iteration $i"

    curl http://bing.com
    echo '{"a" : 3, "b" : "john"}' | jq '.a'
    i=$((i+1))
    sleep 1
done