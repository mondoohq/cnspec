#!/bin/bash

# MIT License

# Copyright (c) 2019 GitHub Inc.

# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

set -e

EXE="$1"

if [ -z "$CERT_FILE" ]; then
  echo "skipping Windows code-signing; CERT_FILE not set" >&2
  exit 0
fi

if [ ! -f "$CERT_FILE" ]; then
  echo "error Windows code-signing; file '$CERT_FILE' not found" >&2
  exit 1
fi

if [ -z "$CERT_PASSWORD" ]; then
  echo "error Windows code-signing; no value for CERT_PASSWORD" >&2
  exit 1
fi

osslsigncode sign -n "Mondoo cnspec" -t http://timestamp.digicert.com \
  -pkcs12 "$CERT_FILE" -readpass <(printf "%s" "$CERT_PASSWORD") -h sha256 \
  -in "$EXE" -out "$EXE"~

mv "$EXE"~ "$EXE"
