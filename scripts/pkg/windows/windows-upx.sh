#!/bin/bash

set -e

EXE="$1"

if [ "$TARGET" == "windows_amd64_v1" ]; then
  echo "Compressing binary with upx"
  upx "$EXE"
fi
