#!/bin/sh

URL="${BASE_URL}/v1/declarations"

curl \
    $CURL_OPTS \
    -u "kmfddm:$API_KEY" \
    -X PUT \
    -T "$1" \
    -w "Response HTTP Code: %{http_code}\n" \
    "$URL"
