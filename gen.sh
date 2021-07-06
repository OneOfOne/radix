#!/bin/sh
set -e
def="$1"
[ "$def" = "" ] && def="interface{}"
perl -pe 's@go1\.18@!go1.18@g;s@\[VT.*?]@@g;s@VT@'${def}'@g;s@^//go:gen.*$@@g' radix.go > radix_go117.go
perl -pe 's@go1\.18@!go1.18@g;s@\[VT.*?]@@g;s@New\[.+?\]@New@g;s@^//go:gen.*$@@g' radix_test.go > radix_go117_test.go
gopls format -w radix_go117.go
