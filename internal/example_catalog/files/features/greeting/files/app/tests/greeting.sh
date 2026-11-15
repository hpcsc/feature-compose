#!/bin/sh
expected='{{ if eq (option "language") "fr" }}Bonjour{{ else }}Hello{{ end }}, {{ .Name }}!'
actual=$(sh ./scripts/greet)
if [ "$actual" != "$expected" ]; then
	echo "tests/greeting.sh: the greeting is \"$actual\", not \"$expected\""
	exit 1
fi
