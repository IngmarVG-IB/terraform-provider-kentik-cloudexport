#!/usr/bin/env bash
# Builds the provider and runs `terraform validate` on every example directory,
# using a throwaway CLI config with dev_overrides so nothing is downloaded and
# no API is contacted. Usage: tools/validate_examples.sh [terraform|tofu]
set -euo pipefail

REPO_DIR=$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TF=${1:-${TF_CLI:-terraform}}
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

go build -o "$WORK/bin/terraform-provider-kentik-cloudexport" "$REPO_DIR"

cat > "$WORK/terraformrc" <<RC
provider_installation {
  dev_overrides {
    "registry.terraform.io/kentik/kentik-cloudexport" = "$WORK/bin"
    "registry.opentofu.org/kentik/kentik-cloudexport" = "$WORK/bin"
  }
  direct {}
}
RC
export TF_CLI_CONFIG_FILE="$WORK/terraformrc"
export TF_IN_AUTOMATION=1
# Dummy credentials satisfy the required provider arguments; validate never
# configures the provider, so nothing is sent anywhere.
export KTAPI_AUTH_EMAIL=validate@example.com KTAPI_AUTH_TOKEN=dummy

status=0
while IFS= read -r dir; do
  rel=${dir#"$REPO_DIR"/}
  dest="$WORK/examples/$rel"
  mkdir -p "$dest"
  cp "$dir"/*.tf "$dest"/
  if (cd "$dest" && "$TF" validate -no-color >"$dest/validate.log" 2>&1); then
    echo "ok   $rel"
  else
    echo "FAIL $rel"
    cat "$dest/validate.log"
    status=1
  fi
done < <(find "$REPO_DIR/examples" -name '*.tf' -exec dirname {} \; | sort -u)

exit $status
