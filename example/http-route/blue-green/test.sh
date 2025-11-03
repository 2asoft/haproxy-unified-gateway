#!/bin/sh

COUNTER=0
while [ $COUNTER -lt 100 ]; do
  curl -s -X GET http://localhost:31080/hostname \
          -H 'Host: blue-green.haproxy.local' | cut -d'-' -f1
  COUNTER=$((COUNTER + 1))
done | sort | uniq -c
