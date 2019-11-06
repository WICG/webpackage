#!/bin/sh
set -e
gen-bundle -baseURL https://example.com/ \
           -primaryURL https://example.com/hello.html \
           -dir hello \
           -o hello.wbn
