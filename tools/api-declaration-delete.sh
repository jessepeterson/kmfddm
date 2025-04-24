#!/bin/sh

URL="${API_BASE_URL}/declarations/$1"

if [ "x$API_USER" = "x" ]; then
    API_USER="kmfddm"
fi

curl \
    $CURL_OPTS \
    -X DELETE \
    -u "$API_USER:$API_KEY" \
    -w "Response HTTP Code: %{http_code}\n" \
    "$URL"
