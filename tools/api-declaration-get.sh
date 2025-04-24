#!/bin/sh

URL="${BASE_URL}/declarations/$1"

if [ "x$API_USER" = "x" ]; then
    API_USER="kmfddm"
fi

curl \
    $CURL_OPTS \
    -u "$API_USER:$API_KEY" \
    "$URL"
