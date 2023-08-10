#!/bin/sh

URL="${BASE_URL}/v1/status-report/$1?index=0"

curl \
    $CURL_OPTS \
    -u kmfddm:$API_KEY \
    "$URL"
