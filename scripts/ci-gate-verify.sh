#!/usr/bin/env bash
# Verifies path-filtered workflows for this PR: required workflows must have
# succeeded on HEAD_SHA; workflows whose paths did not change are not required.
set -euo pipefail

: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY required}"
: "${HEAD_SHA:?HEAD_SHA required}"
: "${BASE_SHA:?BASE_SHA required}"

mapfile -t CHANGED < <(git diff --name-only "$BASE_SHA" "$HEAD_SHA" || true)

needs_go=0
needs_web=0
needs_infra=0
needs_e2e=0
needs_smoke=0
needs_neon=0

for f in "${CHANGED[@]}"; do
  case "$f" in
    pkg/*|gen/*|proto/*|services/*|db/*|go.work|cmd/omnifleet-all/*|scripts/bootstrap-db.sh|scripts/bootstrap-neon.sh|.github/workflows/go-services.yml)
      needs_go=1
      ;;
  esac
  case "$f" in
    db/*|scripts/bootstrap-neon.sh|scripts/seed-demo-users.sh|scripts/seedhash/*|pkg/auth/demo_seed.go|.github/workflows/neon-bootstrap.yml)
      needs_neon=1
      ;;
  esac
  case "$f" in
    cmd/omnifleet-all/*|services/*|pkg/*|db/*|scripts/bootstrap-db.sh|scripts/bootstrap-neon.sh|.github/workflows/smoke-allinone.yml)
      needs_smoke=1
      ;;
  esac
  case "$f" in
    web/dashboard/*|.github/workflows/web-dashboard.yml)
      needs_web=1
      ;;
  esac
  case "$f" in
    mobile/driver/*|.github/workflows/mobile-driver.yml)
      needs_web=1
      ;;
  esac
  case "$f" in
    deploy/terraform/*|deploy/helm/*|Dockerfile.render|render.yaml|.github/workflows/infra.yml)
      needs_infra=1
      ;;
  esac
  case "$f" in
    pkg/*|services/*|db/*|docker-compose.yml|tests/e2e/*|scripts/*|.github/workflows/e2e-compose.yml)
      needs_e2e=1
      ;;
  esac
done

echo "Changed files: ${#CHANGED[@]}"
echo "Required workflows: go=$needs_go web=$needs_web infra=$needs_infra e2e=$needs_e2e smoke=$needs_smoke neon=$needs_neon"

if (( needs_go == 0 && needs_web == 0 && needs_infra == 0 && needs_e2e == 0 && needs_smoke == 0 && needs_neon == 0 )); then
  echo "No path-filtered workflows apply to this diff; gate passes."
  exit 0
fi

wait_for_workflow() {
  local workflow_file=$1
  local label=$2
  echo "--- Waiting for ${label} (${workflow_file}) on commit ${HEAD_SHA} ---"
  local run_id=""
  local wait_start=$SECONDS
  local max_wait_empty=900 # 15m for GitHub to enqueue path-filtered runs

  while [[ -z "$run_id" ]]; do
    if (( SECONDS - wait_start > max_wait_empty )); then
      echo "Timed out waiting for ${label} to start"
      return 1
    fi
    run_id=$(gh run list \
      --repo "$GITHUB_REPOSITORY" \
      --workflow="$workflow_file" \
      --commit="$HEAD_SHA" \
      --limit 1 \
      --json databaseId \
      --jq '.[0].databaseId // empty' 2>/dev/null || true)
    if [[ -z "$run_id" || "$run_id" == "null" ]]; then
      run_id=""
      sleep 15
    fi
  done

  gh run watch "$run_id" --repo "$GITHUB_REPOSITORY" --exit-status
}

(( needs_go == 1 )) && wait_for_workflow "go-services.yml" "Go Services"
(( needs_web == 1 )) && wait_for_workflow "web-dashboard.yml" "Web Dashboard"
(( needs_infra == 1 )) && wait_for_workflow "infra.yml" "Infrastructure"
(( needs_e2e == 1 )) && wait_for_workflow "e2e-compose.yml" "E2E Compose Slice"
(( needs_smoke == 1 )) && wait_for_workflow "smoke-allinone.yml" "Smoke All-in-One"
(( needs_neon == 1 )) && wait_for_workflow "neon-bootstrap.yml" "Neon Bootstrap"

echo "All required workflows passed."
