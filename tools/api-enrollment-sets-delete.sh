#!/bin/sh

URL="${BASE_URL}/v1/enrollment-sets/$1?set=$2"

curl \
    $CURL_OPTS \
    -u kmfddm:$API_KEY \
    -X DELETE \
    -w "Response HTTP Code: %{http_code}\n" \
    "$URL"
