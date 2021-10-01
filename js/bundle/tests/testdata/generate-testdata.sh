#!/bin/sh
set -e
gen-bundle -baseURL https://example.com/ \
           -dir hello \
           -o hello.wbn
