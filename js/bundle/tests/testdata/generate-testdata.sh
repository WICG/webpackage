#!/bin/sh
set -e
gen-bundle -baseURL https://example.com/ \
           -dir hello \
           -version b2 \
           -o hello.wbn
