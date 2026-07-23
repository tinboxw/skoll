#!/usr/bin/env sh
set -eu

action="${1:-package}"
plugin_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$plugin_dir/../.." && pwd)
dist_dir="${SKOLL_PLUGIN_DIST:-$plugin_dir/dist}"
plugins_root="${SKOLL_DEV_PLUGINS_ROOT:-$plugin_dir/.skoll-dev}"
artifact="$dist_dir/pharma_oa-0.7.0.zip"
checksum="$artifact.sha256"
backend="$plugin_dir/backend/bin/pharma_oa-server"

build_plugin() {
  mkdir -p "$(dirname "$backend")"
  (cd "$plugin_dir/frontend" && npm run build)
  (cd "$plugin_dir" && go build -o "$backend" ./backend)
}

case "$action" in
  build) build_plugin ;;
  package) build_plugin; (cd "$repo_root" && go run ./cmd/skoll-plugin package "$plugin_dir" "$dist_dir") ;;
  verify) (cd "$repo_root" && go run ./cmd/skoll-plugin verify-package "$artifact" "$checksum") ;;
  install) (cd "$repo_root" && go run ./cmd/skoll-plugin install-package "$artifact" "$checksum" "$plugins_root") ;;
  dev) build_plugin; (cd "$repo_root" && go run ./cmd/skoll-plugin dev "$plugin_dir" "$dist_dir" "$plugins_root") ;;
  *) echo "usage: $0 {build|package|verify|install|dev}" >&2; exit 2 ;;
esac
