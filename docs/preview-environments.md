# PR Preview Environments

OmniFleet preview environments validate the **vertical slice** on each pull request without touching production.

## Goals

- Build only changed services (path-filtered GitHub Actions workflows).
- Deploy ephemeral namespace per PR to the non-production EKS cluster (Terraform module `deploy/terraform/modules/eks`).
- Wire gateway + dashboard preview URLs with JWT secrets from AWS Secrets Manager.

## Proposed flow

1. **Detect changes** — workflows in `.github/workflows/*` map paths to build matrices (`services/auth`, `web/dashboard`, etc.).
2. **Build & push images** — `ghcr.io/keix40/omnifleet/<service>:pr-<number>-<sha>`.
3. **Helm upgrade** — install charts from `deploy/helm/charts/*` into `omnifleet-pr-<number>` namespace with values overrides:
   - `gateway.image.tag=pr-<number>-<sha>`
   - `NEXT_PUBLIC_GATEWAY_URL=https://pr-<number>.preview.omnifleet.example`
4. **Smoke tests** — login as `dispatcher@acme.test`, open WebSocket, assert one position event from CI GPS simulator job.
5. **Teardown** — on PR close, `helm uninstall` namespace and delete CRDs/load balancers.

## Status

Documented approach only in this foundation PR. GitHub Environment protection rules and cloud credentials are required before enabling automatic previews.
