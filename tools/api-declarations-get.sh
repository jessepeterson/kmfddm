#!/bin/sh

URL="${API_BASE_URL}/declarations"

if [ "x$API_USER" = "x" ]; then
    API_USER="kmfddm"
fi

curl \
    $CURL_OPTS \
    -u "$API_USER:$API_KEY" \
    "$URL"
