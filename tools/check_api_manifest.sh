#!/usr/bin/env bash
# Validates kentik-api.json (the coverage manifest read by the kentikcommunity
# indexer) and checks that its schema_source.ref matches the api-schema-public
# version pinned in go.mod.
set -euo pipefail

REPO_DIR=$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
MANIFEST="$REPO_DIR/kentik-api.json"

jq -e '
  def nonempty: type == "string" and length > 0;
  (.schema_source.repo == "github.com/kentik/api-schema-public")
  and (.schema_source.ref | nonempty)
  and (.services | type == "array" and length > 0)
  and all(.services[];
        (.service | nonempty)
        and (.version | test("^v[0-9]{6}([a-z]+[0-9]*)?$"))
        and (.stability | IN("stable", "beta", "alpha"))
        and (.status | IN("supported", "partial", "experimental", "deprecated"))
        and (.surface | type == "array" and length > 0 and all(.[]; nonempty)))
  and ((.legacy // []) | type == "array"
        and all(.[]; (.api | nonempty) and (.status | IN("supported", "deprecated", "removed"))))
' "$MANIFEST" >/dev/null || { echo "kentik-api.json: schema check failed" >&2; exit 1; }

ref=$(jq -r '.schema_source.ref' "$MANIFEST")
gomod=$(awk '$1 == "github.com/kentik/api-schema-public" {print $2}' "$REPO_DIR/go.mod")
if [[ -z "$gomod" ]]; then
  echo "kentik-api.json: github.com/kentik/api-schema-public not found in go.mod" >&2
  exit 1
fi
# Accept the exact pseudo-version, or a commit SHA whose 12-char prefix is in it.
if [[ "$ref" != "$gomod" && "$gomod" != *"${ref:0:12}" ]]; then
  echo "kentik-api.json: schema_source.ref ($ref) does not match go.mod ($gomod)" >&2
  exit 1
fi
echo "kentik-api.json OK (api-schema-public $gomod)"
