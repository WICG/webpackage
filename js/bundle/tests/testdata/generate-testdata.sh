#!/bin/sh
set -e
gen-bundle -baseURL https://example.com/ \
           -primaryURL https://example.com/hello.html \
           -dir hello \
           -o hello_b1.wbn
gen-bundle -baseURL https://example.com/ \
           -dir hello \
           -version b2 \
           -o hello_b2.wbn
