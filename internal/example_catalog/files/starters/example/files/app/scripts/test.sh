#!/bin/sh
set -e
for test in tests/*.sh; do
	sh "$test"
done
