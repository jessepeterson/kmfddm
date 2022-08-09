#!/bin/sh

URL="${BASE_URL}/v1/declarations"

curl \
    $CURL_OPTS \
    -u kmfddm:$API_KEY \
    "$URL"
