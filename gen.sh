#!/bin/sh
set -e
def="$1"
[ "$def" = "" ] && def="interface{}"
perl -pe 's@go1\.18@!go1.18@g;s@\[VT.*?]@@g;s@VT@'${def}'@g;s@^//go:gen.*$@@g' radix.go > radix_go117.go
gopls format -w radix_go117.go
