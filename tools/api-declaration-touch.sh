#!/bin/sh

URL="${API_BASE_URL}/declarations/$1/touch"

if [ "x$API_USER" = "x" ]; then
    API_USER="kmfddm"
fi

curl \
    $CURL_OPTS \
    -X POST \
    -u "$API_USER:$API_KEY" \
    -w "HTTP %{http_code}\n" \
    "$URL"
