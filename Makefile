.PHONY: docs

docs:
	dirs="$$(go list -f '{{.Dir}}' ./orm/... | sed 's|$(CURDIR)|docs|')"; \
	for d in $$dirs; do mkdir -p $$d; done
	gomarkdoc --output "docs/{{.Dir}}/README.md" ./orm/...
