.PHONY: docs

docs:
	dirs="$$(go list -f '{{.Dir}}' ./orm/... | sed 's|$(CURDIR)|docs|')"; \
	for d in $$dirs; do mkdir -p $$d; done
	go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest --output "docs/{{.Dir}}/README.md" ./orm/...
