.PHONY: docs

docs:
	mkdir -p docs
	gomarkdoc --output "docs/{{.Dir}}/README.md" ./orm/...
