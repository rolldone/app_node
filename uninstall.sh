#!/usr/bin/env bash
set -euo pipefail

# Names must match `setup.sh` values
VOL_CODE="app_node_src"
VOL_KEYDB="app_node_keydb"

confirm() {
  local prompt="$1"
  if [ "${YES:-false}" = true ]; then
    return 0
  fi
  read -r -p "$prompt [y/N]: " ans
  case "$ans" in
    [Yy]|[Yy][Ee][Ss]) return 0 ;;
    *) return 1 ;;
  esac
}

remove_volume() {
  local v="$1"
  if docker volume inspect "$v" >/dev/null 2>&1; then
    echo "Removing docker volume '$v'..."
    docker volume rm "$v"
    echo "Removed volume '$v'."
  else
    echo "Volume '$v' does not exist, skipping."
  fi
}

remove_dir() {
  local d="$1"
  if [ -d "$d" ]; then
    echo "Removing directory '$d'..."
    rm -rf -- "$d"
    echo "Removed directory '$d'."
  else
    echo "Directory '$d' does not exist, skipping."
  fi
}

usage() {
  cat <<EOF
Usage: $0 [--yes] [--remove-dirs]

Options:
  --yes        Do not prompt for confirmation; remove volumes (and dirs if requested).
  --remove-dirs  Also remove the host directories created by `setup.sh` (opt-in).

This script removes the Docker volumes created by `setup.sh`.
Host directories are NOT removed by default; pass `--remove-dirs` to remove them.
EOF
}

# parse args
YES=false
REMOVE_DIRS=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --yes|-y) YES=true; shift ;;
    --remove-dirs) REMOVE_DIRS=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown arg: $1"; usage; exit 2 ;;
  esac
done

CWD="$(pwd)"
if [ "$(basename "$CWD")" = "$VOL_CODE" ]; then
  BASE_DIR="$(dirname "$CWD")"
else
  BASE_DIR="$CWD"
fi

HOST_VOL_CODE_DIR="$BASE_DIR/${VOL_CODE}"
HOST_VOL_KEYDB_DIR="$BASE_DIR/${VOL_KEYDB}"

echo "This will remove the following Docker volumes: $VOL_CODE, $VOL_KEYDB"
if [ "$REMOVE_DIRS" = true ]; then
  echo "Host directories to be removed (because --remove-dirs was passed):"
  echo "  $HOST_VOL_CODE_DIR"
  echo "  $HOST_VOL_KEYDB_DIR"
else
  echo "Host directories will NOT be removed (pass --remove-dirs to change)."
fi

if confirm "Proceed with removal of Docker volumes?"; then
  remove_volume "$VOL_CODE"
  remove_volume "$VOL_KEYDB"

  if [ "$REMOVE_DIRS" = true ]; then
    if confirm "Also remove the host directories listed above?"; then
      remove_dir "$HOST_VOL_CODE_DIR"
      remove_dir "$HOST_VOL_KEYDB_DIR"
    else
      echo "Skipping host directories removal."
    fi
  fi

  echo "Uninstall complete."
else
  echo "Aborted by user."
  exit 1
fi
