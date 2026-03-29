.PHONY: docs

docs:
	packages="$$(go list ./orm/... | grep -v '/internal/')"; \
	dirs="$$(go list -f '{{.Dir}}' $$packages | sed 's|$(CURDIR)|docs|')"; \
	for d in $$dirs; do mkdir -p $$d; done; \
	go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest --output "docs/{{.Dir}}/README.md" $$packages
