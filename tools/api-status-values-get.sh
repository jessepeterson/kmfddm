#!/bin/sh

URL="${BASE_URL}/v1/status-values/$1"

curl \
    $CURL_OPTS \
    -u kmfddm:$API_KEY \
    -G \
    --data-urlencode "prefix=$2" \
    "$URL"
