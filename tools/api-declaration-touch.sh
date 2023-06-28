#!/bin/sh

URL="${BASE_URL}/v1/declarations/$1/touch"

curl \
    $CURL_OPTS \
    -X POST \
    -u kmfddm:$API_KEY \
    -w "HTTP %{http_code}\n" \
    "$URL"
