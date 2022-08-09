#!/bin/sh

URL="${BASE_URL}/v1/set-declarations/$1"

curl \
    $CURL_OPTS \
    -u kmfddm:$API_KEY \
    "$URL"
