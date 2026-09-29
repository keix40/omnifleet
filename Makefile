.PHONY: proto test build compose-up compose-down helm-lint tf-validate

proto:
	buf generate

test:
	cd pkg && go test ./...

test-rls:
	DATABASE_URL="$(DATABASE_URL)" cd pkg && go test ./db -run TestRLS -count=1

build:
	for svc in auth gateway tracking geofencing; do \
		( cd services/$$svc && go build -o ../../bin/$$svc ./cmd/$$svc ); \
	done

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down -v

helm-lint:
	helm lint deploy/helm/omnifleet
	for chart in gateway auth tracking geofencing eta dispatch billing notifications; do helm lint deploy/helm/charts/$$chart; done

tf-validate:
	cd deploy/terraform && terraform init -backend=false && terraform validate
