#!/bin/sh

URL="${BASE_URL}/declaration-items"

curl \
    $CURL_OPTS \
    -H "X-Enrollment-ID: $1" \
    "$URL"
