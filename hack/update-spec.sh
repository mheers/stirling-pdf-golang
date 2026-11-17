#!/usr/bin/env bash
# Fetch the current Stirling PDF Processing API document and regenerate the SDK.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

url="https://registry.scalar.com/@stirlingpdf/apis/stirling-pdf-processing-api/"
echo "Fetching $url"
curl --fail --silent --show-error --location "$url" --output api/openapi.json

# Sanity check the document before generating code from it.
python3 - <<'PY'
import json
with open("api/openapi.json", "rb") as handle:
    document = json.load(handle)
assert document.get("openapi", "").startswith("3."), "unexpected OpenAPI version"
print("OpenAPI", document["openapi"], "version", document["info"]["version"],
      "with", len(document["paths"]), "paths")
PY

go generate ./api ./internal/ops
gofmt -w api internal/ops
echo "Regenerated api and internal/ops. Review the diff, then run: go test ./..."
