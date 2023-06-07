#!/bin/sh

URL="${BASE_URL}/v1/declarations/$1"

curl \
    $CURL_OPTS \
    -X DELETE \
    -u "kmfddm:$API_KEY" \
    -w "Response HTTP Code: %{http_code}\n" \
    "$URL"
