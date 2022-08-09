#!/bin/sh

URL="${BASE_URL}/v1/set-declarations/$1?declaration=$2"

curl \
    $CURL_OPTS \
    -u kmfddm:$API_KEY \
    -X PUT \
    -w "Response HTTP Code: %{http_code}\n" \
    "$URL"
