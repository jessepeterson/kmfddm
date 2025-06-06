#!/bin/sh

URL="${API_BASE_URL}/enrollment-sets/$1?set=$2"

if [ "x$API_USER" = "x" ]; then
    API_USER="kmfddm"
fi

curl \
    $CURL_OPTS \
    -u "$API_USER:$API_KEY" \
    -X PUT \
    -w "Response HTTP Code: %{http_code}\n" \
    "$URL"
