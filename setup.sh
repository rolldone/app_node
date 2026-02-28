#!/usr/bin/env bash
set -euo pipefail

VOL_CODE="app_node_src"
VOL_KEYDB="app_node_keydb"
VOL_PG="app_node_postgres"

# Resolve current working directory
CWD="$(pwd)"
ABS_CWD="$(pwd -P)"
BASE_DIR="$(dirname "$CWD")"

HOST_VOL_KEYDB_DIR="$BASE_DIR/${VOL_KEYDB}"
HOST_VOL_PG_DIR="$BASE_DIR/${VOL_PG}"

ensure_dir() {
  local d="$1"
  if [ -d "$d" ]; then
    echo "Directory '$d' already exists."
  else
    echo "Creating directory '$d'..."
    mkdir -p "$d"
    echo "Created directory '$d'."
  fi
}

ensure_volume() {
  local name="$1"
  local device
  if [ "$name" = "$VOL_CODE" ]; then
    device="$ABS_CWD"
  elif [ "$name" = "$VOL_KEYDB" ]; then
    device="$HOST_VOL_KEYDB_DIR"
  elif [ "$name" = "$VOL_PG" ]; then
    device="$HOST_VOL_PG_DIR"
  fi
  if docker volume inspect "$name" >/dev/null 2>&1; then
    echo "Volume '$name' already exists."
  else
    echo "Creating volume '$name' with bind to $device..."
    docker volume create --driver local \
      --opt type=none --opt device="$device" --opt o=bind "$name" >/dev/null
    echo "Created volume '$name'."
  fi
}

echo "Ensuring host directories exist..."
ensure_dir "$HOST_VOL_KEYDB_DIR"
ensure_dir "$HOST_VOL_PG_DIR"

echo "Ensuring required Docker volumes exist..."
ensure_volume "$VOL_CODE"
ensure_volume "$VOL_KEYDB"
ensure_volume "$VOL_PG"

echo "All volumes are ready: $VOL_CODE, $VOL_KEYDB, $VOL_PG"
echo "Host directories: $ABS_CWD, $HOST_VOL_KEYDB_DIR, $HOST_VOL_PG_DIR"

