#!/bin/sh

URL="${BASE_URL}/tokens"

curl \
    $CURL_OPTS \
    -H "X-Enrollment-ID: $1" \
    "$URL"
