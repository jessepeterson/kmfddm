#!/bin/sh

URL="${API_BASE_URL}/status-values/$1"

if [ "x$API_USER" = "x" ]; then
    API_USER="kmfddm"
fi

curl \
    $CURL_OPTS \
    -u "$API_USER:$API_KEY" \
    -G \
    --data-urlencode "prefix=$2" \
    "$URL"
