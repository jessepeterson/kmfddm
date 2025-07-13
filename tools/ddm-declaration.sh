#!/bin/sh

URL="${BASE_URL}/declaration/$2"

if [ "x$API_USER" = "x" ]; then
    API_USER="kmfddm"
fi

curl \
    $CURL_OPTS \
    -H "X-Enrollment-ID: $1" \
    -u "$API_USER:$API_KEY" \
    "$URL"
