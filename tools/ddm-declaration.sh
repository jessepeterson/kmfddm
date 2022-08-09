#!/bin/sh

URL="${BASE_URL}/declaration/$2"

curl \
    $CURL_OPTS \
    -H "X-Enrollment-ID: $1" \
    "$URL"
