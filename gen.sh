#!/bin/sh
set -e
base="$(dirname $0)"
typ="$1"
[ "$typ" = "" ] && typ="interface{}"

echo "[go.oneofone.dev/radix] generating typed version using '${typ}' as value type."

perl -pe 's@go1\.18@!go1.18@g;s@\[VT.*?]@@g;s@VT@'${typ}'@g;s@^//go:gen.*$@@g' "${base}/radix.go" > radix_go117.go
perl -pe 's@go1\.18@!go1.18@g;s@\[VT.*?]@@g;s@VT@'${typ}'@g;s@^//go:gen.*$@@g' "${base}/safe.go" > safe_go117.go
perl -pe 's@go1\.18@!go1.18@g;s@\[VT.*?]@@g;s@(New|Tree)\[.+?\]@\1@g;s@^//go:gen.*$@@g' "${base}/radix_test.go" > radix_go117_test.go
gopls format -w radix_go117.go
