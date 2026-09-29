# Contributing to OmniFleet

OmniFleet uses **trunk-based development** on `main` with short-lived feature branches.

## Branch naming

- `cursor/<short-description>-<id>` for automated agent branches
- `feat/<issue>-<short-description>` for human contributors

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat(gateway): add websocket live feed`
- `fix(tracking): correct RLS tenant context`
- `chore(ci): path-filter auth service pipeline`

Reference GitHub issues in the footer when applicable: `Refs #123`.

## Pull requests

- Keep PRs focused; prefer vertical slices over mega-changes.
- Update `README.md` roadmap status when service maturity changes.
- Ensure `go test`, `helm lint`, and `terraform validate` pass locally or in CI.
- Do not commit secrets; extend `.env.example` instead.

## Code layout

| Path | Purpose |
|------|---------|
| `proto/` | gRPC contracts (buf) |
| `services/*` | Go microservices |
| `pkg/` | Shared libraries |
| `web/dashboard` | Next.js dispatcher UI |
| `mobile/driver` | Expo driver app (scaffold) |
| `deploy/terraform` | AWS infrastructure modules |
| `deploy/helm` | Kubernetes packaging |

## Preview environments

See [docs/preview-environments.md](docs/preview-environments.md).
