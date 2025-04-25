#!/bin/sh

URL="${API_BASE_URL}/status-report/$1?index=0"

if [ "x$API_USER" = "x" ]; then
    API_USER="kmfddm"
fi

curl \
    $CURL_OPTS \
    -u "$API_USER:$API_KEY" \
    "$URL"
