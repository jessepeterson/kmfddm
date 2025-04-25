#!/bin/sh

URL="${API_BASE_URL}/enrollment-sets-all/$1"

if [ "x$API_USER" = "x" ]; then
    API_USER="kmfddm"
fi

curl \
    $CURL_OPTS \
    -u "$API_USER:$API_KEY" \
    -X DELETE \
    -w "Response HTTP Code: %{http_code}\n" \
    "$URL"
