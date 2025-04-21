#!/bin/sh

set -e

PLACEHOLDER="__MY_NEXT_APP_RPC_PLACEHOLDER__"

RUNTIME_ENV_VAR="RUNTIME_RPC_ADDRESS"

DEFAULT_RPC_ADDRESS="http://127.0.0.1:26657" 

TARGET_DIR="/app/.next/static"

TARGET_VALUE=$(printenv "$RUNTIME_ENV_VAR")

if [ -z "$TARGET_VALUE" ]; then
  echo "INFO: Env '$RUNTIME_ENV_VAR' do not set. Using default: '$DEFAULT_RPC_ADDRESS'"
  TARGET_VALUE="$DEFAULT_RPC_ADDRESS"
else
  echo "INFO: Env found '$RUNTIME_ENV_VAR': '$TARGET_VALUE'"
fi

if [ ! -d "$TARGET_DIR" ]; then
    echo "Err: Dest Folder '$TARGET_DIR' not found. Please check your build."
else
    ESCAPED_TARGET_VALUE=$(echo "$TARGET_VALUE" | sed 's/[\/&]/\\&/g') 
    find "$TARGET_DIR" -type f -name '*.js' -print0 | xargs -0 sed -i "s/$PLACEHOLDER/$ESCAPED_TARGET_VALUE/g"

    echo "INFO: Done."
fi


echo "INFO: Khởi chạy lệnh gốc: $@"

exec "$@"