#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR

go run ../main.go -manifest test.MF \
  -location "https://packages.example.com/test.pack" \
  -describedby "index.html" \
  -signkey $DIR/keys/key_enc.pem \
#  -index
