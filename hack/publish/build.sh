#!/usr/bin/env bash
# Reproducible build of the probe-gate fork's operator images into our AR.
#
# Anyone can replicate:
#   git clone https://github.com/recursionpharma/slurm-operator
#   git checkout ce/1.2.3-probe-opt-out
#   ./hack/publish/build.sh
#
# Requirements: go >= 1.26, gcloud (auth to the AR), network access to
# proxy.golang.org and gcr.io.
#
# Reproducibility notes:
#   * Source is this commit; go.sum pins every module version; CGO is off.
#   * The distroless base is pinned by DIGEST below (recorded 2026-09-30).
#   * Remaining non-determinism: Go build IDs vary across Go PATCH versions.
#     Use go 1.26.x for bit-identical binaries. Same base digest + same
#     binaries => same image digest.
set -euo pipefail
cd "$(dirname "$0")/../.."

# ---- pinned inputs -------------------------------------------------------------
BASE_DIGEST="sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3" # gcr.io/distroless/static:nonroot, verified 2026-09-30
REGISTRY="us-central1-docker.pkg.dev/eng-ops-b1bb36c9/container-images"
TAG="1.2.3-probe-gate"
CRANE="go run github.com/google/go-containerregistry/cmd/crane@v0.20.3"

out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT

# ---- auth (crane/go-containerregistry read docker's config via DOCKER_CONFIG) ---
DOCKER_CONFIG_DIR="$(mktemp -d)"
trap 'rm -rf "$out" "$DOCKER_CONFIG_DIR"' EXIT
cat > "${DOCKER_CONFIG_DIR}/config.json" <<EOF
{"auths":{"us-central1-docker.pkg.dev":{"username":"oauth2accesstoken","password":"$(gcloud auth print-access-token)"}}}
EOF
export DOCKER_CONFIG="${DOCKER_CONFIG_DIR}"
echo "authenticated to $REGISTRY"

# ---- binaries (linux/amd64, static; same invocations as the upstream Dockerfile) --
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$out/manager" ./cmd/manager
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$out/webhook" ./cmd/webhook

# ---- publish -----------------------------------------------------------------------
# manager -> rxrx-slurm-operator ; webhook -> rxrx-slurm-operator-webhook
publish() { # <bin> <image-name>
  (
    cd hack/publish
    go run . "$1" "$out/$1" "gcr.io/distroless/static@${BASE_DIGEST}" "${REGISTRY}/$2:${TAG}"
  )
}
publish manager  "rxrx-slurm-operator"
publish webhook  "rxrx-slurm-operator-webhook"

echo
echo "Done. Pin these digests in the Argo values:"
$CRANE digest "${REGISTRY}/rxrx-slurm-operator:${TAG}"
$CRANE digest "${REGISTRY}/rxrx-slurm-operator-webhook:${TAG}"
