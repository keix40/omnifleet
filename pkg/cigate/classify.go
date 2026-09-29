package cigate

// Requirements lists CI workflows that must pass for a diff.
type Requirements struct {
	Go    bool
	Web   bool
	Infra bool
	E2E   bool
}

// ClassifyChangedFiles maps changed paths to required workflow groups.
func ClassifyChangedFiles(paths []string) Requirements {
	var req Requirements
	for _, f := range paths {
		if matchGo(f) {
			req.Go = true
		}
		if matchWeb(f) {
			req.Web = true
		}
		if matchInfra(f) {
			req.Infra = true
		}
		if matchE2E(f) {
			req.E2E = true
		}
	}
	return req
}

func matchGo(f string) bool {
	switch {
	case hasPrefix(f, "pkg/"), hasPrefix(f, "gen/"), hasPrefix(f, "proto/"), hasPrefix(f, "services/"), hasPrefix(f, "db/"):
		return true
	case f == "go.work", f == "scripts/bootstrap-db.sh", f == ".github/workflows/go-services.yml":
		return true
	default:
		return false
	}
}

func matchWeb(f string) bool {
	return hasPrefix(f, "web/dashboard/") ||
		hasPrefix(f, "mobile/driver/") ||
		f == ".github/workflows/web-dashboard.yml" ||
		f == ".github/workflows/mobile-driver.yml"
}

func matchInfra(f string) bool {
	return hasPrefix(f, "deploy/terraform/") || hasPrefix(f, "deploy/helm/") || f == ".github/workflows/infra.yml"
}

func matchE2E(f string) bool {
	switch {
	case hasPrefix(f, "pkg/"), hasPrefix(f, "services/"), hasPrefix(f, "db/"), hasPrefix(f, "tests/e2e/"), hasPrefix(f, "scripts/"):
		return true
	case f == "docker-compose.yml", f == ".github/workflows/e2e-compose.yml":
		return true
	default:
		return false
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
