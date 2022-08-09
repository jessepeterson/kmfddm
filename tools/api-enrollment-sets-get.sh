#!/bin/sh

URL="${BASE_URL}/v1/enrollment-sets/$1"

curl \
    $CURL_OPTS \
    -u kmfddm:$API_KEY \
    "$URL"
