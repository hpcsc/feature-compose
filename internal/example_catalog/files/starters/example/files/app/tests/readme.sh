#!/bin/sh
if ! grep -q '^# {{ .Name }}$' README.md; then
	echo 'tests/readme.sh: README.md has no heading "# {{ .Name }}"'
	exit 1
fi
