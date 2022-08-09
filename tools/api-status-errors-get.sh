#!/bin/sh

URL="${BASE_URL}/v1/status-errors/$1"

curl \
    $CURL_OPTS \
    -u kmfddm:$API_KEY \
    "$URL"
