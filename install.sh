#!/usr/bin/env bash
set -euo pipefail

# c3po is installed from this checkout, never from a GitHub release.

APP=c3po
INSTALL_DIR="${C3PO_INSTALL_DIR:-$HOME/.local/bin}"
MODULE_PATH="github.com/ryangerardwilson/c3po"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

usage() {
  cat <<EOF
${APP} installer

Usage:
  install.sh                 Install from this checkout (git-stamped)
  install.sh from <path>     Install from a source checkout (git-stamped)
  install.sh help            Show this help

Installs to \$C3PO_INSTALL_DIR or ~/.local/bin.
Links rgw-ast to the same binary, removes xl, cx, and complexity links that point at it, and removes a managed nlp launcher.
Removes ~/.local/bin/notify and installs the ryan.notify omarchy-shell plugin.
EOF
}

die() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "'$1' is required"
}

stamp_vars() {
  local pkg="$1"
  local version="$2"
  local commit="$3"
  local source_hash="$4"
  local build_time="$5"
  printf -- '-X %s.Version=%s -X %s.Commit=%s -X %s.SourceHash=%s -X %s.BuildTime=%s ' \
    "$pkg" "$version" "$pkg" "$commit" "$pkg" "$source_hash" "$pkg" "$build_time"
}

build_ldflags() {
  local version="$1"
  local commit="$2"
  local source_hash="$3"
  local build_time="$4"
  local flags
  flags="$(stamp_vars "${MODULE_PATH}/internal/version" "$version" "$commit" "$source_hash" "$build_time")"
  flags+="$(stamp_vars "${MODULE_PATH}/ast/internal/version" "$version" "$commit" "$source_hash" "$build_time")"
  printf '%s' "-s -w ${flags}"
}

source_version_meta() {
  local source_path="$1"
  local version commit source_hash
  commit="$(git -C "$source_path" rev-parse --short HEAD 2>/dev/null || echo unknown)"
  if [[ -n "$(git -C "$source_path" status --porcelain --untracked-files=normal -- . 2>/dev/null)" ]]; then
    commit="${commit}-dirty"
  fi
  version="$(sed -n 's/^[[:space:]]*Version[[:space:]]*= *"\([^"]*\)".*/\1/p' "$source_path/internal/version/version.go" | head -1)"
  if [[ -z "$version" ]]; then
    version="0.0.0-dev"
  fi
  source_hash="$(
    find "$source_path" -type f -name '*.go' -not -path '*/.git/*' -print0 |
      sort -z |
      xargs -0 sha256sum |
      sha256sum |
      awk '{print substr($1, 1, 16)}'
  )"
  printf '%s %s %s\n' "$version" "$commit" "$source_hash"
}

link_name() {
  local name="$1"
  ln -sfn "$APP" "$INSTALL_DIR/$name"
}

remove_compat_link() {
  local name="$1"
  local path="$INSTALL_DIR/$name"
  local target
  [[ -L "$path" ]] || return 0
  target="$(readlink "$path")"
  if [[ "$target" == "$APP" || "$target" == "$INSTALL_DIR/$APP" ]]; then
    rm -f "$path"
  fi
}

remove_retired_nlp() {
  local path="$INSTALL_DIR/nlp"
  [[ -f "$path" ]] || return 0
  if grep -Fq "Managed by nlp installer" "$path"; then
    rm -f "$path"
  fi
}

remove_retired_notify() {
  local path="$HOME/.local/bin/notify"
  if [[ -L "$path" || -f "$path" ]]; then
    rm -f "$path"
  fi
}

install_notify_plugin() {
  local source_path="$1"
  local dest="${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/ryan.notify"
  mkdir -p "$dest"
  cp "$source_path/notify/plugin/manifest.json" "$source_path/notify/plugin/Overlay.qml" "$dest/"
}

install_from_source() {
  local source_path=$1
  [[ -d "$source_path" ]] || die "source path does not exist: $source_path"
  require_command go
  mkdir -p "$INSTALL_DIR"
  local version commit source_hash build_time ldflags
  read -r version commit source_hash < <(source_version_meta "$source_path")
  build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  ldflags="$(build_ldflags "$version" "$commit" "$source_hash" "$build_time")"
  (cd "$source_path" && GOWORK=off CGO_ENABLED=0 go build -ldflags "$ldflags" -o "$INSTALL_DIR/$APP" ./cmd/c3po)
  chmod 755 "$INSTALL_DIR/$APP"
  link_name rgw-ast
  remove_compat_link xl
  remove_compat_link cx
  remove_compat_link complexity
  remove_retired_nlp
  remove_retired_notify
  install_notify_plugin "$source_path"
  "$INSTALL_DIR/$APP" version
  write_snapshot_units
  activate_snapshot_timer
}

write_snapshot_units() {
  local dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
  mkdir -p "$dir"
  cat > "$dir/c3po-snapshot.service" <<'EOF'
[Unit]
Description=Save the c3po database backup

[Service]
Type=oneshot
ExecStart=%h/.local/bin/c3po __snapshot
EOF
  cat > "$dir/c3po-snapshot.timer" <<'EOF'
[Unit]
Description=Periodically save the c3po database backup

[Timer]
OnActiveSec=10min
OnUnitActiveSec=10min
AccuracySec=5s
Unit=c3po-snapshot.service

[Install]
WantedBy=timers.target
EOF
}

activate_snapshot_timer() {
  [[ "${C3PO_INSTALL_ACTIVATE_SYSTEMD:-1}" == "1" ]] || return 0
  command -v systemctl >/dev/null 2>&1 || return 0
  systemctl --user daemon-reload >/dev/null 2>&1 || return 0
  systemctl --user enable c3po-snapshot.timer >/dev/null 2>&1 || true
  systemctl --user start c3po-snapshot.timer >/dev/null 2>&1 || true
}

case "${1:-install}" in
  install)
    install_from_source "$SCRIPT_DIR"
    ;;
  from)
    [[ -n "${2:-}" ]] || die "from requires a path"
    install_from_source "$2"
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    die "unknown command: $1 (c3po installs from source; see install.sh help)"
    ;;
esac
